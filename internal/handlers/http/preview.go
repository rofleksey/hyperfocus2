package http

import (
	"net/http"
	"path/filepath"
	"strings"

	"hyperfocus/internal/pkg/httputil"
)

// Preview serves a captured full-resolution preview image from disk. Filenames
// are opaque UUIDs, so we only reject path traversal (no separators in the name).
func (a *API) Preview(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		httputil.Problem(w, http.StatusBadRequest, "invalid filename")
		return
	}
	path := filepath.Join(a.previews.Dir(), name)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	http.ServeFile(w, r, path)
}

// Thumb serves a fast low-resolution thumbnail from the thumbs subdirectory.
func (a *API) Thumb(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		httputil.Problem(w, http.StatusBadRequest, "invalid filename")
		return
	}
	path := filepath.Join(a.previews.ThumbsDir(), name)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	http.ServeFile(w, r, path)
}
