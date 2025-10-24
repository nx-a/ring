package quicSrv

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/nx-a/ring/hook"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

type QUICServer struct {
	listener     *quic.Listener
	config       *quic.Config
	dataService  ports.DataService
	tokenService ports.TokenService
}

func New(addr string, tlsConfig *tls.Config, config *quic.Config, dataService ports.DataService, tokenService ports.TokenService) (*QUICServer, error) {

	listener, err := quic.ListenAddr(addr, tlsConfig, config)
	if err != nil {
		return nil, err
	}
	return &QUICServer{
		listener:     listener,
		config:       config,
		dataService:  dataService,
		tokenService: tokenService,
	}, nil
}
func (s *QUICServer) Start(ctx context.Context) {
	log.Infof("QUIC server started on %s", s.listener.Addr())
	for {
		select {
		case <-ctx.Done():
			log.Info("QUIC server stopping...")
			return
		default:
			conn, err := s.listener.Accept(ctx)
			if err != nil {
				log.Errorf("QUIC server accept error: %s", err)
				continue
			}
			go s.handle(conn)

		}
	}

}
func (s *QUICServer) handle(conn *quic.Conn) {
	defer conn.CloseWithError(0, "connection closed")
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("New QUIC connection from: %s", remoteAddr)
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Printf("Failed to accept stream from %s: %v", remoteAddr, err)
			break
		}

		go s.handleStream(stream, conn)
	}
}
func (s *QUICServer) handleStream(stream *quic.Stream, conn *quic.Conn) {
	defer stream.Close()
	remoteAddr := conn.RemoteAddr().String()
	streamID := stream.StreamID()

	log.Printf("New stream %d from %s", streamID, remoteAddr)

	buffer := make([]byte, 4096)
	for {
		// Устанавливаем таймаут для чтения
		stream.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := stream.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Printf("Stream %d from %s: client closed stream", streamID, remoteAddr)
			} else {
				log.Printf("Stream %d from %s: read error: %v", streamID, remoteAddr, err)
			}
			break
		}
		data := buffer[:n]
		response := s.processMessage(string(data))
		stream.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, err = stream.Write([]byte(response)); err != nil {
			log.Printf("Stream %d from %s: write error: %v", streamID, remoteAddr, err)
			break
		}
	}
}
func (s *QUICServer) processMessage(message string) string {
	rawJson, err := base64.StdEncoding.DecodeString(strings.TrimSpace(message))
	if err != nil {
		log.Infof("Decode message failed: %v", err)
	}
	var entry hook.LogEntry
	err = json.Unmarshal(rawJson, &entry)
	if err != nil {
		log.Infof("Decode message failed: %v", err)
	}
	claim, err := s.tokenService.GetByToken(entry.Token)
	if err != nil {
		log.Infof("Get token failed: %v", err)
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
	s.dataService.Write(context.WithValue(context.Background(), "control", claim), domain.Data{
		Ext:   entry.AppName,
		Time:  &_time,
		Level: entry.Level,
		Val:   val,
	})
	return "DONE\n"
}
func (s *QUICServer) Close() error {
	return s.listener.Close()
}
