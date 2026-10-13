package account

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func (s *Service) ListUsers(ctx context.Context, page httpx.PageParams, search string) (AccountPage, error) {
	if _, err := authz.RequireInstanceStaff(ctx); err != nil {
		return AccountPage{}, err
	}
	after := uuid.Nil
	if err := httpx.DecodeCursor(page.Cursor, &after); err != nil {
		return AccountPage{}, err
	}
	if page.Limit == 0 {
		page.Limit = 50
	}
	rows, err := s.db.Q().ListUsers(ctx, dbgen.ListUsersParams{After: after, Search: search, PageLimit: int32(page.Limit + 1)})
	if err != nil {
		return AccountPage{}, err
	}
	out := AccountPage{Items: []apimodel.Account{}}
	if len(rows) > page.Limit {
		out.NextCursor = httpx.EncodeCursor(rows[page.Limit-1].ID)
		rows = rows[:page.Limit]
	}
	for _, row := range rows {
		account, err := accountFrom(ctx, s.db.Q(), row)
		if err != nil {
			return AccountPage{}, err
		}
		out.Items = append(out.Items, account)
	}
	return out, nil
}

func (s *Service) CreateUser(ctx context.Context, in AccountCreate) (apimodel.Account, error) {
	p, err := authz.RequireInstanceStaff(ctx)
	if err != nil {
		return apimodel.Account{}, err
	}
	if in.Password == nil && (in.Email == nil || strings.TrimSpace(*in.Email) == "") {
		return apimodel.Account{}, apperr.Invalid(apperr.CodeValidation, "Give the account a password or an email address.")
	}
	role := in.InstanceRole
	if role == "" {
		role = string(authz.InstanceMember)
	}
	if role == string(authz.InstanceAdmin) && p.InstanceRole != authz.InstanceOwner {
		return apimodel.Account{}, apperr.ErrRoleInsufficient()
	}
	var hash *string
	if in.Password != nil {
		value, err := crypto.HashPassword(*in.Password)
		if err != nil {
			return apimodel.Account{}, err
		}
		hash = &value
	}
	var out apimodel.Account
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		user, err := q.CreateUser(ctx, dbgen.CreateUserParams{
			Username: in.Username, Email: in.Email, DisplayName: strings.TrimSpace(in.DisplayName),
			PasswordHash: hash, InstanceRole: role,
		})
		if constraint, ok := pg.UniqueViolation(err); ok {
			switch constraint {
			case "users_username_key":
				return apperr.Conflict("user.username_taken", "That username is already used.")
			case "users_email_key":
				return apperr.Conflict("user.email_taken", "That email is already used.")
			}
		}
		if err != nil {
			return err
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{Action: authz.UserCreate, TargetType: "user", TargetID: &user.ID}); err != nil {
			return err
		}
		out, err = accountFrom(ctx, q, user)
		return err
	})
	return out, err
}

func (s *Service) LookupUser(ctx context.Context, username string) (User, error) {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return User{}, err
	}
	manager, err := s.db.Q().HasManagerMembership(ctx, p.UserID)
	if err != nil {
		return User{}, err
	}
	if !manager {
		return User{}, apperr.ErrRoleInsufficient()
	}
	row, err := s.db.Q().LookupUserByUsername(ctx, username)
	if pg.IsNotFound(err) {
		return User{}, apperr.ErrNotFound()
	}
	if err != nil {
		return User{}, err
	}
	return User{ID: row.ID, Username: row.Username, DisplayName: row.DisplayName}, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (apimodel.Account, error) {
	if _, err := authz.RequireInstanceStaff(ctx); err != nil {
		return apimodel.Account{}, err
	}
	row, err := s.db.Q().GetUserByID(ctx, id)
	if pg.IsNotFound(err) {
		return apimodel.Account{}, apperr.ErrNotFound()
	}
	if err != nil {
		return apimodel.Account{}, err
	}
	return accountFrom(ctx, s.db.Q(), row)
}

