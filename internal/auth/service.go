package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/config"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type Service struct {
	db       *pg.DB
	config   *config.Config
	sessions *Sessions
	limiter  *loginLimiter
}

func NewService(db *pg.DB, cfg *config.Config, sessions *Sessions) *Service {
	return &Service{db: db, config: cfg, sessions: sessions, limiter: newLoginLimiter(time.Now)}
}

// Providers is public (permissions.md §3 auth.providers).
func (s *Service) Providers(_ context.Context) AuthProviders {
	out := AuthProviders{Password: true, Providers: []ProviderOption{}}
	if s.config.GoogleEnabled() {
		out.Providers = append(out.Providers, ProviderOption{ID: "google", Name: "Google"})
	}
	if s.config.DiscordEnabled() {
		out.Providers = append(out.Providers, ProviderOption{ID: "discord", Name: "Discord"})
	}
	return out
}

// Login is public (permissions.md §3 auth.login). All bad credentials have one response.
func (s *Service) Login(ctx context.Context, in LoginRequest) (apimodel.Account, http.Cookie, error) {
	client := httpx.ClientFrom(ctx)
	ip := ""
	if client.IP.IsValid() {
		ip = client.IP.String()
	}
	if retry := s.limiter.allow(ip, in.Username); retry > 0 {
		if err := s.failedLogin(ctx, nil, "rate_limited"); err != nil {
			return apimodel.Account{}, http.Cookie{}, err
		}
		return apimodel.Account{}, http.Cookie{}, apperr.RateLimited(retry)
	}

	user, err := s.db.Q().GetUserByUsername(ctx, in.Username)
	if pg.IsNotFound(err) {
		crypto.DummyVerify(in.Password)
		return s.invalidLogin(ctx, nil)
	}
	if err != nil {
		return apimodel.Account{}, http.Cookie{}, fmt.Errorf("find login account: %w", err)
	}
	if user.PasswordHash == nil {
		crypto.DummyVerify(in.Password)
		return s.invalidLogin(ctx, &user.ID)
	}
	valid, err := crypto.VerifyPassword(in.Password, *user.PasswordHash)
	if err != nil {
		return apimodel.Account{}, http.Cookie{}, fmt.Errorf("verify login password: %w", err)
	}
	if !valid || user.Status != "active" {
		return s.invalidLogin(ctx, &user.ID)
	}

	var replacement *string
	if crypto.NeedsRehash(*user.PasswordHash) {
		hash, err := crypto.HashPassword(in.Password) // slow work before opening a transaction
		if err != nil {
			return apimodel.Account{}, http.Cookie{}, err
		}
		replacement = &hash
	}
	var account apimodel.Account
	var cookie http.Cookie
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		current, err := q.GetUserByID(ctx, user.ID)
		if err != nil {
			return err
		}
		if current.Status != "active" || current.PasswordHash == nil || *current.PasswordHash != *user.PasswordHash {
			return apperr.Unauthenticated("auth.invalid_credentials", "Invalid username or password.")
		}
		if replacement != nil {
			changed, err := q.RehashUserPassword(ctx, dbgen.RehashUserPasswordParams{
				UserID: user.ID, OldHash: user.PasswordHash, NewHash: replacement,
			})
			if err != nil {
				return err
			}
			if changed != 1 {
				return apperr.Unauthenticated("auth.invalid_credentials", "Invalid username or password.")
			}
		}
		cookie, err = s.sessions.Start(ctx, q, user.ID, in.Remember)
		if err != nil {
			return err
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.AuthLogin, TargetType: "user", TargetID: &user.ID, ActorID: &user.ID,
		}); err != nil {
			return err
		}
		current, err = q.GetUserByID(ctx, user.ID)
		if err != nil {
			return err
		}
		providers, err := q.ListIdentityProviders(ctx, user.ID)
		if err != nil {
			return err
		}
		account = apimodel.AccountFrom(current, providers)
		return nil
	})
	if ae, ok := apperr.As(err); ok && ae.Code == "auth.invalid_credentials" {
		return s.invalidLogin(ctx, &user.ID)
	}
	return account, cookie, err
}

func (s *Service) invalidLogin(ctx context.Context, userID *uuid.UUID) (apimodel.Account, http.Cookie, error) {
	if err := s.failedLogin(ctx, userID, "invalid_credentials"); err != nil {
		return apimodel.Account{}, http.Cookie{}, err
	}
	return apimodel.Account{}, http.Cookie{}, apperr.Unauthenticated("auth.invalid_credentials", "Invalid username or password.")
}

func (s *Service) failedLogin(ctx context.Context, userID *uuid.UUID, reason string) error {
	return s.db.Tx(ctx, func(q *dbgen.Queries) error {
		return auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.AuthLogin, TargetType: "user", TargetID: userID, ActorID: userID,
			Metadata: map[string]any{"reason": reason},
		})
	})
}

// Logout requires an account and removes just this session (permissions.md §3 auth.logout).
func (s *Service) Logout(ctx context.Context) (http.Cookie, error) {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return http.Cookie{}, err
	}
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		if err := q.DeleteSession(ctx, p.SessionID); err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.AuthLogout, TargetType: "user", TargetID: &p.UserID,
		})
	})
	if err != nil {
		return http.Cookie{}, err
	}
	return s.sessions.Cookies.Clear(httpx.SessionCookieName, "/"), nil
}
