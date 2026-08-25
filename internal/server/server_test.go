package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFrontendHandler(t *testing.T) {
	h := frontendHandler()

	t.Run("index served with no-cache", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("got %d want 200", w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Fatalf("got Cache-Control %q want no-cache", cc)
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("got Content-Type %q", ct)
		}
	})

	t.Run("SPA route falls back to index", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/live", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("got %d want 200", w.Code)
		}
		if !strings.Contains(w.Body.String(), "<div id=\"app\">") {
			t.Fatal("expected SPA shell")
		}
	})

	t.Run("missing hashed asset is a real 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/does-not-exist-1234.js", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("got %d want 404", w.Code)
		}
	})

	t.Run("existing asset gets immutable cache", func(t *testing.T) {
		index := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, index)
		// find a real asset reference in the served html
		body := w.Body.String()
		start := strings.Index(body, "/assets/")
		if start < 0 {
			t.Skip("no hashed assets referenced in index.html")
		}
		end := strings.IndexAny(body[start:], "\"'")
		assetPath := body[start : start+end]

		wa := httptest.NewRecorder()
		h.ServeHTTP(wa, httptest.NewRequest(http.MethodGet, assetPath, nil))
		if wa.Code != http.StatusOK {
			t.Fatalf("asset %s: got %d want 200", assetPath, wa.Code)
		}
		if cc := wa.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
			t.Fatalf("got Cache-Control %q", cc)
		}
	})

	t.Run("api paths never fall back to SPA", func(t *testing.T) {
		for _, p := range []string{"/api/whatever", "/previews/x.jpg"} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
			if w.Code != http.StatusNotFound {
				t.Fatalf("%s: got %d want 404", p, w.Code)
			}
		}
	})

	t.Run("non-GET on unknown path is 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/some/spa/route", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("got %d want 404", w.Code)
		}
	})

	t.Run("directory listing disabled", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("got %d want 404", w.Code)
		}
	})

	t.Run("static root file with short cache", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("got %d want 200", w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=3600" {
			t.Fatalf("got Cache-Control %q", cc)
		}
	})
}
