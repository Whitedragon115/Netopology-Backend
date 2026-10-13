package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// Sessions issues and ends sessions. Other features (setup, OAuth callback) use it through the
// SessionStarter interface instead of importing this package's internals.
type Sessions struct {
	TTL     time.Duration
	Cookies httpx.Cookies
}

// RememberTTL is the minimum inactivity lifetime for remembered sessions.
const RememberTTL = 30 * 24 * time.Hour

func sessionLifetime(ttl time.Duration, remember bool) time.Duration {
	if remember && ttl < RememberTTL {
		return RememberTTL
	}
	return ttl
}

// Start creates a session row inside the caller's transaction and returns the cookie to set.
// The raw token only ever exists in the cookie; the table stores its SHA-256.
func (s *Sessions) Start(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, remember bool) (http.Cookie, error) {
	token, err := crypto.NewToken()
	if err != nil {
		return http.Cookie{}, err
	}
	ttl := sessionLifetime(s.TTL, remember)
	c := httpx.ClientFrom(ctx)
	params := dbgen.CreateSessionParams{UserID: userID, TokenHash: crypto.HashToken(token), ExpiresAt: time.Now().Add(ttl), Remember: remember}
	if c.IP.IsValid() {
		ip := c.IP
		params.Ip = &ip
	}
	if c.UserAgent != "" {
		ua := c.UserAgent
		params.UserAgent = &ua
	}
	if _, err := q.CreateSession(ctx, params); err != nil {
		return http.Cookie{}, fmt.Errorf("create session: %w", err)
	}
	if err := q.SetUserLastLogin(ctx, userID); err != nil {
		return http.Cookie{}, err
	}
	return s.Cookies.Session(token, ttl, remember), nil
}
