package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nx-a/ring/internal/core/ports"
	log "github.com/sirupsen/logrus"
	"net/http"
	"runtime/debug"
	"time"
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
		Addr:         ":" + cfg.Get("server.port"),
		Handler:      def(auth(mux, services.TokenService)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
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
	j, _ := json.Marshal(data)
	_, err := w.Write(j)
	if err != nil {
		log.Error(err)
	}

}
func (s *Server) Write(w http.ResponseWriter, data any) {
	j, err := json.Marshal(data)
	if err != nil {
		log.Error(err)
		_, err = w.Write([]byte(`{"error": "internal server error"}`))
		if err != nil {
			log.Error(err)
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(j)
	if err != nil {
		log.Error(err)
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
			log.Error(err)
			done <- true
		}
	}()
	fmt.Println("Listening on " + s.srv.Addr)
	select {
	case <-ctx.Done():
		break
	case <-done:
		cancelFunc()
		break
	}
}
func (s *Server) Close() error {
	fmt.Println("Closing web server")
	err := s.srv.Close()
	if err != nil {
		log.Error(err)
		return err
	}
	err = s.srv.Shutdown(s.ctx)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (s *Server) Control(r *http.Request, w http.ResponseWriter) (map[string]any, bool) {
	control, ok := r.Context().Value("control").(map[string]any)
	if !ok {
		s.Error(w, http.StatusInternalServerError, map[string]any{"error": "control not found"})
		return nil, false
	}
	return control, true
}
func def(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err)
				debug.PrintStack()
			}
		}()
		next.ServeHTTP(w, r)
	})
}
