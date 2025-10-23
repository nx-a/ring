package tcp

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	listener net.Listener
	clients  map[*Client]bool
	mutex    sync.RWMutex
	wg       sync.WaitGroup
	shutdown chan struct{}
}

func NewServer(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		listener: listener,
		clients:  make(map[*Client]bool),
		shutdown: make(chan struct{}),
	}, nil
}
func (s *Server) AddClient(client *Client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clients[client] = true
}
func (s *Server) RemoveClient(client *Client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.clients, client)
}
func (s *Server) CloseAllClients() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for client := range s.clients {
		client.Close()
	}
}
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	client := NewClient(conn)
	s.AddClient(client)
	defer s.RemoveClient(client)

	log.Infof("New client connected: %s (total: %d)", conn.RemoteAddr(), len(s.clients))

	client.Run()

	log.Infof("Client disconnected: %s (remaining: %d)", conn.RemoteAddr(), len(s.clients))
}
func (s *Server) Run(ctx context.Context) {
	log.Info("Server started on", s.listener.Addr())
	go s.handleSignals(ctx)

	for {
		s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := s.listener.Accept()
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			if !isClosedError(err) {
				log.Errorf("Accept error: %v", err)
			}
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}
func (s *Server) handleSignals(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-ctx.Done()
	close(s.shutdown)

	// Даем время на завершение активных соединений
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		s.wg.Wait()
		cancel()
	}()

	<-ctx.Done()
	s.CloseAllClients()
	s.listener.Close()
}
func isClosedError(err error) bool {
	return err.Error() == "use of closed network connection"
}
