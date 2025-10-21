package route

import (
	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/adapter/web/server/route/dto"
	_map "github.com/nx-a/ring/internal/adapter/web/server/route/map"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func Token(s *server.Server, ts ports.TokenService) {
	s.Mux().HandleFunc("DELETE /token/{id}", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		id := conv.ParseValue[uint64](r, "id")
		if id == 0 {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "id not found"})
			return
		}
		ts.Remove(conv.ToUint(control["ControlId"]), id)
	})
	s.Mux().HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		_dto := conv.Parse[dto.Token](w, r)
		_token := _map.TokenToDomain(_dto)
		log.Debug(_dto)
		log.Debug(_token)
		if _token == nil || _token.BucketId == 0 || _token.Type == 0 {
			s.Error(w, 400, "invalid token")
			return
		}
		dt := ts.Add(conv.ToUint(control["ControlId"]), *_token)
		s.Write(w, _map.TokenFromDomain(&dt))
	})
	s.Mux().HandleFunc("GET /token/by/bucket/{id}", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		id := conv.ParseValue[uint64](r, "id")
		if id == 0 {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "id not found"})
			return
		}
		dts := ts.GetByBucketId(conv.ToUint(control["ControlId"]), id)
		if dts == nil {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "bad request"})
			return
		}
		dtsDto := make([]*dto.Token, 0, len(dts))
		for _, dt := range dts {
			dtsDto = append(dtsDto, _map.TokenFromDomain(&dt))
		}
		s.Write(w, dtsDto)
	})
}
