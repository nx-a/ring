package ring

import (
	"bytes"
	"compress/zlib"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type Client struct {
	address string
	timeout time.Duration
	queue   chan []byte
	done    chan struct{}
	config  *tls.Config
	wg      sync.WaitGroup
}

func NewClient(address string, config *tls.Config) *Client {
	return &Client{
		address: address,
		config:  config,
		timeout: 5 * time.Second,
		queue:   make(chan []byte, 1000),
		done:    make(chan struct{}),
	}
}

func (c *Client) Start() error {
	c.wg.Add(1)
	go c.processQueue()
	return nil
}

func (c *Client) Stop() {
	close(c.done)
	c.wg.Wait()
}

func (c *Client) Send(data []byte) error {
	select {
	case c.queue <- data:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("queue is full")
	case <-c.done:
		return fmt.Errorf("client is stopped")
	}
}

func (c *Client) processQueue() {
	defer c.wg.Done()
	var conn *tls.Conn

	reconnect := func() {
		if conn != nil {
			_ = conn.Close()
			conn = nil
		}
		for {
			select {
			case <-c.done:
				return
			default:
			}
			var err error
			conn, err = tls.Dial("tcp", c.address, c.config)
			if err == nil {
				fmt.Printf("Connected to log server %s\n", c.address)
				return
			}
			fmt.Printf("Failed to connect to log server: %s error: %v, retrying in 5s...\n", c.address, err)
			select {
			case <-c.done:
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
	reconnect()

	batch := make([][]byte, 0, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 || conn == nil {
			return
		}
		data := buildBatch(batch)
		if err := c.sendRaw(conn, data); err != nil {
			fmt.Printf("Failed to send batch: %v, reconnecting...\n", err)
			reconnect()
			if conn != nil {
				if err2 := c.sendRaw(conn, data); err2 != nil {
					fmt.Printf("Failed to send batch after reconnect: %v\n", err2)
				}
			}
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-c.done:
			flush()
			if conn != nil {
				_ = conn.Close()
			}
			return
		case data := <-c.queue:
			batch = append(batch, data)
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func buildBatch(batch [][]byte) []byte {
	if len(batch) == 0 {
		return nil
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, b := range batch {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(b)
	}
	buf.WriteByte(']')
	return buf.Bytes()
}

func (c *Client) sendRaw(conn *tls.Conn, data []byte) error {
	var bufData bytes.Buffer
	writer, err := zlib.NewWriterLevel(&bufData, zlib.DefaultCompression)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	compressed := bufData.Bytes()
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(compressed)))
	base64.StdEncoding.Encode(buf, compressed)
	buf = append(buf, '\n')

	if err := conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err = conn.Write(buf)
	return err
}
