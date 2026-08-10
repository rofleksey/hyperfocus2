package http

import (
	"net/http"
	"time"

	"hyperfocus/internal/pkg/httputil"
)

// SampleDetail handles GET /api/sample?streamer_id=&at=.
func (a *API) SampleDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	streamerID := q.Get("streamer_id")
	if streamerID == "" {
		httputil.Problem(w, http.StatusBadRequest, "streamer_id required")
		return
	}
	var at time.Time
	if raw := q.Get("at"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httputil.Problem(w, http.StatusBadRequest, "invalid 'at' (use RFC3339)")
			return
		}
		at = t.UTC()
	}

	d, err := a.moments.SampleAt(r.Context(), streamerID, at)
	if err != nil {
		a.log.Error("sample query failed", "error", err, "streamer_id", streamerID)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}
	if d == nil {
		httputil.Problem(w, http.StatusNotFound, "stream not found in any snapshot")
		return
	}
	httputil.JSON(w, http.StatusOK, toStreamDTO(*d))
}
