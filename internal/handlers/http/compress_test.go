package http

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompressGzipsJSON(t *testing.T) {
	body := strings.Repeat(`{"ok":true}`, 100)
	h := Compress()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected gzip Content-Encoding")
	}
	if !strings.Contains(w.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatal("expected Vary: Accept-Encoding")
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("body is not valid gzip: %v", err)
	}
	decoded, _ := io.ReadAll(zr)
	if string(decoded) != body {
		t.Fatal("decoded body mismatch")
	}
}

func TestCompressSkipsImages(t *testing.T) {
	h := Compress()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0xff, 0xd8, 0xff, 0xe0})
	}))

	r := httptest.NewRequest(http.MethodGet, "/previews/x.jpg", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("jpeg responses must not be gzipped")
	}
}

func TestCompressPassthroughWithoutAcceptEncoding(t *testing.T) {
	h := Compress()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<html></html>")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Header().Get("Content-Encoding") != "" {
		t.Fatal("expected no Content-Encoding for plain client")
	}
	if !strings.Contains(w.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatal("expected Vary: Accept-Encoding so caches don't mix representations")
	}
	if w.Body.String() != "<html></html>" {
		t.Fatal("body must be uncompressed")
	}
}
