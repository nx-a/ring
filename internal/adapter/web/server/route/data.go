package route

import (
	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/adapter/web/server/route/dto"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/conv"
	"net/http"
)

func Data(s *server.Server, ds ports.DataService) {
	s.Mux().HandleFunc("POST /data", func(w http.ResponseWriter, r *http.Request) {
		_dto := conv.Parse[dto.Data](w, r)
		if _dto == nil || _dto.Ext == "" {
			s.Error(w, http.StatusInternalServerError, map[string]any{"error": "data not found"})
			return
		}
		ds.Write(r.Context(), domain.Data{
			Ext:  _dto.Ext,
			Time: _dto.Time,
			Val:  _dto.Data,
		})

		/*
			resp["tokenId"] = d.TokenId
			resp["bucketId"] = d.BucketId
			resp["bucket"] = _backet.SystemName
			resp["type"] = d.Type
			resp["token"] = d.Val

		*/

	})
}
