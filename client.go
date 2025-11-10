package ring

import (
	"bytes"
	"compress/zlib"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"time"
)

type Client struct {
	address string
	timeout time.Duration
	queue   chan []byte
	done    chan struct{}
	config  *tls.Config
}

func NewClient(address string, config *tls.Config) *Client {
	return &Client{
		address: address,
		config:  config,
		timeout: 5 * time.Second,
		queue:   make(chan []byte, 1000), // Буферизованная очередь
		done:    make(chan struct{}),
	}
}
func (c *Client) Start() error {
	go c.processQueue()
	return nil
}
func (c *Client) Stop() {
	close(c.done)
}
func (c *Client) Send(data []byte) error {
	var bufData bytes.Buffer
	writer, err := zlib.NewWriterLevel(&bufData, zlib.DefaultCompression)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	buf := make([]byte, base64.StdEncoding.EncodedLen(bufData.Len())+1)
	base64.StdEncoding.Encode(bufData.Bytes(), buf)
	buf = append(buf, '\n')
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
	var conn *tls.Conn
	var err error

	reconnect := func() {
		if conn != nil {
			err = conn.Close()
			if err != nil {
				fmt.Println("Error closing TLS connection: ", err.Error())
			}
			conn = nil
		}

		for {
			select {
			case <-c.done:
				fmt.Println("client is stopped")
				return
			default:
			}

			conn, err = tls.Dial("tcp", c.address, c.config)
			if err == nil {
				fmt.Printf("Connected to log server %s\n", c.address)
				break
			}

			fmt.Printf("Failed to connect to log server: %v, retrying in 5s...\n", err)
			time.Sleep(5 * time.Second)
		}
	}
	reconnect()
	for {
		select {
		case <-c.done:
			if conn != nil {
				err = conn.Close()
				if err != nil {
					fmt.Println("Error closing TLS connection: ", err.Error())
				}
			}
			return

		case data := <-c.queue:
			if conn == nil {
				reconnect()
			}

			if conn != nil {
				err = conn.SetWriteDeadline(time.Now().Add(c.timeout))
				if err != nil {
					fmt.Println("Error setting write deadline: ", err.Error())
				}

				_, err := conn.Write(data)
				if err != nil {
					fmt.Printf("Failed to send log: %v, reconnecting...\n", err)
					reconnect()
					if conn != nil {
						err = conn.SetWriteDeadline(time.Now().Add(c.timeout))
						if err != nil {
							fmt.Println("Error setting write deadline: ", err.Error())
						}
						_, err := conn.Write(data)
						if err != nil {
							fmt.Printf("Failed to send log: %v, reconnecting...\n", err)
						}
					}
				}
			}
		}
	}
}
