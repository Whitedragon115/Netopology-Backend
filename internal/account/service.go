package account

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
)

type OAuthLinkStarter interface {
	StartLink(ctx context.Context, provider string, userID uuid.UUID) (string, http.Cookie, error)
}

type Service struct {
	db          *pg.DB
	linkStarter OAuthLinkStarter
}

func NewService(db *pg.DB) *Service { return &Service{db: db} }

func (s *Service) SetOAuthLinkStarter(starter OAuthLinkStarter) { s.linkStarter = starter }

// GetMe implements permissions.md §3 me.read: any signed-in account may read itself.
func (s *Service) GetMe(ctx context.Context) (apimodel.Account, error) {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return apimodel.Account{}, err
	}
	user, err := s.db.Q().GetUserByID(ctx, p.UserID)
	if err != nil {
		return apimodel.Account{}, fmt.Errorf("read own account: %w", err)
	}
	providers, err := s.db.Q().ListIdentityProviders(ctx, p.UserID)
	if err != nil {
		return apimodel.Account{}, fmt.Errorf("read own identities: %w", err)
	}
	return apimodel.AccountFrom(user, providers), nil
}
