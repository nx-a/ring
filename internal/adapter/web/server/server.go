package server

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/nx-a/ring/internal/core/ports"
	ctx "github.com/nx-a/ring/internal/engine/context"
	"github.com/nx-a/ring/internal/engine/logger"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	srv      *http.Server
	mux      *http.ServeMux
	ctx      context.Context
	services *Services
}
type Services struct {
	TokenService  ports.TokenService
	BucketService ports.BucketService
	DataService   ports.DataService
	PointService  ports.PointService
}
type Config interface {
	Get(string) string
}

func New(cfg Config, services *Services) *Server {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(cfg.Get("server.static")))
	mux.Handle("/", fileServer)

	srv := http.Server{
		Addr:              ":" + cfg.Get("server.port"),
		Handler:           requestID(def(auth(mux, services.TokenService))),
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return &Server{
		srv:      &srv,
		mux:      mux,
		services: services,
	}
}
func (s *Server) Error(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	j, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("failed to marshal error response")
	}
	_, err = w.Write(j)
	if err != nil {
		log.WithError(err).Error("failed to write error response")
	}
}
func (s *Server) Write(w http.ResponseWriter, data any) {
	j, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("failed to marshal response")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(j)
	if err != nil {
		log.WithError(err).Error("failed to write response")
	}
}
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

func (s *Server) Listen(ctx context.Context, cancelFunc context.CancelFunc) {
	s.ctx = ctx
	done := make(chan bool, 1)
	go func() {
		err := s.srv.ListenAndServe()
		if err != nil {
			log.WithError(err).Error("http server error")
			done <- true
		}
	}()
	log.WithField("addr", s.srv.Addr).Info("Listening")
	select {
	case <-ctx.Done():
		break
	case <-done:
		cancelFunc()
		break
	}
}
func (s *Server) Close() error {
	log.Info("Closing web server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		log.WithError(err).Error("http server shutdown error")
		return err
	}
	return nil
}

func (s *Server) Control(r *http.Request, w http.ResponseWriter) (map[string]any, bool) {
	control, ok := ctx.Control(r.Context())
	if !ok {
		s.Error(w, http.StatusInternalServerError, map[string]any{"error": "control not found"})
		return nil, false
	}
	return control, true
}

func (s *Server) RequireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !ctx.IsAdmin(r.Context()) {
		s.Error(w, http.StatusForbidden, map[string]any{"error": "admin required"})
		return false
	}
	return true
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = logger.NewRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		c := ctx.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(c))
	})
}

func def(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		entry := logger.FromContext(r.Context()).WithFields(log.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		})
		entry.Info("request started")
		defer func() {
			if err := recover(); err != nil {
				entry.WithField("panic", err).Error("panic recovered")
				debug.PrintStack()
				http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
			}
			entry.WithField("duration", time.Since(start).String()).Info("request finished")
		}()
		next.ServeHTTP(w, r)
	})
}
