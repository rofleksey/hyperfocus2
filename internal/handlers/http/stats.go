package http

import (
	"net/http"
	"strconv"

	"hyperfocus/internal/pkg/httputil"
)

// Stats handles GET /api/stats?n= (default 50, max 500).
func (a *API) Stats(w http.ResponseWriter, r *http.Request) {
	n := 50
	if raw := r.URL.Query().Get("n"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 500 {
			n = v
		}
	}
	rows, err := a.statsRepo.SnapshotStats(r.Context(), n)
	if err != nil {
		a.log.Error("stats query failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"snapshots": rows})
}

// SubscriptionStats handles GET /api/subscriptions/stats.
func (a *API) SubscriptionStats(w http.ResponseWriter, r *http.Request) {
	rows, err := a.statsRepo.VerifiedSubscriberSeries(r.Context())
	if err != nil {
		a.log.Error("subscription stats query failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"points": rows})
}
