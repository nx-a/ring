package route

import (
	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func Data(s *server.Server, ds ports.DataService) {
	s.Mux().HandleFunc("POST /data", func(w http.ResponseWriter, r *http.Request) {
		_dto := conv.Parse[dto.Data](w, r)
		if _dto == nil {
			return
		}
		if _dto.Ext == "" {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": "data not found"})
			return
		}
		err := ds.Write(r.Context(), domain.Data{
			Ext:   _dto.Ext,
			Time:  _dto.Time,
			Level: _dto.Level,
			Val:   _dto.Data,
		})
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		}
		s.Write(w, map[string]any{"write": "done"})

	})
	s.Mux().HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		_dtoFind := dto.DataSelect{
			Ext:       q["ext"],
			TimeStart: conv.FirstQuery[time.Time](q["timeStart"]),
			TimeEnd:   conv.FirstQuery[time.Time](q["timeEnd"]),
			Level:     q["level"],
			Data:      q["data"],
		}
		log.Info(_dtoFind)
		//ds.Write()

	})
}
