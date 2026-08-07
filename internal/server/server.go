// Package server assembles the HTTP server: API routes (via handlers), the
// embedded Vue SPA with history-fallback, and the middleware chain.
package server

import (
	"io/fs"
	"net/http"
	"strings"
	"time"

	"log/slog"

	httpHandlers "hyperfocus/internal/handlers/http"
	"hyperfocus/web"
)

// Config is the subset of config the server needs.
type Config struct {
	Addr    string
	Version string
}

// New builds the *http.Server with all routes and middleware mounted.
func New(logger *slog.Logger, apiDeps httpHandlers.Deps, cfg Config) *http.Server {
	mux := http.NewServeMux()

	api := httpHandlers.NewAPI(apiDeps)
	httpHandlers.Register(mux, api)

	mux.Handle("/", frontendHandler())

	handler := httpHandlers.Chain(mux,
		httpHandlers.CORS(),
		httpHandlers.Recover(logger),
		httpHandlers.Logging(logger),
	)

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// frontendHandler serves the embedded SPA, falling back to index.html for
// client-side routes (anything not matching a real asset file).
func frontendHandler() http.Handler {
	sub, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		// Should never happen given the embedded dist always exists.
		panic("server: cannot sub embedded dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never let the SPA shadow API or preview routes.
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/previews/") {
			http.NotFound(w, r)
			return
		}

		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(sub, clean); err != nil {
			// Unknown path: serve the SPA shell so client-side routing can resolve it.
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
