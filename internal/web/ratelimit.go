package web

import (
	"sync"
	"time"
)

// rateLimiter is a per-key sliding-window limiter: at most limit calls to
// allow within window. Not swept of stale keys — acceptable for a login
// endpoint's IP set at current scale; revisit if it ever shows up as a
// real memory concern.
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-l.window)

	var kept []time.Time
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}

	l.attempts[key] = append(kept, time.Now())
	return true
}
