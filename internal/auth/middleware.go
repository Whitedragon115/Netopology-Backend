package auth

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/logging"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/settings"
)

// Middleware resolves the session cookie to an authz.Principal for every request. It never
// rejects a request: no cookie, an unknown, expired or disabled session all mean "anonymous",
// and each operation decides whether that is enough (see authz scopes).
//
// Guests (share-link cookie) are resolved by internal/share, only on /share routes.
func Middleware(db *pg.DB, ttl time.Duration, cookies httpx.Cookies, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if oauth, err := r.Cookie(httpx.OAuthCookieName); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), oauthCookieKey{}, oauth.Value))
			}
			ck, err := r.Cookie(httpx.SessionCookieName)
			if err != nil || ck.Value == "" || len(ck.Value) > 100 {
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()
			row, err := db.Q().GetSessionPrincipal(ctx, crypto.HashToken(ck.Value))
			if err != nil {
				if !pg.IsNotFound(err) {
					log.ErrorContext(ctx, "resolve session", "err", err)
				}
				next.ServeHTTP(w, r)
				return
			}
			// sliding expiry, written at most once a minute per session
			if time.Since(row.LastSeenAt) > time.Minute {
				baseTTL, settingErr := settings.SessionTTL(ctx, db.Q(), ttl)
				if settingErr != nil {
					log.WarnContext(ctx, "read session TTL", "err", settingErr)
				} else {
					lifetime := sessionLifetime(baseTTL, row.Remember)
					if err := db.Q().TouchSession(ctx, dbgen.TouchSessionParams{ID: row.SessionID, ExpiresAt: time.Now().Add(lifetime)}); err != nil {
						log.WarnContext(ctx, "touch session", "err", err)
					} else if row.Remember {
						// The browser expiry must slide alongside the server-side expiry.
						cookie := cookies.Session(ck.Value, lifetime, true)
						http.SetCookie(w, &cookie)
					}
				}
			}
			p := &authz.Principal{
				UserID:       row.UserID,
				Username:     row.Username,
				InstanceRole: authz.InstanceRole(row.InstanceRole),
				SessionID:    row.SessionID,
			}
			ctx = authz.WithPrincipal(ctx, p)
			ctx = logging.With(ctx, slog.String("user_id", p.UserID.String()))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
