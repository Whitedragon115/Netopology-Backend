package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/settings"
)

type oauthState struct {
	State    string    `json:"state"`
	Verifier string    `json:"verifier"`
	Provider string    `json:"provider"`
	Mode     string    `json:"mode"`
	UserID   uuid.UUID `json:"userId,omitempty"`
	ReturnTo string    `json:"returnTo"`
}

type OAuthResult struct {
	Location string
	Cookies  []http.Cookie
}

type oauthCookieKey struct{}

func oauthCookieFrom(ctx context.Context) string {
	value, _ := ctx.Value(oauthCookieKey{}).(string)
	return value
}

func safeReturnTo(value string) string {
	if value == "" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") {
		return "/"
	}
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil ||
		strings.HasPrefix(u.Path, "//") || strings.Contains(u.Path, "\\") {
		return "/"
	}
	return value
}

func (s *Service) oauthRedirect(provider string) string {
	return s.config.PublicURL + "/api/v1/auth/oauth/" + provider + "/callback"
}

// StartSignIn implements the public auth.oauth_start operation.
func (s *Service) StartSignIn(_ context.Context, provider, returnTo string) (string, http.Cookie, error) {
	return s.beginOAuth(provider, "login", uuid.Nil, returnTo)
}

// StartLink is called only after the account service authorizes me.identity_link.
func (s *Service) StartLink(_ context.Context, provider string, userID uuid.UUID) (string, http.Cookie, error) {
	return s.beginOAuth(provider, "link", userID, "/")
}

func (s *Service) beginOAuth(provider, mode string, userID uuid.UUID, returnTo string) (string, http.Cookie, error) {
	client := s.providers[provider]
	if client == nil {
		return "", http.Cookie{}, apperr.ErrNotFound()
	}
	state, err := crypto.NewToken()
	if err != nil {
		return "", http.Cookie{}, err
	}
	value := oauthState{
		State: state, Verifier: oauth2.GenerateVerifier(), Provider: provider,
		Mode: mode, UserID: userID, ReturnTo: safeReturnTo(returnTo),
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", http.Cookie{}, err
	}
	cookie := s.sessions.Cookies.OAuth(s.signer.Sign(raw, 10*time.Minute))
	return client.Authorize(state, value.Verifier, s.oauthRedirect(provider)), cookie, nil
}

func (s *Service) finishFailure(code string, cookies []http.Cookie) OAuthResult {
	return OAuthResult{Location: s.config.PublicURL + "/login?error=" + url.QueryEscape(code), Cookies: cookies}
}

// FinishOAuth always returns a browser redirect. No provider token or verifier reaches the URL.
func (s *Service) FinishOAuth(ctx context.Context, provider, code, state, providerError, signedCookie string) OAuthResult {
	cookies := []http.Cookie{s.sessions.Cookies.Clear(httpx.OAuthCookieName, "/api/v1")}
	raw, err := s.signer.Verify(signedCookie)
	if err != nil {
		return s.finishFailure("invalid_state", cookies)
	}
	var saved oauthState
	if err := json.Unmarshal(raw, &saved); err != nil || saved.Provider != provider || saved.State == "" ||
		subtle.ConstantTimeCompare([]byte(saved.State), []byte(state)) != 1 ||
		(saved.Mode != "login" && saved.Mode != "link") {
		return s.finishFailure("invalid_state", cookies)
	}
	if providerError != "" {
		if providerError == "access_denied" {
			return s.finishFailure("access_denied", cookies)
		}
		return s.finishFailure("provider_error", cookies)
	}
	if saved.Mode == "link" {
		p, err := authz.RequireSignedIn(ctx)
		if err != nil || saved.UserID == uuid.Nil || p.UserID != saved.UserID {
			return s.finishFailure("sign_in_required", cookies)
		}
	}
	client := s.providers[provider]
	if client == nil || code == "" {
		return s.finishFailure("provider_error", cookies)
	}
	profile, err := client.Profile(ctx, code, saved.Verifier, s.oauthRedirect(provider))
	if err != nil || profile.Subject == "" {
		return s.finishFailure("provider_error", cookies)
	}
	if saved.Mode == "link" {
		if err := s.finishLink(ctx, provider, saved.UserID, profile); err != nil {
			return s.finishFailure(oauthErrorCode(err), cookies)
		}
		return OAuthResult{Location: s.config.PublicURL + saved.ReturnTo, Cookies: cookies}
	}
	session, err := s.finishSignIn(ctx, provider, profile)
	if err != nil {
		return s.finishFailure(oauthErrorCode(err), cookies)
	}
	cookies = append(cookies, session)
	return OAuthResult{Location: s.config.PublicURL + saved.ReturnTo, Cookies: cookies}
}

