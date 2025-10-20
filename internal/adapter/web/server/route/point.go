package route

import (
	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/adapter/web/server/route/dto"
	_map "github.com/nx-a/ring/internal/adapter/web/server/route/map"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
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
		ps.Remove(control["ControlId"].(uint64), id)
		w.Write([]byte{})
	})
	s.Mux().HandleFunc("POST /point", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		_dto := conv.Parse[dto.Point](w, r)
		_point := _map.PointToDomain(_dto)
		if _point == nil || strings.TrimSpace(_point.ExternalId) == "" {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": "create error"})
			return
		}
		_point.ExternalId = strings.TrimSpace(_point.ExternalId)
		_pointResponse := ps.Add(control["ControlId"].(uint64), *_point)
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
		s.Write(w, ps.GetByBacketId(control["ControlId"].(uint64), id))
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
		s.Write(w, ps.GetByExternal(control["ControlId"].(uint64), id))
	})
}
