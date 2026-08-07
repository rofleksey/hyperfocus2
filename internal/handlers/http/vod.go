package http

import (
	"errors"
	"net/http"
	"time"

	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/httputil"
)

type vodDTO struct {
	VodID        string    `json:"vod_id"`
	StreamerID   string    `json:"streamer_id"`
	StartedAt    time.Time `json:"started_at"`
	DurationSecs *int      `json:"duration_seconds,omitempty"`
}

// Vod handles GET /api/vods/{id}.
func (a *API) Vod(w http.ResponseWriter, r *http.Request) {
	vodID := r.PathValue("id")

	v, err := a.vods.GetVod(r.Context(), vodID)
	if errors.Is(err, entity.ErrNotFound) {
		httputil.Problem(w, http.StatusNotFound, "vod not found")
		return
	}
	if err != nil {
		a.log.Error("vod query failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}

	httputil.JSON(w, http.StatusOK, vodDTO{
		VodID:        v.VodID,
		StreamerID:   v.StreamerID,
		StartedAt:    v.StartedAt,
		DurationSecs: v.DurationSeconds,
	})
}