func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, in AccountUpdate) (apimodel.Account, error) {
	p, err := authz.RequireInstanceStaff(ctx)
	if err != nil {
		return apimodel.Account{}, err
	}
	if in.DisplayName == nil && !in.EmailSet && in.Status == nil && in.InstanceRole == nil {
		return apimodel.Account{}, apperr.Invalid(apperr.CodeValidation, "Change at least one field.")
	}
	var out apimodel.Account
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		owner, err := q.GetInstanceOwnerForUpdate(ctx)
		if err != nil {
			return err
		}
		current, err := q.GetUserForUpdate(ctx, id)
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		// permissions.md §3: server admins cannot change the server owner or another admin.
		if p.InstanceRole == authz.InstanceAdmin {
			if current.InstanceRole == string(authz.InstanceOwner) ||
				(current.InstanceRole == string(authz.InstanceAdmin) && current.ID != p.UserID) {
				return apperr.ErrRoleInsufficient()
			}
			if in.InstanceRole != nil && *in.InstanceRole != string(authz.InstanceMember) {
				return apperr.ErrRoleInsufficient()
			}
		}
		if current.InstanceRole == string(authz.InstanceOwner) {
			if in.InstanceRole != nil && *in.InstanceRole != string(authz.InstanceOwner) {
				return apperr.Conflict("user.last_instance_owner", "Transfer ownership to another account first.")
			}
			if in.Status != nil && *in.Status == "disabled" {
				return apperr.Conflict("user.last_instance_owner", "Transfer ownership to another account first.")
			}
		}
		role := ""
		if in.InstanceRole != nil {
			role = *in.InstanceRole
		}
		if role == string(authz.InstanceOwner) && current.ID != owner.ID {
			if p.InstanceRole != authz.InstanceOwner {
				return apperr.ErrRoleInsufficient()
			}
			if in.Status != nil && *in.Status == "disabled" {
				return apperr.Conflict("user.disabled_owner", "An owner must be active.")
			}
			if current.Status != "active" && in.Status == nil {
				return apperr.Conflict("user.disabled_owner", "An owner must be active.")
			}
			if err := q.DemoteInstanceOwner(ctx, owner.ID); err != nil {
				return err
			}
		}
		name, status := "", ""
		if in.DisplayName != nil {
			name = *in.DisplayName
		}
		if in.Status != nil {
			status = *in.Status
		}
		user, err := q.UpdateUserFields(ctx, dbgen.UpdateUserFieldsParams{
			UserID: id, SetDisplayName: in.DisplayName != nil, DisplayName: name,
			SetEmail: in.EmailSet, Email: in.Email, SetStatus: in.Status != nil, Status: status,
			SetInstanceRole: in.InstanceRole != nil, InstanceRole: role,
		})
		if constraint, ok := pg.UniqueViolation(err); ok && constraint == "users_email_key" {
			return apperr.Conflict("user.email_taken", "That email is already used.")
		}
		if err != nil {
			return err
		}
		if in.Status != nil && *in.Status == "disabled" {
			if _, err := q.DeleteUserSessions(ctx, id); err != nil {
				return err
			}
		}
		fields := []string{}
		if in.DisplayName != nil {
			fields = append(fields, "displayName")
		}
		if in.EmailSet {
			fields = append(fields, "email")
		}
		if in.Status != nil {
			fields = append(fields, "status")
		}
		if in.InstanceRole != nil {
			fields = append(fields, "instanceRole")
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.UserUpdate, TargetType: "user", TargetID: &id,
			Metadata: map[string]any{"fields": fields},
		}); err != nil {
			return err
		}
		out, err = accountFrom(ctx, q, user)
		return err
	})
	return out, err
}

func (s *Service) ResetUserPassword(ctx context.Context, id uuid.UUID, in PasswordReset) error {
	p, err := authz.RequireInstanceStaff(ctx)
	if err != nil {
		return err
	}
	hash, err := crypto.HashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	return s.db.Tx(ctx, func(q *dbgen.Queries) error {
		user, err := q.GetUserForUpdate(ctx, id)
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if p.InstanceRole == authz.InstanceAdmin &&
			(user.InstanceRole == string(authz.InstanceOwner) ||
				(user.InstanceRole == string(authz.InstanceAdmin) && user.ID != p.UserID)) {
			return apperr.ErrRoleInsufficient()
		}
		if err := q.SetUserPassword(ctx, dbgen.SetUserPasswordParams{ID: id, PasswordHash: &hash}); err != nil {
			return err
		}
		if _, err := q.DeleteUserSessions(ctx, id); err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{Action: authz.UserPasswordReset, TargetType: "user", TargetID: &id})
	})
}
