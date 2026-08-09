package http

import (
	"net/http"
	"strconv"
	"time"

	"hyperfocus/internal/pkg/httputil"
	"hyperfocus/internal/usecases/moments"
)

// Moment handles GET /api/moments?at=&q=&sort=&dir=&limit=.
func (a *API) Moment(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var at time.Time
	if raw := q.Get("at"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httputil.Problem(w, http.StatusBadRequest, "invalid 'at' (use RFC3339)")
			return
		}
		at = t.UTC()
	}

	// No limit by default: return every stream in the moment. An explicit
	// ?limit=N (<=100000) is honored if provided.
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100000 {
			limit = n
		}
	}

	result, err := a.moments.MomentAt(r.Context(), moments.Params{
		At:       at,
		Query:    q.Get("q"),
		Survivor: q.Get("survivor"),
		Language: q.Get("language"),
		Sort:     q.Get("sort"),
		Dir:      q.Get("dir"),
		Limit:    limit,
	})
	if err != nil {
		a.log.Error("moment query failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}

	resp := momentResponse{
		RequestedAt: result.RequestedAt,
		HasData:     result.HasData,
		Streams:     []streamDTO{},
	}
	if result.HasData {
		resp.Snapshot = toSnapshotDTO(result.Snapshot)
		for _, d := range result.Streams {
			resp.Streams = append(resp.Streams, toStreamDTO(d))
		}
	}
	httputil.JSON(w, http.StatusOK, resp)
}
