package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/nx-a/ring/internal/core/ports"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type Server struct {
	listener     net.Listener
	clients      map[*Client]bool
	mutex        sync.RWMutex
	wg           sync.WaitGroup
	shutdown     chan struct{}
	dataService  ports.DataService
	tokenService ports.TokenService
}

func NewServer(addr string, cfg *tls.Config) (*Server, error) {
	listener, err := tls.Listen("tcp", addr, cfg)
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

	tlsConn, ok := conn.(*tls.Conn)
	if ok {
		// Выполняем handshake
		if err := tlsConn.Handshake(); err != nil {
			log.Printf("TLS handshake failed: %v", err)
			return
		}

		state := tlsConn.ConnectionState()
		log.Printf("New TLS connection from %s, Version: %x, CipherSuite: %x",
			conn.RemoteAddr(), state.Version, state.CipherSuite)

		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			log.Printf("Client certificate: %s", cert.Subject.CommonName)
		}
	}

	client := NewClient(conn, s.dataService, s.tokenService)
	s.AddClient(client)
	defer s.RemoveClient(client)

	log.Infof("New client connected: %s (total: %d)", conn.RemoteAddr(), len(s.clients))

	client.Run()

	log.Infof("Client disconnected: %s (remaining: %d)", conn.RemoteAddr(), len(s.clients)-1)
}
func (s *Server) Run(ctx context.Context) {
	log.Info("Server started on", s.listener.Addr())

	for {
		select {
		case <-ctx.Done():
			log.Info("Server shutting down")
			return
		case <-s.shutdown:
			log.Info("Server shutting down")
			return
		default:
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
}
func (s *Server) Close() error {
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
	return nil
}

func (s *Server) AddHandler(dataService ports.DataService, tokenService ports.TokenService) {
	s.dataService = dataService
	s.tokenService = tokenService
}
func isClosedError(err error) bool {
	return err.Error() == "use of closed network connection"
}
