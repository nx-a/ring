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

func Bucket(s *server.Server, service ports.BucketService) {
	s.Mux().HandleFunc("GET /bucket", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		controls, err := service.GetByControl(conv.ToUint(control["ControlId"]))
		log.Info(controls, err)
		if err != nil || controls == nil || len(controls) == 0 {
			s.Write(w, []any{})
			return
		}
		s.Write(w, controls)
	})
	s.Mux().HandleFunc("POST /bucket", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		_dto := conv.Parse[dto.Bucket](w, r)
		_bucket := _map.BucketToDomain(_dto)
		if _bucket == nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": "create error"})
			return
		}
		_bucket.ControlId = conv.ToUint(control["ControlId"])
		if _bucket.TimeLife == 0 {
			_bucket.TimeLife = 10 * 24 //10 дней
		}
		_bucket = service.Add(_bucket)
		s.Write(w, _bucket)
	})
	s.Mux().HandleFunc("DELETE /bucket/{id}", func(w http.ResponseWriter, r *http.Request) {
		control, ok := s.Control(r, w)
		if !ok {
			return
		}
		id := conv.ParseValue[uint64](r, "id")
		if id == 0 {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": "id not found"})
			return
		}
		_buckets, err := service.GetByControl(conv.ToUint(control["ControlId"]))
		if err != nil || _buckets == nil || len(_buckets) == 0 {
			s.Write(w, []any{})
			return
		}
		for _, ctr := range _buckets {
			if ctr.BucketId == id {
				service.Remove(ctr.BucketId)
				s.Write(w, []any{})
				return
			}
		}
		s.Write(w, []any{})
	})
}
