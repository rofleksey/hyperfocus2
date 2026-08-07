// Package httputil contains small helpers shared by HTTP handlers.
package httputil

import (
	"encoding/json"
	"net/http"
)

// JSON writes value as JSON with the given status, setting standard headers.
func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

// Problem writes a JSON error envelope consistent across all API failures.
func Problem(w http.ResponseWriter, status int, title string) {
	JSON(w, status, map[string]any{
		"status": status,
		"error":  title,
	})
}

// DecodeJSON decodes a JSON request body into dst. It limits the read to 1 MiB.
func DecodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
