package scans

import (
	"net/http"
	"sync"
	"time"

	"github.com/ivantan02/fitness-tracker-app/internal/auth"
)

// extractRateLimiter caps how many extraction calls a single user can make
// per day, so one user can't burn through Gemini's shared 250 req/day quota
// and lock everyone else out. It's an in-memory counter reset once a day —
// fine for this app's single-process, free-tier scale; a restart just resets
// everyone's count early, which is an acceptable trade-off at this size.
type extractRateLimiter struct {
	maxPerDay int

	mu        sync.Mutex
	counts    map[string]int
	windowEnd time.Time
}

func newExtractRateLimiter(maxPerDay int) *extractRateLimiter {
	return &extractRateLimiter{
		maxPerDay: maxPerDay,
		counts:    make(map[string]int),
		windowEnd: time.Now().Add(24 * time.Hour),
	}
}

// remaining reports whether userID has quota left to attempt an extraction
// call right now, without consuming it. Call record after a call actually
// succeeds — a failed Gemini call (e.g. an upstream outage) shouldn't cost
// the user their daily quota.
func (l *extractRateLimiter) remaining(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.resetIfExpired()
	return l.counts[userID] < l.maxPerDay
}

// record counts a successful extraction against userID's daily quota.
func (l *extractRateLimiter) record(userID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.resetIfExpired()
	l.counts[userID]++
}

func (l *extractRateLimiter) resetIfExpired() {
	if time.Now().After(l.windowEnd) {
		l.counts = make(map[string]int)
		l.windowEnd = time.Now().Add(24 * time.Hour)
	}
}

func (l *extractRateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())
		if !l.remaining(userID) {
			http.Error(w, "daily extraction limit reached, please use manual entry or try again tomorrow", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
