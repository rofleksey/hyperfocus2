// Package server assembles the HTTP server: API routes (via handlers), the
// embedded Vue SPA with history-fallback, and the middleware chain.
package server

import (
	"io/fs"
	"net/http"
	"path"
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
	httpHandlers.Register(mux, api, logger)

	mux.Handle("/", frontendHandler())

	handler := httpHandlers.Chain(mux,
		httpHandlers.Recover(logger),
		httpHandlers.Logging(logger),
		httpHandlers.Compress(),
	)

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
}

// frontendHandler serves the embedded Vue SPA:
//   - real files under /assets/ (content-hashed) get immutable year-long
//     caching; other root files (favicon, robots.txt, …) get a short cache;
//   - missing files under /assets/ are a hard 404 (never the SPA shell, which
//     would poison deploys with stale-index 200s);
//   - unknown extensionless paths fall back to index.html (no-cache) so
//     client-side routing works;
//   - directory listings are disabled and the fallback is GET/HEAD only.
func frontendHandler() http.Handler {
	sub, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		panic("server: cannot sub embedded dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))
	indexBytes := readIndex(sub)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never let the SPA shadow API or preview routes.
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/previews/") {
			http.NotFound(w, r)
			return
		}

		clean := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if clean == "" || clean == "." {
			clean = "index.html"
		}

		// The SPA shell is always served with no-cache, whether requested
		// directly or via a history-fallback route.
		stat, statErr := fs.Stat(sub, clean)
		switch {
		case clean == "index.html":
			serveShell(w, r, indexBytes)
		case statErr == nil && !stat.IsDir():
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "public, max-age=3600")
			}
			fileServer.ServeHTTP(w, r)
		case statErr == nil && stat.IsDir():
			// No directory listings.
			http.NotFound(w, r)
		case strings.HasPrefix(r.URL.Path, "/assets/"):
			// Stale hashed asset after a deploy: a real 404, not the SPA shell.
			http.NotFound(w, r)
		case r.Method != http.MethodGet && r.Method != http.MethodHead:
			http.NotFound(w, r)
		case strings.Contains(path.Base(clean), "."):
			// A missing file with an extension (outside /assets) is still a
			// 404, not an SPA route.
			http.NotFound(w, r)
		default:
			serveShell(w, r, indexBytes)
		}
	})
}

func serveShell(w http.ResponseWriter, r *http.Request, indexBytes []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(indexBytes)
	}
}

func readIndex(sub fs.FS) []byte {
	b, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic("server: embedded dist has no index.html: " + err.Error())
	}
	return b
}
