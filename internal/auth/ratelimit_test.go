package auth

import (
	"testing"
	"time"
)

func TestLoginLimiter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := newLoginLimiter(func() time.Time { return now })
	for i := range 10 {
		if retry := l.allow("192.0.2.1", "Alice"); retry != 0 {
			t.Fatalf("attempt %d retry = %d", i, retry)
		}
	}
	if retry := l.allow("192.0.2.2", "ALICE"); retry != 90 {
		t.Fatalf("username limit retry = %d, want 90", retry)
	}
	now = now.Add(90 * time.Second)
	if retry := l.allow("192.0.2.2", "alice"); retry != 0 {
		t.Fatalf("refilled username retry = %d", retry)
	}
	for i := range 20 {
		if retry := l.allow("198.51.100.1", string(rune('a'+i))); retry != 0 {
			t.Fatalf("IP attempt %d retry = %d", i, retry)
		}
	}
	if retry := l.allow("198.51.100.1", "new-user"); retry != 3 {
		t.Fatalf("IP limit retry = %d, want 3", retry)
	}
}

func TestSessionLifetime(t *testing.T) {
	for _, tc := range []struct {
		base     time.Duration
		remember bool
		want     time.Duration
	}{
		{24 * time.Hour, false, 24 * time.Hour},
		{24 * time.Hour, true, RememberTTL},
		{40 * 24 * time.Hour, true, 40 * 24 * time.Hour},
	} {
		if got := sessionLifetime(tc.base, tc.remember); got != tc.want {
			t.Errorf("sessionLifetime(%v, %v) = %v, want %v", tc.base, tc.remember, got, tc.want)
		}
	}
}
