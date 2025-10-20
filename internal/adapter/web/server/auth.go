package server

import (
	"context"
	"github.com/nx-a/ring/internal/core/ports"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

var wl = map[string]bool{
	"/auth/in": true, "/auth/reg": true,
}

func auth(next http.Handler, tokenService ports.TokenService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, find := wl[r.URL.Path]; find {
			next.ServeHTTP(w, r)
			return
		}
		var token string
		if token = r.Header.Get("Authorization"); token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")
		claim, err := Verify(token)
		if err != nil {
			claim, err = tokenService.GetByToken(token)
		}
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		log.Info(claim)
		r = r.WithContext(context.WithValue(context.Background(), "control", claim))
		next.ServeHTTP(w, r)
	})
}
