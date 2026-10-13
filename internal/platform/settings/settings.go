// Package settings reads instance settings shared by account and authentication features.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

const (
	SessionTTLKey    = "sessionTtlHours"
	OAuthAutoLinkKey = "oauthAutoLinkByEmail"
)

func SessionTTL(ctx context.Context, q *dbgen.Queries, fallback time.Duration) (time.Duration, error) {
	raw, err := q.GetInstanceSetting(ctx, SessionTTLKey)
	if pg.IsNotFound(err) {
		return fallback, nil
	}
	if err != nil {
		return 0, err
	}
	var hours int
	if err := json.Unmarshal(raw, &hours); err != nil {
		return 0, fmt.Errorf("decode session TTL: %w", err)
	}
	if hours < 1 || hours > 8760 {
		return 0, fmt.Errorf("session TTL out of range")
	}
	return time.Duration(hours) * time.Hour, nil
}

func OAuthAutoLink(ctx context.Context, q *dbgen.Queries) (bool, error) {
	raw, err := q.GetInstanceSetting(ctx, OAuthAutoLinkKey)
	if pg.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	var enabled bool
	if err := json.Unmarshal(raw, &enabled); err != nil {
		return false, fmt.Errorf("decode OAuth auto-link: %w", err)
	}
	return enabled, nil
}
