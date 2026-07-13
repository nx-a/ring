package route

import (
	"net/http"

	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/core/ports"
)

func Status(s *server.Server, service ports.StatusService) {
	s.Mux().HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		status, err := service.Status(r.Context())
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		s.Write(w, status)
	})
	s.Mux().HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics, err := service.Metrics(r.Context())
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		s.Write(w, metrics)
	})
}
