package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// rateLimiter is a simple in-memory token-bucket rate limiter keyed by IP.
// It is sufficient for a single-instance deploy; behind a load balancer a
// shared store (Redis) would be required.
type rateLimiter struct {
	mu       sync.Mutex
	requests int
	window   time.Duration
	buckets  map[string]*bucket
}

type bucket struct {
	count    int
	resetAt  time.Time
}

// newRateLimiter creates a rate limiter allowing `requests` hits per `window`.
func newRateLimiter(requests int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: requests,
		window:   window,
		buckets:  make(map[string]*bucket),
	}
	// Sweep stale buckets periodically.
	go rl.sweepLoop()
	return rl
}

func (rl *rateLimiter) sweepLoop() {
	t := time.NewTicker(rl.window)
	defer t.Stop()
	for range t.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.buckets {
			if now.After(b.resetAt) {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[ip]
	if !ok || now.After(b.resetAt) {
		rl.buckets[ip] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.requests {
		return false
	}
	b.count++
	return true
}

// RateLimit limits each client IP to `requests` hits per `window`.
// It applies a 429 response with a JSON body when the limit is exceeded.
func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	rl := newRateLimiter(requests, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if !rl.allow(ip) {
				log.Warn().Str("ip", ip).Str("path", r.URL.Path).Msg("rate limit exceeded")
				w.Header().Set("Retry-After", "60")
				respondJSON(w, http.StatusTooManyRequests, map[string]string{
					"error": "Too many requests. Please slow down.",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByKey is a variant keyed by an arbitrary string (e.g. user ID
// extracted from the JWT claims).  Useful for per-user limits on heavy
// endpoints like quiz start or report-card PDF generation.
func RateLimitByKey(requests int, window time.Duration, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	rl := newRateLimiter(requests, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				key = realIP(r)
			}
			if !rl.allow(key) {
				respondJSON(w, http.StatusTooManyRequests, map[string]string{
					"error": "Too many requests.",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// realIP extracts the best-guess client IP from the request, honouring
// X-Forwarded-For when present (Cloudflare/Railway proxy chain).
func realIPMiddleware(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		// Take the left-most entry — the original client.
		if i := indexByte(v, ','); i >= 0 {
			return trimSpace(v[:i])
		}
		return trimSpace(v)
	}
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return trimSpace(v)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// thin wrappers to avoid pulling in strings/bytes here
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}