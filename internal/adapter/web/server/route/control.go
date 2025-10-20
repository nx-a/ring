package route

import (
	"encoding/json"
	"github.com/fatih/structs"
	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
	"net/http"
)

type Auth struct {
	Login    string
	Password string
}

func Control(s *server.Server, service ports.ControlService) {
	s.Mux().HandleFunc("POST /auth/in", func(w http.ResponseWriter, r *http.Request) {
		auth := conv.Parse[Auth](w, r)
		if auth == nil {
			return
		}
		_control, err := service.LogIn(auth.Login, auth.Password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		token, err := server.NewJwt(structs.Map(_control))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		jsonData, _ := json.Marshal(map[string]any{
			"token":  token,
			"status": "ok",
		})
		w.Write(jsonData)
	})
	s.Mux().HandleFunc("POST /auth/reg", func(w http.ResponseWriter, r *http.Request) {
		auth := conv.Parse[Auth](w, r)
		if auth == nil {
			return
		}
		_control, err := service.Reg(auth.Login, auth.Password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		token, err := server.NewJwt(structs.Map(_control))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		jsonData, _ := json.Marshal(map[string]any{
			"token":  token,
			"status": "ok",
		})
		w.Write(jsonData)
	})

}
