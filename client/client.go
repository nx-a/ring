package client

import (
	"encoding/base64"
	"fmt"
	"net"
	"time"
)

type RingClient struct {
	address string
	timeout time.Duration
	queue   chan []byte
	done    chan struct{}
}

func New(address string, timeout time.Duration) *RingClient {
	return &RingClient{
		address: address,
		timeout: timeout,
		queue:   make(chan []byte, 1000), // Буферизованная очередь
		done:    make(chan struct{}),
	}
}
func (c *RingClient) Start() error {
	go c.processQueue()
	return nil
}
func (c *RingClient) Stop() {
	close(c.done)
}
func (c *RingClient) Send(data []byte) error {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(buf, data)
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
func (c *RingClient) processQueue() {
	var conn net.Conn
	var err error

	reconnect := func() {
		if conn != nil {
			conn.Close()
			conn = nil
		}

		for {
			select {
			case <-c.done:
				return
			default:
			}

			conn, err = net.DialTimeout("tcp", c.address, c.timeout)
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
				conn.Close()
			}
			return

		case data := <-c.queue:
			if conn == nil {
				reconnect()
			}

			// Пытаемся отправить данные
			if conn != nil {
				conn.SetWriteDeadline(time.Now().Add(c.timeout))
				_, err := conn.Write(data)
				if err != nil {
					fmt.Printf("Failed to send log: %v, reconnecting...\n", err)
					reconnect()
					// Пытаемся отправить снова после реконнекта
					if conn != nil {
						conn.SetWriteDeadline(time.Now().Add(c.timeout))
						conn.Write(data)
					}
				}
			}
		}
	}
}
