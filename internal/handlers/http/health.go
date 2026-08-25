package http

import (
	"net/http"

	"hyperfocus/internal/pkg/httputil"
)

// Health reports service liveness plus a database readiness probe: 200 with
// status "ok" when the DB answers a ping, 503 otherwise.
func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	if a.db != nil {
		if err := a.db.Ping(r.Context()); err != nil {
			httputil.JSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":  "degraded",
				"version": a.version,
				"db":      "unreachable",
			})
			return
		}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": a.version,
	})
}
