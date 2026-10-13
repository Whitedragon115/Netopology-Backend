package account

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func accountFrom(ctx context.Context, q *dbgen.Queries, user dbgen.User) (apimodel.Account, error) {
	providers, err := q.ListIdentityProviders(ctx, user.ID)
	if err != nil {
		return apimodel.Account{}, err
	}
	return apimodel.AccountFrom(user, providers), nil
}

// UpdateMe implements permissions.md rule 9: a signed-in user controls only their own profile.
func (s *Service) UpdateMe(ctx context.Context, in MeUpdate) (apimodel.Account, error) {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return apimodel.Account{}, err
	}
	if in.DisplayName == nil && !in.EmailSet {
		return apimodel.Account{}, apperr.Invalid(apperr.CodeValidation, "Change at least one field.")
	}
	var out apimodel.Account
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		name := ""
		if in.DisplayName != nil {
			name = *in.DisplayName
		}
		user, err := q.UpdateUserFields(ctx, dbgen.UpdateUserFieldsParams{
			UserID: p.UserID, SetDisplayName: in.DisplayName != nil, DisplayName: name,
			SetEmail: in.EmailSet, Email: in.Email,
		})
		if constraint, ok := pg.UniqueViolation(err); ok && constraint == "users_email_key" {
			return apperr.Conflict("user.email_taken", "That email is already used.")
		}
		if err != nil {
			return err
		}
		fields := []string{}
		if in.DisplayName != nil {
			fields = append(fields, "displayName")
		}
		if in.EmailSet {
			fields = append(fields, "email")
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.MeUpdate, TargetType: "user", TargetID: &p.UserID,
			Metadata: map[string]any{"fields": fields},
		}); err != nil {
			return err
		}
		out, err = accountFrom(ctx, q, user)
		return err
	})
	return out, err
}

// ChangeMyPassword checks the current method before writing and keeps only this session.
func (s *Service) ChangeMyPassword(ctx context.Context, in PasswordChange) error {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return err
	}
	current, err := s.db.Q().GetUserByID(ctx, p.UserID)
	if err != nil {
		return err
	}
	if current.PasswordHash != nil {
		valid, err := crypto.VerifyPassword(in.CurrentPassword, *current.PasswordHash)
		if err != nil {
			return fmt.Errorf("verify current password: %w", err)
		}
		if !valid {
			return apperr.Unauthenticated("auth.invalid_credentials", "Current password is incorrect.")
		}
	}
	hash, err := crypto.HashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	return s.db.Tx(ctx, func(q *dbgen.Queries) error {
		locked, err := q.GetUserForUpdate(ctx, p.UserID)
		if err != nil {
			return err
		}
		if locked.PasswordHash == nil {
			// permissions.md rule 9: a provider-only account needs a recent provider sign-in.
			recent, err := q.HasRecentProviderSignIn(ctx, dbgen.HasRecentProviderSignInParams{SessionID: p.SessionID, UserID: p.UserID})
			if err != nil {
				return err
			}
			if !recent {
				return apperr.Forbidden(apperr.CodeRoleInsufficient, "Sign in with a provider again before setting a password.")
			}
		} else if current.PasswordHash == nil || *locked.PasswordHash != *current.PasswordHash {
			return apperr.Unauthenticated("auth.invalid_credentials", "Current password changed. Try again.")
		}
		if err := q.SetUserPassword(ctx, dbgen.SetUserPasswordParams{ID: p.UserID, PasswordHash: &hash}); err != nil {
			return err
		}
		if _, err := q.DeleteOtherUserSessions(ctx, dbgen.DeleteOtherUserSessionsParams{UserID: p.UserID, ID: p.SessionID}); err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{Action: authz.MePassword, TargetType: "user", TargetID: &p.UserID})
	})
}

// ListMyIdentities never includes provider subjects or tokens.
func (s *Service) ListMyIdentities(ctx context.Context) ([]Identity, bool, error) {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return nil, false, err
	}
	user, err := s.db.Q().GetUserByID(ctx, p.UserID)
	if err != nil {
		return nil, false, err
	}
	rows, err := s.db.Q().ListIdentities(ctx, p.UserID)
	if err != nil {
		return nil, false, err
	}
	items := make([]Identity, 0, len(rows))
	for _, row := range rows {
		items = append(items, identityFrom(row))
	}
	return items, user.PasswordHash != nil, nil
}

func (s *Service) UnlinkMyIdentity(ctx context.Context, id uuid.UUID) error {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return err
	}
	return s.db.Tx(ctx, func(q *dbgen.Queries) error {
		user, err := q.GetUserForUpdate(ctx, p.UserID)
		if err != nil {
			return err
		}
		if _, err := q.GetIdentityForUser(ctx, dbgen.GetIdentityForUserParams{ID: id, UserID: p.UserID}); pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		} else if err != nil {
			return err
		}
		count, err := q.CountUserIdentities(ctx, p.UserID)
		if err != nil {
			return err
		}
		if user.PasswordHash == nil && count <= 1 {
			return apperr.Conflict(apperr.CodeLastSignInMethod, "Keep at least one sign-in method.")
		}
		if _, err := q.UnlinkIdentity(ctx, dbgen.UnlinkIdentityParams{ID: id, UserID: p.UserID}); err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{Action: authz.MeIdentityUnlink, TargetType: "identity", TargetID: &id})
	})
}
