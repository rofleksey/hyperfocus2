package http

import (
	"net/http"
)

// Register mounts the API routes on the given mux (Go 1.22+ pattern routing).
func Register(mux *http.ServeMux, api *API) {
	mux.HandleFunc("GET /api/healthz", api.Health)
	mux.HandleFunc("GET /api/moments", api.Moment)
	mux.HandleFunc("GET /api/sample", api.SampleDetail)
	mux.HandleFunc("GET /api/snapshots", api.Snapshots)
	mux.HandleFunc("GET /api/stats", api.Stats)
	mux.HandleFunc("GET /api/streamers", api.Streamers)
	mux.HandleFunc("GET /api/streamers/{id}", api.Streamer)
	mux.HandleFunc("GET /previews/{filename}", api.Preview)
	mux.HandleFunc("GET /previews/thumbs/{filename}", api.Thumb)
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}
