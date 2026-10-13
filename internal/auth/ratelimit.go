package auth

import (
	"math"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// loginLimiter bounds attempts per client IP and per case-insensitive username.
// The clock is injectable so refill and Retry-After behavior can be tested without sleeping.
type loginLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	byIP     map[string]bucket
	byUser   map[string]bucket
	lastTrim time.Time
}

func newLoginLimiter(now func() time.Time) *loginLimiter {
	return &loginLimiter{now: now, byIP: make(map[string]bucket), byUser: make(map[string]bucket)}
}

func (l *loginLimiter) allow(ip, username string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastTrim) > time.Hour {
		for key, b := range l.byIP {
			if now.Sub(b.last) > time.Hour {
				delete(l.byIP, key)
			}
		}
		for key, b := range l.byUser {
			if now.Sub(b.last) > time.Hour {
				delete(l.byUser, key)
			}
		}
		l.lastTrim = now
	}
	user := strings.ToLower(username)
	// An absent IP is one shared bucket, so requests without an address cannot bypass the limit.
	ipBucket, ipRetry := refill(l.byIP[ip], now, 20, time.Minute)
	userBucket, userRetry := refill(l.byUser[user], now, 10, 15*time.Minute)
	if ipRetry > 0 || userRetry > 0 {
		l.byIP[ip], l.byUser[user] = ipBucket, userBucket
		return max(ipRetry, userRetry)
	}
	ipBucket.tokens--
	userBucket.tokens--
	l.byIP[ip], l.byUser[user] = ipBucket, userBucket
	return 0
}

func refill(b bucket, now time.Time, capacity float64, period time.Duration) (bucket, int) {
	if b.last.IsZero() {
		b = bucket{tokens: capacity, last: now}
	} else {
		b.tokens = math.Min(capacity, b.tokens+now.Sub(b.last).Seconds()*capacity/period.Seconds())
		b.last = now
	}
	if b.tokens < 1 {
		return b, max(1, int(math.Ceil((1-b.tokens)*period.Seconds()/capacity)))
	}
	return b, 0
}
