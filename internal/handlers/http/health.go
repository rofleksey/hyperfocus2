package http

import (
	"net/http"

	"hyperfocus/internal/pkg/httputil"
)

// Health responds with the service version.
func (a *API) Health(w http.ResponseWriter, _ *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": a.version,
	})
}
