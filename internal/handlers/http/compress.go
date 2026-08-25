package http

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// Compress transparently gzip-compresses compressible responses for clients
// that send Accept-Encoding: gzip. The decision is made from the response's
// Content-Type once headers are written, so binary payloads (preview JPEGs,
// fonts) pass through untouched.
func Compress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !acceptsGzip(r) {
				next.ServeHTTP(&varyWriter{ResponseWriter: w}, r)
				return
			}
			gw := &gzipResponseWriter{ResponseWriter: w}
			defer gw.close()
			next.ServeHTTP(gw, r)
		})
	}
}

func acceptsGzip(r *http.Request) bool {
	enc := r.Header.Get("Accept-Encoding")
	for _, part := range strings.Split(enc, ",") {
		if strings.EqualFold(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]), "gzip") {
			return true
		}
	}
	return false
}

// varyWriter only adds the Vary header for non-gzip clients so shared caches
// do not serve compressed responses to them.
type varyWriter struct {
	http.ResponseWriter
	wrote bool
}

func (v *varyWriter) ensure() {
	if v.wrote {
		return
	}
	v.wrote = true
	v.Header().Add("Vary", "Accept-Encoding")
}

func (v *varyWriter) WriteHeader(code int) {
	v.ensure()
	v.ResponseWriter.WriteHeader(code)
}

func (v *varyWriter) Write(b []byte) (int, error) {
	v.ensure()
	return v.ResponseWriter.Write(b)
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func (g *gzipResponseWriter) compressible() bool {
	h := g.Header()
	if h.Get("Content-Encoding") != "" {
		return false
	}
	ct := h.Get("Content-Type")
	mt := strings.TrimSpace(strings.ToLower(strings.SplitN(ct, ";", 2)[0]))
	switch {
	case strings.HasPrefix(mt, "text/"):
		return true
	case mt == "application/json",
		mt == "application/javascript",
		mt == "application/xml",
		mt == "image/svg+xml",
		mt == "application/manifest+json":
		return true
	}
	return false
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	if code >= 200 && code != http.StatusNoContent && code != http.StatusNotModified && g.compressible() {
		h := g.Header()
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
		g.gz = gzip.NewWriter(g.ResponseWriter)
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if g.gz != nil {
		return g.gz.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipResponseWriter) Flush() {
	if g.gz != nil {
		_ = g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) close() {
	if g.gz != nil {
		_ = g.gz.Close()
	}
}
