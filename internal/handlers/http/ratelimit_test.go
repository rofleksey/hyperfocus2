package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVisitorLimiterAllowsUpToBudget(t *testing.T) {
	l := newVisitorLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		ok, _ := l.Allow("1.2.3.4")
		if !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	ok, retry := l.Allow("1.2.3.4")
	if ok {
		t.Fatal("4th request should be denied")
	}
	if retry <= 0 {
		t.Fatalf("expected positive retry duration, got %v", retry)
	}
	// Other keys are unaffected.
	if ok, _ := l.Allow("5.6.7.8"); !ok {
		t.Fatal("different key should be allowed")
	}
}

func TestRateLimitMiddlewareReturns429(t *testing.T) {
	l := newVisitorLimiter(1, time.Minute)
	h := rateLimitMiddleware(nil, "test", l)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r1 := httptest.NewRequest(http.MethodGet, "/api/subscribe", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: got %d want 200", w1.Code)
	}

	r2 := httptest.NewRequest(http.MethodGet, "/api/subscribe", nil)
	r2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: got %d want 429", w2.Code)
	}
	if ra := w2.Header().Get("Retry-After"); ra == "" {
		t.Fatal("expected Retry-After header on 429")
	}
	if !strings.Contains(w2.Body.String(), "rate limit exceeded") {
		t.Fatalf("unexpected body: %s", w2.Body.String())
	}
}

func TestClientIPPrefersXFFBehindProxy(t *testing.T) {
	// Rightmost XFF entry is what the trusted proxy actually saw; earlier
	// entries may be client-spoofed.
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:9000"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.7")
	if got := clientIP(r); got != "203.0.113.7" {
		t.Fatalf("got %q want 203.0.113.7", got)
	}

	// Direct public peer: XFF must be ignored.
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "198.51.100.5:9000"
	r2.Header.Set("X-Forwarded-For", "203.0.113.7")
	if got := clientIP(r2); got != "198.51.100.5" {
		t.Fatalf("got %q want 198.51.100.5", got)
	}
}
