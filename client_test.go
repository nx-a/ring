package ring

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// logServer — тестовый TLS-сервер, принимающий пачки логов.
type logServer struct {
	listener net.Listener
	mu       sync.Mutex
	entries  []map[string]any
	conns    map[net.Conn]struct{}
	wg       sync.WaitGroup
}

func newLogServer(t *testing.T, addr string) *logServer {
	t.Helper()
	cert, err := tls.LoadX509KeyPair("cmd/web/tls/server.crt", "cmd/web/tls/server.key")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := tls.Listen("tcp", addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := &logServer{listener: listener, conns: make(map[net.Conn]struct{})}
	s.wg.Add(1)
	go s.acceptLoop()
	t.Cleanup(s.close)
	return s
}

// close закрывает слушатель и все активные соединения.
func (s *logServer) close() {
	_ = s.listener.Close()
	s.mu.Lock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *logServer) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()
		s.wg.Add(1)
		go s.handle(conn)
	}
}

func (s *logServer) handle(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		_ = conn.Close()
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		raw, err := base64.StdEncoding.DecodeString(message[:len(message)-1])
		if err != nil {
			return
		}
		zr, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			return
		}
		data, err := io.ReadAll(zr)
		_ = zr.Close()
		if err != nil {
			return
		}
		var entries []map[string]any
		if err := json.Unmarshal(data, &entries); err != nil {
			return
		}
		s.mu.Lock()
		s.entries = append(s.entries, entries...)
		s.mu.Unlock()
		_, _ = conn.Write([]byte("done\n"))
	}
}

func (s *logServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *logServer) waitFor(t *testing.T, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.count() >= want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server received %d entries, expected at least %d", s.count(), want)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func newTestClient(t *testing.T, addr string) *Client {
	t.Helper()
	c := NewClient(addr, &tls.Config{InsecureSkipVerify: true},
		WithCacheDir(t.TempDir()),
		// маленький лимит памяти, чтобы часть логов ушла в spill-файл
		WithMaxMemoryCache(512),
		WithMaxFileCache(1<<20),
	)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Stop)
	return c
}

// TestClientCachesWhileDisconnectedAndReplays проверяет, что при отсутствии
// связи логи кешируются (память + файл), а при появлении сервера
// отправляются пачками в исходном порядке.
func TestClientCachesWhileDisconnectedAndReplays(t *testing.T) {
	addr := freeAddr(t)
	client := newTestClient(t, addr)

	const total = 30
	for i := 0; i < total; i++ {
		data := []byte(fmt.Sprintf(`{"n":%d,"message":"m%d"}`, i, i))
		if err := client.Send(data); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	// даём время переместить логи из очереди в кеш
	time.Sleep(300 * time.Millisecond)
	st := client.CacheStats()
	if st.MemEntries == 0 && st.FileBytes == 0 {
		t.Fatal("expected logs to be cached while server is down")
	}
	if st.FileBytes == 0 {
		t.Fatal("expected memory cache overflow to spill file")
	}

	// поднимаем сервер — клиент должен переподключиться (раз в 5 секунд)
	// и отправить накопленные логи
	server := newLogServer(t, addr)
	server.waitFor(t, total, 15*time.Second)

	// свежие логи после восстановления связи
	for i := total; i < total+5; i++ {
		data := []byte(fmt.Sprintf(`{"n":%d,"message":"m%d"}`, i, i))
		if err := client.Send(data); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	server.waitFor(t, total+5, 5*time.Second)

	server.mu.Lock()
	defer server.mu.Unlock()
	for i, e := range server.entries {
		n, ok := e["n"].(float64)
		if !ok || int(n) != i {
			t.Fatalf("entry %d out of order: %v", i, e)
		}
	}
	if client.cache.hasPending() {
		t.Fatal("cache must be empty after successful replay")
	}
}

// TestClientSurvivesServerRestart проверяет, что при разрыве соединения
// логи продолжают кешироваться и будут доставлены после перезапуска сервера.
func TestClientSurvivesServerRestart(t *testing.T) {
	addr := freeAddr(t)
	server := newLogServer(t, addr)
	client := newTestClient(t, addr)

	for i := 0; i < 5; i++ {
		if err := client.Send([]byte(fmt.Sprintf(`{"n":%d}`, i))); err != nil {
			t.Fatal(err)
		}
	}
	server.waitFor(t, 5, 5*time.Second)

	// сервер падает
	server.close()

	for i := 5; i < 15; i++ {
		if err := client.Send([]byte(fmt.Sprintf(`{"n":%d}`, i))); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(300 * time.Millisecond)
	if st := client.CacheStats(); st.MemEntries == 0 && st.FileBytes == 0 {
		t.Fatal("expected logs to be cached after server shutdown")
	}

	// сервер поднимается снова — кеш должен уйти пачками
	server2 := newLogServer(t, addr)
	server2.waitFor(t, 10, 15*time.Second)

	server2.mu.Lock()
	defer server2.mu.Unlock()
	for i, e := range server2.entries {
		n, ok := e["n"].(float64)
		if !ok || int(n) != i+5 {
			t.Fatalf("entry %d out of order: %v", i, e)
		}
	}
}
