package route

import (
	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/adapter/web/server/route/dto"
	_map "github.com/nx-a/ring/internal/adapter/web/server/route/map"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

func Point(s *server.Server, ps ports.PointService) {
	s.Mux().HandleFunc("DELETE /point/{id}", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		id := conv.ParseValue[uint64](r, "id")
		if id == 0 {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "id not found"})
			return
		}
		ps.Remove(conv.ToUint(control["ControlId"]), id)
		w.Write([]byte{})
	})
	s.Mux().HandleFunc("POST /point", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		_dto := conv.Parse[dto.Point](w, r)
		_point := _map.PointToDomain(_dto)
		if _point == nil {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "point is empty"})
		}
		_point.ExternalId = strings.TrimSpace(_point.ExternalId)
		if _point.ExternalId == "" {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "ext is empty"})
			return
		}
		if _point.BucketId == 0 {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "bucket is required"})
			return
		}
		_point.ExternalId = strings.TrimSpace(_point.ExternalId)
		log.Debug(conv.ToUint(control["ControlId"]))
		_pointResponse := ps.Add(conv.ToUint(control["ControlId"]), *_point)
		s.Write(w, _pointResponse)
	})
	s.Mux().HandleFunc("GET /point/by/bucket/{id}", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		id := conv.ParseValue[uint64](r, "id")
		if id == 0 {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "id not found"})
			return
		}
		s.Write(w, ps.GetByBacketId(conv.ToUint(control["ControlId"]), id))
	})
	s.Mux().HandleFunc("GET /point/by/external/{id}", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		id := strings.TrimSpace(conv.ParseValue[string](r, "id"))
		if id == "" {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "id not found"})
			return
		}
		s.Write(w, ps.GetByExternal(conv.ToUint(control["ControlId"]), id))
	})
}
