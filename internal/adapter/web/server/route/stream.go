package route

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
)

func Stream(s *server.Server, ds ports.DataService) {
	s.Mux().HandleFunc("GET /logs/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": "streaming not supported"})
			return
		}

		fmt.Fprint(w, "event: connected\ndata: {}\n\n")
		flusher.Flush()

		ch, id := ds.Subscribe()
		defer ds.Unsubscribe(id)

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case log, ok := <-ch:
				if !ok {
					return
				}
				if err := sendSSE(w, flusher, &log); err != nil {
					return
				}
			}
		}
	})
}

type logEvent struct {
	DataId   string         `json:"dataId"`
	BucketId uint64         `json:"bucketId"`
	PointId  uint64         `json:"pointId"`
	Time     *time.Time     `json:"time"`
	Level    string         `json:"level"`
	Val      map[string]any `json:"val"`
}

func sendSSE(w http.ResponseWriter, flusher http.Flusher, log *domain.Data) error {
	parsed := make(map[string]any)
	_ = json.Unmarshal(log.Val, &parsed)
	evt := logEvent{
		DataId:   log.DataId,
		BucketId: log.BucketId,
		PointId:  log.PointId,
		Time:     log.Time,
		Level:    log.Level,
		Val:      parsed,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
	if err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
