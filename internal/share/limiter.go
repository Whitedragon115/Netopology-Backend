package share

import (
	"math"
	"sync"
	"time"
)

type (
	bucket struct {
		tokens float64
		last   time.Time
	}
	limiter struct {
		mu    sync.Mutex
		links map[string]bucket
		ips   map[string]bucket
		trim  time.Time
	}
)

func newLimiter() *limiter { return &limiter{links: map[string]bucket{}, ips: map[string]bucket{}} }
func (l *limiter) allow(link, ip string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.trim) > time.Hour {
		for _, buckets := range []map[string]bucket{l.links, l.ips} {
			for key, b := range buckets {
				if now.Sub(b.last) > time.Hour {
					delete(buckets, key)
				}
			}
		}
		l.trim = now
	}
	a, ra := refill(l.links[link], now, 10, 15*time.Minute)
	b, rb := refill(l.ips[ip], now, 20, time.Minute)
	if ra > 0 || rb > 0 {
		l.links[link], l.ips[ip] = a, b
		return max(ra, rb)
	}
	a.tokens--
	b.tokens--
	l.links[link], l.ips[ip] = a, b
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
