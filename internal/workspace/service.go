package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/slug"
)

func (s *Service) List(ctx context.Context) (Page, error) {
	p, err := authz.RequireSignedIn(ctx)
	if err != nil {
		return Page{}, err
	}
	rows, err := s.db.Q().ListWorkspacesForUser(ctx, p.UserID)
	if err != nil {
		return Page{}, fmt.Errorf("list workspaces: %w", err)
	}
	page := Page{Items: make([]Workspace, 0, len(rows))}
	for _, row := range rows {
		item, err := fromRow(dbgen.Workspace{
			ID: row.ID, Name: row.Name, Slug: row.Slug, Description: row.Description,
			IsProtected: row.IsProtected, Settings: row.Settings, CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
		}, Membership{Role: row.MemberRole, CanViewSensitive: row.CanViewSensitive})
		if err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (s *Service) Create(ctx context.Context, req WorkspaceCreate) (Workspace, error) {
	p, err := authz.RequireInstanceStaff(ctx)
	if err != nil {
		return Workspace{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		return Workspace{}, apperr.Invalid(apperr.CodeValidation, "Invalid workspace name.")
	}
	wsSlug := req.Slug
	if wsSlug == "" {
		wsSlug = slug.Make(name)
	}
	var row dbgen.Workspace
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		var err error
		row, err = q.CreateWorkspace(ctx, dbgen.CreateWorkspaceParams{Name: name, Slug: wsSlug, Description: req.Description, CreatedBy: &p.UserID})
		if err != nil {
			return err
		}
		if err := q.InsertMember(ctx, dbgen.InsertMemberParams{
			WorkspaceID: row.ID, UserID: p.UserID, Role: string(authz.RoleOwner), CanViewSensitive: true, AddedBy: &p.UserID,
		}); err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &row.ID, Action: authz.WorkspaceCreate, TargetType: "workspace", TargetID: &row.ID})
	})
	if err != nil {
		return Workspace{}, workspaceError(err)
	}
	return fromRow(row, Membership{Role: string(authz.RoleOwner), CanViewSensitive: true})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Workspace, error) {
	actor, err := s.authz.Actor(ctx, id, authz.WorkspaceRead)
	if err != nil {
		return Workspace{}, err
	}
	row, err := s.db.Q().GetWorkspace(ctx, id)
	if pg.IsNotFound(err) {
		return Workspace{}, apperr.ErrNotFound()
	}
	if err != nil {
		return Workspace{}, fmt.Errorf("get workspace: %w", err)
	}
	return fromRow(row, Membership{Role: string(actor.Role), CanViewSensitive: actor.CanViewSensitive})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req WorkspaceUpdate, match *int32) (Workspace, error) {
	actor, err := s.authz.Actor(ctx, id, authz.WorkspaceUpdate)
	if err != nil {
		return Workspace{}, err
	}
	if req.Name == nil && req.Slug == nil && req.Description == nil && req.IsProtected == nil && req.Settings == nil {
		return Workspace{}, apperr.Invalid(apperr.CodeValidation, "At least one field is required.")
	}
	if req.Name != nil && (strings.TrimSpace(*req.Name) == "" || len(strings.TrimSpace(*req.Name)) > 100) {
		return Workspace{}, apperr.Invalid(apperr.CodeValidation, "Invalid workspace name.")
	}
	if req.IsProtected != nil && !*req.IsProtected && actor.Role == authz.RoleAdmin {
		return Workspace{}, apperr.Forbidden(apperr.CodeProtected, "Only the workspace owner can remove protection.")
	}
	var updated dbgen.Workspace
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		current, err := q.LockWorkspace(ctx, id)
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if match != nil && current.Version != *match {
			return apperr.PreconditionFailed()
		}
		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.Slug != nil {
			current.Slug = *req.Slug
		}
		if req.Description != nil {
			current.Description = *req.Description
		}
		if req.IsProtected != nil {
			current.IsProtected = *req.IsProtected
		}
		if req.Settings != nil {
			current.Settings, err = json.Marshal(req.Settings)
			if err != nil {
				return err
			}
		}
		updated, err = q.UpdateWorkspace(ctx, dbgen.UpdateWorkspaceParams{
			ID: id, Name: current.Name, Slug: current.Slug, Description: current.Description,
			IsProtected: current.IsProtected, Settings: current.Settings, Version: current.Version,
		})
		if err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &id, Action: authz.WorkspaceUpdate, TargetType: "workspace", TargetID: &id})
	})
	if err != nil {
		return Workspace{}, workspaceError(err)
	}
	return fromRow(updated, Membership{Role: string(actor.Role), CanViewSensitive: actor.CanViewSensitive})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, match *int32) error {
	_, err := s.authz.Actor(ctx, id, authz.WorkspaceDelete)
	if err != nil {
		return err
	}
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		current, err := q.LockWorkspace(ctx, id)
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if match != nil && current.Version != *match {
			return apperr.PreconditionFailed()
		}
		if current.IsProtected {
			return apperr.Forbidden(apperr.CodeProtected, "Remove workspace protection before deleting it.")
		}
		n, err := q.DeleteWorkspace(ctx, dbgen.DeleteWorkspaceParams{ID: id, Version: current.Version})
		if err != nil {
			return err
		}
		if n != 1 {
			return apperr.PreconditionFailed()
		}
		// Workspace-scoped audit rows cascade on deletion; preserve this entry outside that FK.
		return auditlog.Write(ctx, q, auditlog.Entry{Action: authz.WorkspaceDelete, TargetType: "workspace", TargetID: &id})
	})
	return err
}

func workspaceError(err error) error {
	if constraint, ok := pg.UniqueViolation(err); ok && constraint == "workspaces_slug_key" {
		return apperr.Conflict(apperr.CodeSlugTaken, "That workspace slug is already used.")
	}
	return err
}
