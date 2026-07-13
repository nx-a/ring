package route

import (
	"encoding/json"
	"net/http"

	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
	log "github.com/sirupsen/logrus"
)

type Auth struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func Control(s *server.Server, service ports.ControlService) {
	s.Mux().HandleFunc("POST /auth/in", func(w http.ResponseWriter, r *http.Request) {
		auth := conv.Parse[Auth](w, r)
		if auth == nil {
			return
		}
		_control, err := service.LogIn(auth.Login, auth.Password)
		if err != nil {
			s.Error(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
			return
		}
		token, err := server.NewJwt(map[string]any{
			"ControlId": _control.ControlId,
			"Login":     _control.Login,
			"Role":      _control.Role,
			"Buckets":   _control.Buckets,
		})
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		jsonData, err := json.Marshal(map[string]any{
			"token":  token,
			"status": "ok",
		})
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonData)
	})
	s.Mux().HandleFunc("POST /auth/reg", func(w http.ResponseWriter, r *http.Request) {
		auth := conv.Parse[Auth](w, r)
		if auth == nil {
			return
		}
		log.Info(auth)
		_control, err := service.Reg(auth.Login, auth.Password, auth.Role)
		if err != nil {
			s.Error(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
			return
		}
		log.Info(_control)
		token, err := server.NewJwt(map[string]any{
			"ControlId": _control.ControlId,
			"Login":     _control.Login,
			"Role":      _control.Role,
			"Buckets":   _control.Buckets,
		})
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		jsonData, err := json.Marshal(map[string]any{
			"token":  token,
			"status": "ok",
		})
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonData)
	})

}
