package route

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
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
		err := ds.Write(r.Context(), &domain.Data{
			Ext:   _dto.Ext,
			Time:  _dto.Time,
			Level: _dto.Level,
			Val:   _dto.Data,
		})
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		s.Write(w, map[string]any{"write": "done"})
	})
	s.Mux().HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
		_dtoFind, err := buildDataSelect(r)
		if err != nil {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		resp, err := ds.Find(r.Context(), _dtoFind)
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		s.Write(w, resp)
	})
	s.Mux().HandleFunc("GET /data/export", func(w http.ResponseWriter, r *http.Request) {
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}
		_dtoFind, err := buildDataSelect(r)
		if err != nil {
			s.Error(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		resp, err := ds.Find(r.Context(), _dtoFind)
		if err != nil {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		switch format {
		case "csv":
			exportCSV(w, resp)
		case "txt":
			exportTXT(w, resp)
		default:
			exportJSON(w, resp)
		}
	})
}

func buildDataSelect(r *http.Request) (*dto.DataSelect, error) {
	q := r.URL.Query()
	buckets := q["bucket"]
	if len(buckets) == 0 || buckets[0] == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	return &dto.DataSelect{
		Ext:       q["ext"],
		TimeStart: conv.FirstQuery[time.Time](q["timeStart"]),
		TimeEnd:   conv.FirstQuery[time.Time](q["timeEnd"]),
		Level:     q["level"],
		Data:      q["data"],
		Bucket:    buckets[0],
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func exportJSON(w http.ResponseWriter, data []domain.Data) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"logs.json\"")
	w.WriteHeader(http.StatusOK)
	items := make([]map[string]any, 0, len(data))
	for i := range data {
		items = append(items, dataToMap(&data[i]))
	}
	_ = json.NewEncoder(w).Encode(items)
}

func exportTXT(w http.ResponseWriter, data []domain.Data) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=\"logs.txt\"")
	w.WriteHeader(http.StatusOK)
	for _, d := range data {
		fmt.Fprintf(w, "%s [%s] point=%d %s\n", formatTime(d.Time), d.Level, d.PointId, string(d.Val))
	}
}

func exportCSV(w http.ResponseWriter, data []domain.Data) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"logs.csv\"")
	w.WriteHeader(http.StatusOK)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"time", "level", "point_id", "data"})
	for _, d := range data {
		_ = writer.Write([]string{formatTime(d.Time), d.Level, strconv.FormatUint(d.PointId, 10), string(d.Val)})
	}
	writer.Flush()
}

func dataToMap(d *domain.Data) map[string]any {
	parsed := make(map[string]any)
	_ = json.Unmarshal(d.Val, &parsed)
	return map[string]any{
		"dataId":   d.DataId,
		"bucketId": d.BucketId,
		"pointId":  d.PointId,
		"time":     d.Time,
		"level":    d.Level,
		"data":     parsed,
	}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
