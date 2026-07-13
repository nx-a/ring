package tcp

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/nx-a/ring"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	appctx "github.com/nx-a/ring/internal/engine/context"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	conn         net.Conn
	reader       *bufio.Reader
	writer       *bufio.Writer
	done         chan bool
	isClosed     atomic.Bool
	dataService  ports.DataService
	tokenService ports.TokenService
}

func NewClient(conn net.Conn, service ports.DataService, tokenService ports.TokenService) *Client {
	return &Client{
		conn:         conn,
		reader:       bufio.NewReader(conn),
		writer:       bufio.NewWriter(conn),
		done:         make(chan bool),
		dataService:  service,
		tokenService: tokenService,
	}
}

func (c *Client) Run() {
	c.ReadLoop()
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
			if errors.Is(err, io.EOF) {
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
	return c.isClosed.Load()
}

func (c *Client) HandleMessage(message string) {
	addr := c.conn.RemoteAddr()
	log.Debugf("Received from %s message: %s", addr, message)

	bufData, err := c.decode(message)
	if err != nil {
		log.WithError(err).Warnf("Client %s: decode failed", addr)
		return
	}

	var entries []ring.LogEntry
	if err := json.Unmarshal(bufData, &entries); err == nil {
		for i := range entries {
			c.writeEntry(&entries[i])
		}
	} else {
		var entry ring.LogEntry
		if err := json.Unmarshal(bufData, &entry); err != nil {
			log.WithError(err).Warnf("Client %s: unmarshal failed", addr)
			return
		}
		c.writeEntry(&entry)
	}

	if err := c.SendMessage("done\n"); err != nil {
		log.WithError(err).Warnf("send ack failed for %s", addr)
		c.Close()
	}
}

func (c *Client) decode(message string) ([]byte, error) {
	rawJSON, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	reader, err := zlib.NewReader(bytes.NewReader(rawJSON))
	if err != nil {
		return nil, fmt.Errorf("zlib reader: %w", err)
	}
	defer reader.Close()
	var bufData bytes.Buffer
	if _, err := io.Copy(&bufData, reader); err != nil {
		return nil, fmt.Errorf("zlib decompress: %w", err)
	}
	return bufData.Bytes(), nil
}

func (c *Client) writeEntry(entry *ring.LogEntry) {
	claim, err := c.tokenService.GetByToken(entry.Token)
	if err != nil {
		log.WithError(err).Warnf("Client %s: get token failed", c.conn.RemoteAddr())
		return
	}
	_time, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		_time = time.Now()
	}
	if entry.Fields == nil {
		entry.Fields = make(map[string]interface{})
	}
	entry.Fields["message"] = entry.Message
	if entry.File != "" {
		entry.Fields["file"] = entry.File
	}
	val, err := json.Marshal(entry.Fields)
	if err != nil {
		log.WithError(err).Warn("marshal fields failed")
		return
	}
	if err := c.dataService.Write(appctx.WithControl(context.Background(), claim), &domain.Data{
		Ext:   entry.AppName,
		Time:  &_time,
		Level: entry.Level,
		Val:   val,
	}); err != nil {
		log.WithError(err).Warn("write data failed")
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
	if c.isClosed.CompareAndSwap(false, true) {
		if err := c.conn.Close(); err != nil {
			log.Errorf("close connection error: %v", err)
		}
		close(c.done)
	}
}
