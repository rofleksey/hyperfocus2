package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigEndpoint(t *testing.T) {
	api := &API{public: PublicConfig{RetentionHours: 3, NotifyEnabled: true}}
	w := httptest.NewRecorder()
	api.Config(w, httptest.NewRequest(http.MethodGet, "/api/config", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("got %d want 200", w.Code)
	}
	var got struct {
		RetentionHours int  `json:"retention_hours"`
		NotifyEnabled  bool `json:"notify_enabled"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if got.RetentionHours != 3 || !got.NotifyEnabled {
		t.Fatalf("unexpected payload: %+v", got)
	}
}
