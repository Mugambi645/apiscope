// Package ratelimit is a per-key token bucket with idle-key eviction.
package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	buckets map[string]*bucket
	now     func() time.Time
}

func New(perMinute int, burst int) *Limiter {
	return &Limiter{
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		buckets: map[string]*bucket{},
		now:     time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Sweep drops buckets idle for longer than ttl. Call it periodically.
func (l *Limiter) Sweep(ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := l.now().Add(-ttl)
	for k, b := range l.buckets {
		if b.last.Before(cut) {
			delete(l.buckets, k)
		}
	}
}

// ClientIP prefers the first X-Forwarded-For entry only when trustProxy is set.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *Limiter) Middleware(trustProxy bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(ClientIP(r, trustProxy)) {
			w.Header().Set("Retry-After", "10")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
