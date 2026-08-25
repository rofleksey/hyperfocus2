package http

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"hyperfocus/internal/pkg/httputil"
)

// visitorLimiter wraps golang.org/x/time/rate limiters keyed per client IP,
// with lazy cleanup of stale entries.
type visitorLimiter struct {
	mu       sync.Mutex
	limit    rate.Limit
	burst    int
	visitors map[string]*visitorEntry
}

type visitorEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newVisitorLimiter(max int, per time.Duration) *visitorLimiter {
	return &visitorLimiter{
		limit:    rate.Every(per / time.Duration(max)),
		burst:    max,
		visitors: make(map[string]*visitorEntry),
	}
}

// Allow reports whether key may proceed. When denied it returns the delay
// until the next token is available.
func (v *visitorLimiter) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(v.visitors) > 10_000 {
		for k, e := range v.visitors {
			if now.Sub(e.lastSeen) >= time.Hour {
				delete(v.visitors, k)
			}
		}
	}

	e, ok := v.visitors[key]
	if !ok {
		e = &visitorEntry{limiter: rate.NewLimiter(v.limit, v.burst)}
		v.visitors[key] = e
	}
	e.lastSeen = now

	res := e.limiter.Reserve()
	if !res.OK() {
		return false, time.Hour
	}
	delay := res.Delay()
	return delay == 0, delay
}

// clientIP extracts the per-request client key. X-Forwarded-For is honored
// only when the direct peer is a loopback/private address (i.e. traffic
// arriving through a local reverse proxy such as Traefik); with a single
// trusted hop the rightmost XFF entry is the address the proxy actually saw.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" && (ip.IsLoopback() || ip.IsPrivate()) {
		parts := strings.Split(xff, ",")
		if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
			if parsed := net.ParseIP(last); parsed != nil {
				return parsed.String()
			}
		}
	}
	return ip.String()
}

// rateLimitMiddleware rejects requests over the limiter's budget with 429 and
// a Retry-After header.
func rateLimitMiddleware(log *slog.Logger, name string, l *visitorLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if ok, retryAfter := l.Allow(key); !ok {
				if log != nil {
					log.Warn("http: rate limit exceeded",
						slog.String("limiter", name),
						slog.String("remote", key),
						slog.String("path", r.URL.Path),
					)
				}
				if retryAfter < time.Second {
					retryAfter = time.Second
				}
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
				httputil.Problem(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
