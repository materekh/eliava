package server

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rateLimiter is an in-memory fixed-window limiter. It is per process, which is enough for a single instance.
type rateLimiter struct {
	mu   sync.Mutex
	hits map[string]*rateWindow
}

type rateWindow struct {
	start time.Time
	n     int
}

func newRateLimiter() *rateLimiter { return &rateLimiter{hits: map[string]*rateWindow{}} }

func (l *rateLimiter) allow(key string, max int, per time.Duration) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.hits) > 50_000 {
		for k, w := range l.hits {
			if now.Sub(w.start) > 24*time.Hour {
				delete(l.hits, k)
			}
		}
	}
	w := l.hits[key]
	if w == nil || now.Sub(w.start) >= per {
		l.hits[key] = &rateWindow{start: now, n: 1}
		return true
	}
	if w.n >= max {
		return false
	}
	w.n++
	return true
}

// limit returns a 429 error when key has been used more than max times within per.
func (s *Server) limit(key string, max int, per time.Duration) error {
	if s.cfg.DisableRateLimit || s.rl.allow(key, max, per) {
		return nil
	}
	return errStatus(http.StatusTooManyRequests, "Too many attempts. Please wait and try again later.")
}

// clientIP prefers the proxy-provided address (Render sits behind Cloudflare) over the socket address.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) limitUser(u *User, action string, max int, per time.Duration) error {
	return s.limit(action+":"+strconv.FormatInt(u.ID, 10), max, per)
}