func oauthErrorCode(err error) string {
	if ae, ok := apperr.As(err); ok {
		switch ae.Code {
		case apperr.CodeIdentityNotLinked:
			return "identity_not_linked"
		case "auth.identity_in_use":
			return "identity_in_use"
		case apperr.CodeNotSignedIn:
			return "sign_in_required"
		}
	}
	return "provider_error"
}

func (s *Service) finishSignIn(ctx context.Context, provider string, profile OAuthProfile) (http.Cookie, error) {
	var session http.Cookie
	err := s.db.Tx(ctx, func(q *dbgen.Queries) error {
		identity, err := q.GetIdentityByProviderSubject(ctx, dbgen.GetIdentityByProviderSubjectParams{
			Provider: provider, Subject: profile.Subject,
		})
		var user dbgen.User
		linkedNow := false
		if err == nil {
			user, err = q.GetUserByID(ctx, identity.UserID)
			if err != nil {
				return err
			}
			if user.Status != "active" {
				return apperr.Unauthenticated(apperr.CodeIdentityNotLinked, "Identity is not linked to an active account.")
			}
			if err := q.TouchIdentity(ctx, identity.ID); err != nil {
				return err
			}
		} else if pg.IsNotFound(err) {
			allowed, err := settings.OAuthAutoLink(ctx, q)
			if err != nil {
				return err
			}
			if !allowed || !profile.EmailVerified || profile.Email == nil || *profile.Email == "" {
				return apperr.Unauthenticated(apperr.CodeIdentityNotLinked, "Identity is not linked to an account.")
			}
			user, err = q.GetActiveUserByEmail(ctx, *profile.Email)
			if pg.IsNotFound(err) {
				return apperr.Unauthenticated(apperr.CodeIdentityNotLinked, "Identity is not linked to an account.")
			}
			if err != nil {
				return err
			}
			if _, err := q.GetIdentityForUserProvider(ctx, dbgen.GetIdentityForUserProviderParams{
				UserID: user.ID, Provider: provider,
			}); err == nil {
				return apperr.Unauthenticated(apperr.CodeIdentityNotLinked, "Identity is not linked to an account.")
			} else if !pg.IsNotFound(err) {
				return err
			}
			if _, err := q.LinkIdentity(ctx, dbgen.LinkIdentityParams{
				UserID: user.ID, Provider: provider,
				Subject: profile.Subject, Email: profile.Email, DisplayName: profile.DisplayName,
			}); err != nil {
				if _, ok := pg.UniqueViolation(err); ok {
					return apperr.Unauthenticated(apperr.CodeIdentityNotLinked, "Identity is not linked to an account.")
				}
				return err
			}
			linkedNow = true
		} else {
			return err
		}
		// OAuth sessions use a browser-session cookie (ARCHITECTURE §6.1).
		session, err = s.sessions.Start(ctx, q, user.ID, false)
		if err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.AuthOAuthCB, TargetType: "user", TargetID: &user.ID,
			ActorID: &user.ID, Metadata: map[string]any{"provider": provider, "autoLinked": linkedNow},
		})
	})
	return session, err
}

func (s *Service) finishLink(ctx context.Context, provider string, userID uuid.UUID, profile OAuthProfile) error {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return err
	}
	if userID == uuid.Nil || p.UserID != userID {
		return apperr.Unauthenticated(apperr.CodeNotSignedIn, "Sign in again to link this account.")
	}
	return s.db.Tx(ctx, func(q *dbgen.Queries) error {
		user, err := q.GetUserForUpdate(ctx, userID)
		if err != nil {
			return err
		}
		if user.Status != "active" {
			return apperr.Unauthenticated(apperr.CodeNotSignedIn, "Sign in again to link this account.")
		}
		identity, err := q.GetIdentityByProviderSubject(ctx, dbgen.GetIdentityByProviderSubjectParams{Provider: provider, Subject: profile.Subject})
		if err == nil {
			if identity.UserID != userID {
				return apperr.Conflict("auth.identity_in_use", "That provider account is linked elsewhere.")
			}
			return nil
		}
		if !pg.IsNotFound(err) {
			return err
		}
		if _, err := q.GetIdentityForUserProvider(ctx, dbgen.GetIdentityForUserProviderParams{UserID: userID, Provider: provider}); err == nil {
			return apperr.Conflict("auth.identity_in_use", "This account already has that provider linked.")
		} else if !pg.IsNotFound(err) {
			return err
		}
		linked, err := q.LinkIdentity(ctx, dbgen.LinkIdentityParams{
			UserID: userID, Provider: provider,
			Subject: profile.Subject, Email: profile.Email, DisplayName: profile.DisplayName,
		})
		if _, ok := pg.UniqueViolation(err); ok {
			return apperr.Conflict("auth.identity_in_use", "That provider account is linked elsewhere.")
		}
		if err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.MeIdentityLink, TargetType: "identity", TargetID: &linked.ID,
			Metadata: map[string]any{"provider": provider},
		})
	})
}
