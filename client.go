package ring

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	batchSize         = 100
	flushInterval     = 100 * time.Millisecond
	reconnectInterval = 5 * time.Second
)

type Client struct {
	address string
	timeout time.Duration
	queue   chan []byte
	done    chan struct{}
	config  *tls.Config
	wg      sync.WaitGroup
	cache   *logCache
}

type clientOptions struct {
	cacheDir     string
	maxMemCache  int64
	maxFileCache int64
}

// Option настраивает параметры клиента.
type Option func(*clientOptions)

// WithCacheDir задаёт директорию для файлового кеша логов
// (по умолчанию os.TempDir()/ring-log-cache).
func WithCacheDir(dir string) Option {
	return func(o *clientOptions) {
		o.cacheDir = dir
	}
}

// WithMaxMemoryCache задаёт лимит кеша логов в памяти в байтах
// (по умолчанию 20 Мб).
func WithMaxMemoryCache(bytes int64) Option {
	return func(o *clientOptions) {
		o.maxMemCache = bytes
	}
}

// WithMaxFileCache задаёт лимит файлового кеша логов в байтах
// (по умолчанию 50 Мб).
func WithMaxFileCache(bytes int64) Option {
	return func(o *clientOptions) {
		o.maxFileCache = bytes
	}
}

func NewClient(address string, config *tls.Config, opts ...Option) *Client {
	options := clientOptions{
		maxMemCache:  DefaultMaxMemoryCache,
		maxFileCache: DefaultMaxFileCache,
	}
	for _, opt := range opts {
		opt(&options)
	}
	cache, err := newLogCache(options.cacheDir, options.maxMemCache, options.maxFileCache, address)
	if err != nil {
		fmt.Printf("Log cache disabled: %v\n", err)
	}
	return &Client{
		address: address,
		config:  config,
		timeout: 5 * time.Second,
		queue:   make(chan []byte, 1000),
		done:    make(chan struct{}),
		cache:   cache,
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

// CacheStats возвращает текущее состояние кеша неотправленных логов.
func (c *Client) CacheStats() CacheStats {
	if c.cache == nil {
		return CacheStats{}
	}
	return c.cache.stats()
}

func (c *Client) processQueue() {
	defer c.wg.Done()
	var conn *tls.Conn
	var reader *bufio.Reader

	closeConn := func() {
		if conn != nil {
			_ = conn.Close()
			conn = nil
		}
		reader = nil
		if c.cache != nil {
			c.cache.closeReplay()
		}
	}

	tryConnect := func() {
		if conn != nil {
			return
		}
		dialer := &net.Dialer{Timeout: c.timeout}
		nc, err := tls.DialWithDialer(dialer, "tcp", c.address, c.config)
		if err != nil {
			return
		}
		conn = nc
		reader = bufio.NewReaderSize(conn, 64*1024)
		fmt.Printf("Connected to log server %s\n", c.address)
	}

	tryConnect()
	if conn == nil {
		fmt.Printf("Log server %s is unavailable, logs will be cached\n", c.address)
	}

	batch := make([][]byte, 0, batchSize)
	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()
	reconnectTicker := time.NewTicker(reconnectInterval)
	defer reconnectTicker.Stop()

	cacheEntries := func(entries ...[]byte) {
		if c.cache == nil {
			return
		}
		for _, e := range entries {
			c.cache.add(e)
		}
	}

	// sendBatch отправляет пачку и ждёт подтверждение ("done") от сервера.
	// Без подтверждения пачка считается недоставленной, а соединение
	// закрывается для переподключения.
	sendBatch := func(entries [][]byte) bool {
		if conn == nil || len(entries) == 0 {
			return false
		}
		if err := c.sendRaw(conn, buildBatch(entries)); err != nil {
			fmt.Printf("Failed to send batch: %v, reconnecting...\n", err)
			closeConn()
			return false
		}
		if err := conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
			closeConn()
			return false
		}
		ack, err := reader.ReadString('\n')
		if err != nil || ack != "done\n" {
			if err != nil {
				fmt.Printf("Failed to receive ack: %v, reconnecting...\n", err)
			}
			closeConn()
			return false
		}
		return true
	}

	// flushFresh отправляет накопленную свежую пачку,
	// при отсутствии связи или ошибке — кеширует её.
	flushFresh := func() {
		if len(batch) == 0 {
			return
		}
		if !sendBatch(batch) {
			cacheEntries(batch...)
		}
		batch = batch[:0]
	}

	// replayCache отправляет одну пачку из кеша. Отправка идёт по тику,
	// пачками, чтобы не перегружать сервер после восстановления связи.
	replayCache := func() {
		if c.cache == nil || !c.cache.hasPending() {
			return
		}
		entries := c.cache.nextBatch(batchSize)
		if len(entries) == 0 {
			return
		}
		if !sendBatch(entries) {
			c.cache.restore(entries)
		}
	}

	for {
		select {
		case <-c.done:
			flushFresh()
			// дочитываем остаток очереди в кеш и сбрасываем кеш в файл,
			// чтобы логи не пропали при следующем запуске
			for {
				select {
				case data := <-c.queue:
					cacheEntries(data)
				default:
					if c.cache != nil {
						c.cache.flushAll()
					}
					closeConn()
					return
				}
			}
		case data := <-c.queue:
			if conn == nil {
				cacheEntries(data)
				continue
			}
			batch = append(batch, data)
			if len(batch) >= batchSize {
				flushFresh()
			}
		case <-flushTicker.C:
			if conn == nil {
				continue
			}
			flushFresh()
			replayCache()
		case <-reconnectTicker.C:
			if conn == nil {
				tryConnect()
			}
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
