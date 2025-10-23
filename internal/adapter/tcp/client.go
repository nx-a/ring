package tcp

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nx-a/ring/hook"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type Client struct {
	conn         net.Conn
	reader       *bufio.Reader
	writer       *bufio.Writer
	done         chan bool
	isClosed     bool
	mutex        sync.RWMutex
	dataService  ports.DataService
	tokenService ports.TokenService
}

func NewClient(conn net.Conn, service ports.DataService, tokenService ports.TokenService) *Client {
	return &Client{
		conn:         conn,
		reader:       bufio.NewReader(conn),
		writer:       bufio.NewWriter(conn),
		done:         make(chan bool),
		isClosed:     false,
		dataService:  service,
		tokenService: tokenService,
	}
}

func (c *Client) Run() {
	// Запускаем heartbeat
	go c.Heartbeat()

	// Основной цикл чтения
	c.ReadLoop()
}
func (c *Client) Heartbeat() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.IsClosed() {
				return
			}
			if err := c.SendMessage("check\n"); err != nil {
				log.Infof("Heartbeat failed for %s: %v", c.conn.RemoteAddr(), err)
				c.Close()
				return
			}
		case <-c.done:
			return
		}
	}
}
func (c *Client) ReadLoop() {
	defer c.Close()

	for {
		if c.IsClosed() {
			return
		}

		c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		message, err := c.reader.ReadString('\n')
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			if err.Error() == "EOF" {
				log.Infof("Client %s: connection closed by client", c.conn.RemoteAddr())
			} else {
				log.Infof("Client %s: read error: %v", c.conn.RemoteAddr(), err)
			}
			return
		}

		c.HandleMessage(message)
	}
}
func (c *Client) IsClosed() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isClosed
}
func (c *Client) HandleMessage(message string) {
	addr := c.conn.RemoteAddr()
	fmt.Printf("Received from %s", addr)
	rawJson, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		log.Infof("Client %s: decode message failed: %v", addr, err)
	}
	var entry hook.LogEntry
	err = json.Unmarshal(rawJson, &entry)
	if err != nil {
		log.Infof("Client %s: decode message failed: %v", addr, err)
	}
	claim, err := c.tokenService.GetByToken(entry.Token)
	if err != nil {
		log.Infof("Client %s: get token failed: %v", addr, err)
	}
	_time, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		_time = time.Now()
	}
	if entry.Fields == nil {
		entry.Fields = make(map[string]interface{})
	}
	entry.Fields["message"] = entry.Message
	val, err := json.Marshal(entry.Fields)
	c.dataService.Write(context.WithValue(context.Background(), "control", claim), domain.Data{
		Ext:   entry.AppName,
		Time:  &_time,
		Level: entry.Level,
		Val:   val,
	})
	if err = c.SendMessage("done\n"); err != nil {
		log.Printf("Heartbeat failed for %s: %v", addr, err)
		c.Close()
	}
}
func (c *Client) SendMessage(message string) error {
	if c.IsClosed() {
		return fmt.Errorf("connection closed")
	}

	c.conn.SetWriteDeadline(time.Now().Add(1 * time.Second))

	if _, err := c.writer.WriteString(message); err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *Client) Close() {
	if !c.isClosed {
		c.isClosed = true
		err := c.conn.Close()
		if err != nil {
			log.Errorf("close connection error: %v", err)
		}
		close(c.done)
	}
}
