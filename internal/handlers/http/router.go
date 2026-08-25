package http

import (
	"log/slog"
	"net/http"
	"time"
)

// Register mounts the API routes on the given mux (Go 1.22+ pattern routing).
// All /api routes share a per-IP rate limit; /api/subscribe gets a much
// stricter one on top because each POST triggers outbound Twitch/Steam calls
// and an IRC join+message.
func Register(mux *http.ServeMux, api *API, log *slog.Logger) {
	apiLimiter := rateLimitMiddleware(log, "api", newVisitorLimiter(120, time.Minute))
	subLimiter := rateLimitMiddleware(log, "subscribe", newVisitorLimiter(10, time.Minute))

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /api/healthz", api.Health)
	apiMux.HandleFunc("GET /api/config", api.Config)
	apiMux.HandleFunc("GET /api/moments", api.Moment)
	apiMux.HandleFunc("GET /api/sample", api.SampleDetail)
	apiMux.HandleFunc("GET /api/snapshots", api.Snapshots)
	apiMux.HandleFunc("GET /api/stats", api.Stats)
	apiMux.HandleFunc("GET /api/subscriptions/stats", api.SubscriptionStats)
	apiMux.HandleFunc("GET /api/streamers", api.Streamers)
	apiMux.HandleFunc("GET /api/streamers/{id}", api.Streamer)
	apiMux.HandleFunc("/api/subscribe", api.Subscribe)

	mux.Handle("/api/subscribe", subLimiter(apiMux))
	mux.Handle("/api/", apiLimiter(apiMux))

	mux.HandleFunc("GET /previews/{filename}", api.Preview)
	mux.HandleFunc("GET /previews/thumbs/{filename}", api.Thumb)
}
