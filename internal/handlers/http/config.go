package http

import (
	"net/http"

	"hyperfocus/internal/pkg/httputil"
)

type configResponse struct {
	RetentionHours int  `json:"retention_hours"`
	NotifyEnabled  bool `json:"notify_enabled"`
}

// Config exposes the public runtime configuration (GET /api/config).
func (a *API) Config(w http.ResponseWriter, _ *http.Request) {
	httputil.JSON(w, http.StatusOK, configResponse{
		RetentionHours: a.public.RetentionHours,
		NotifyEnabled:  a.public.NotifyEnabled,
	})
}
