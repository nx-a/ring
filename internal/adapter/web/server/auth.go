package server

import (
	"net/http"
	"strings"

	"github.com/nx-a/ring/internal/core/ports"
	ctx "github.com/nx-a/ring/internal/engine/context"
	"github.com/nx-a/ring/internal/engine/logger"
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
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")
		claim, err := Verify(token)
		if err != nil {
			claim, err = tokenService.GetByToken(token)
		}
		if err != nil {
			logger.FromContext(r.Context()).WithError(err).Warn("auth failed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		logger.FromContext(r.Context()).WithField("claim", claim).Info("auth success")
		r = r.WithContext(ctx.WithControl(r.Context(), claim))
		next.ServeHTTP(w, r)
	})
}
