package ring

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
	"time"
)

type Client struct {
	serverAddr  string
	tlsConfig   *tls.Config
	quicConfig  *quic.Config
	conn        *quic.Conn
	queue       chan []byte
	done        chan struct{}
	isConnected bool
}

func New(addr string) *Client {
	return &Client{
		serverAddr: addr,
		tlsConfig: &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"ring-quic"},
		},
		quicConfig: &quic.Config{
			KeepAlivePeriod: 50 * time.Minute,
			MaxIdleTimeout:  60 * time.Minute,
		},
		queue:       make(chan []byte, 1000), // Буферизованная очередь
		done:        make(chan struct{}),
		isConnected: false,
	}
}
func (c *Client) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Minute)
	defer cancel()
	conn, err := quic.DialAddr(ctx, c.serverAddr, c.tlsConfig, c.quicConfig)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	c.conn = conn
	c.isConnected = true
	log.Printf("Connected to %s", c.serverAddr)
	go c.connectionMonitor()
	return nil
}
func (c *Client) connectionMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if !c.isConnected {
			return
		}

		if _, err := c.SendRequest([]byte("")); err != nil {
			log.Printf("Connection lost: %v", err)
			c.isConnected = false

			// Пытаемся переподключиться
			go c.reconnect()
			return
		}
	}
}
func (c *Client) reconnect() {
	for {
		log.Printf("Attempting to reconnect to %s...", c.serverAddr)

		if err := c.Connect(); err == nil {
			log.Printf("Reconnected successfully")
			return
		}

		time.Sleep(5 * time.Second)
	}
}
func (c *Client) Start() error {
	err := c.Connect()
	if err != nil {
		return err
	}
	go c.processQueue()
	return nil
}
func (c *Client) Stop() {
	close(c.done)
}
func (c *Client) Send(data []byte) error {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(buf, data)
	select {
	case c.queue <- buf:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("queue is full")
	case <-c.done:
		return fmt.Errorf("client is stopped")
	}
}
func (c *Client) processQueue() {
	var conn = c.conn
	for {
		select {
		case <-c.done:
			return

		case data := <-c.queue:
			if c.conn == nil {
				c.reconnect()
			}
			if conn != nil {
				done, err := c.SendRequest(data)
				if err != nil {
					log.Printf("Connection lost: %v", err)
				}
				log.Printf(done)
			}
		}
	}
}
func (c *Client) SendRequest(message []byte) (string, error) {
	message = append(message, '\n')
	if !c.isConnected {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := c.conn.OpenStreamSync(ctx)
	if err != nil {
		c.isConnected = false
		return "", fmt.Errorf("failed to open stream: %v", err)
	}
	defer stream.Close()

	stream.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if _, err := stream.Write(message); err != nil {
		c.isConnected = false
		return "", fmt.Errorf("write failed: %v", err)
	}

	stream.SetReadDeadline(time.Now().Add(3 * time.Second))
	buffer := make([]byte, 4096)
	n, err := stream.Read(buffer)
	if err != nil {
		c.isConnected = false
		return "", fmt.Errorf("read failed: %v", err)
	}

	return string(buffer[:n]), nil
}
