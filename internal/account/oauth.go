package account

import (
	"context"
	"net/http"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// LinkMyIdentity starts a provider redirect only for this signed-in account.
func (s *Service) LinkMyIdentity(ctx context.Context, provider string) (string, http.Cookie, error) {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return "", http.Cookie{}, err
	}
	if provider != "google" && provider != "discord" {
		return "", http.Cookie{}, apperr.ErrNotFound()
	}
	if _, err := s.db.Q().GetIdentityForUserProvider(ctx, dbgen.GetIdentityForUserProviderParams{
		UserID: p.UserID, Provider: provider,
	}); err == nil {
		return "", http.Cookie{}, apperr.Conflict("auth.identity_in_use", "This account already has that provider linked.")
	} else if !pg.IsNotFound(err) {
		return "", http.Cookie{}, err
	}
	if s.linkStarter == nil {
		return "", http.Cookie{}, apperr.ErrNotFound()
	}
	return s.linkStarter.StartLink(ctx, provider, p.UserID)
}
