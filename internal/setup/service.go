// Package setup is first-run setup: create the first account (the server owner).
//
// It is the REFERENCE FEATURE: small, but it shows every pattern the other features follow
// (docs/ARCHITECTURE.md §12): handler -> service -> sqlc queries, one transaction for the change +
// its audit entry, typed apperr errors, no HTTP in the service, dependencies as small interfaces.
package setup

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/slug"
)

// SessionStarter is the one thing setup needs from internal/auth (rule: depend on small interfaces,
// not on another feature's internals).
type SessionStarter interface {
	Start(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, remember bool) (http.Cookie, error)
}

type Service struct {
	db       *pg.DB
	sessions SessionStarter
}

func NewService(db *pg.DB, sessions SessionStarter) *Service {
	return &Service{db: db, sessions: sessions}
}

// Status: setup is needed while no account exists.
func (s *Service) Status(ctx context.Context) (SetupStatus, error) {
	n, err := s.db.Q().CountUsers(ctx)
	if err != nil {
		return SetupStatus{}, fmt.Errorf("count users: %w", err)
	}
	return SetupStatus{NeedsSetup: n == 0}, nil
}

// Complete creates the server owner (and optionally the first workspace) and signs them in.
// Only works while no account exists: 409 setup.already_completed afterwards.
func (s *Service) Complete(ctx context.Context, req SetupRequest) (apimodel.Account, http.Cookie, error) {
	// Hash outside the transaction: argon2id is deliberately slow, don't hold locks meanwhile.
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return apimodel.Account{}, http.Cookie{}, err
	}

	var (
		user   dbgen.User
		cookie http.Cookie
	)
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		// Two browsers finishing setup at once: the advisory lock + count make exactly one win.
		locked, err := q.TryLockSetup(ctx)
		if err != nil {
			return err
		}
		n, err := q.CountUsers(ctx)
		if err != nil {
			return err
		}
		if !locked || n > 0 {
			return apperr.Conflict(apperr.CodeSetupDone, "Setup is already complete. Sign in instead.")
		}

		user, err = q.CreateUser(ctx, dbgen.CreateUserParams{
			Username:     req.Username,
			Email:        optional(req.Email),
			DisplayName:  strings.TrimSpace(req.DisplayName),
			PasswordHash: &hash,
			InstanceRole: string(authz.InstanceOwner),
		})
		if err != nil {
			return fmt.Errorf("create owner: %w", err)
		}

		if name := strings.TrimSpace(req.WorkspaceName); name != "" {
			ws, err := q.CreateWorkspace(ctx, dbgen.CreateWorkspaceParams{
				Name: name, Slug: slug.Make(name), Description: "", CreatedBy: &user.ID,
			})
			if err != nil {
				return fmt.Errorf("create first workspace: %w", err)
			}
			if err := q.InsertMember(ctx, dbgen.InsertMemberParams{
				WorkspaceID: ws.ID, UserID: user.ID, Role: string(authz.RoleOwner), CanViewSensitive: true, AddedBy: &user.ID,
			}); err != nil {
				return fmt.Errorf("add owner membership: %w", err)
			}
			if err := auditlog.Write(ctx, q, auditlog.Entry{
				WorkspaceID: &ws.ID, Action: authz.WorkspaceCreate, TargetType: "workspace", TargetID: &ws.ID, ActorID: &user.ID,
			}); err != nil {
				return err
			}
		}

		if err := auditlog.Write(ctx, q, auditlog.Entry{
			Action: authz.SetupComplete, TargetType: "user", TargetID: &user.ID, ActorID: &user.ID,
		}); err != nil {
			return err
		}

		cookie, err = s.sessions.Start(ctx, q, user.ID, false)
		return err
	})
	if err != nil {
		return apimodel.Account{}, http.Cookie{}, err
	}
	return apimodel.AccountFrom(user, nil), cookie, nil
}

func optional(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
