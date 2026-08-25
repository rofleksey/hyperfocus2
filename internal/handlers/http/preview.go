package http

import (
	"net/http"
	"path/filepath"
	"strings"

	"hyperfocus/internal/pkg/httputil"
)

// Preview serves a captured full-resolution preview image from disk. Filenames
// are opaque UUIDs, so we only reject path traversal (no separators in the name).
// Cache lifetime matches the data retention window: files are pruned after it,
// so advertising longer caching would leave caches pointing at 404s.
func (a *API) Preview(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		httputil.Problem(w, http.StatusBadRequest, "invalid filename")
		return
	}
	path := filepath.Join(a.previews.Dir(), name)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=10800")
	http.ServeFile(w, r, path)
}

// Thumb serves a low-resolution thumbnail. Thumbnails live in the same flat
// directory as previews (distinct UUID filenames).
func (a *API) Thumb(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		httputil.Problem(w, http.StatusBadRequest, "invalid filename")
		return
	}
	path := filepath.Join(a.previews.Dir(), name)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=10800")
	http.ServeFile(w, r, path)
}
