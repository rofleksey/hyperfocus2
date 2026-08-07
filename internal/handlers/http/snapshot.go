package http

import (
	"net/http"
	"strconv"
	"time"

	"hyperfocus/internal/pkg/httputil"
)

// Snapshots handles GET /api/snapshots?from=&to=&limit=.
func (a *API) Snapshots(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var from, to *time.Time
	if raw := q.Get("from"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			tt := t.UTC()
			from = &tt
		} else {
			httputil.Problem(w, http.StatusBadRequest, "invalid 'from'")
			return
		}
	}
	if raw := q.Get("to"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			tt := t.UTC()
			to = &tt
		} else {
			httputil.Problem(w, http.StatusBadRequest, "invalid 'to'")
			return
		}
	}

	limit := 100
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	snaps, err := a.moments.Snapshots(r.Context(), from, to, limit)
	if err != nil {
		a.log.Error("snapshots query failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}

	out := make([]snapshotDTO, 0, len(snaps))
	for _, s := range snaps {
		out = append(out, *toSnapshotDTO(s))
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"data": out})
}
