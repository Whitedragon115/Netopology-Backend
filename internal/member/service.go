package member

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type memberRow struct {
	userID                      uuid.UUID
	username, displayName, role string
	sensitive                   bool
	created                     time.Time
}

func rowFromGet(row dbgen.GetMemberRow) memberRow {
	return memberRow{row.UserID, row.Username, row.DisplayName, row.Role, row.CanViewSensitive, row.CreatedAt}
}

func rowFromList(row dbgen.ListMembersRow) memberRow {
	return memberRow{row.UserID, row.Username, row.DisplayName, row.Role, row.CanViewSensitive, row.CreatedAt}
}

func render(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, row memberRow, manager bool) (Member, error) {
	m := Member{User: PublicUser{ID: row.userID, Username: row.username, DisplayName: row.displayName}, Role: row.role}
	if !manager {
		return m, nil
	} // permissions.md §3: editors and viewers see names and roles only.
	sensitive := row.sensitive || row.role == string(authz.RoleOwner) || row.role == string(authz.RoleAdmin)
	m.CanViewSensitive = &sensitive
	m.CreatedAt = &row.created
	ids := []uuid.UUID{}
	if row.role == string(authz.RoleEditor) {
		var err error
		ids, err = q.ListMemberGrants(ctx, dbgen.ListMemberGrantsParams{WorkspaceID: wsID, UserID: row.userID})
		if err != nil {
			return Member{}, err
		}
	}
	m.CanvasIDs = &ids
	return m, nil
}

func (s *Service) List(ctx context.Context, wsID uuid.UUID) (MemberPage, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.MemberList)
	if err != nil {
		return MemberPage{}, err
	}
	page := MemberPage{Items: []Member{}}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		rows, err := q.ListMembers(ctx, wsID)
		if err != nil {
			return err
		}
		for _, row := range rows {
			m, err := render(ctx, q, wsID, rowFromList(row), actor.IsManager())
			if err != nil {
				return err
			}
			page.Items = append(page.Items, m)
		}
		return nil
	})
	return page, err
}

func (s *Service) Add(ctx context.Context, wsID uuid.UUID, in MemberAdd) (Member, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.MemberAdd)
	if err != nil {
		return Member{}, err
	}
	if !validRole(in.Role) {
		return Member{}, apperr.Invalid(apperr.CodeValidation, "Invalid workspace role.")
	}
	if actor.Role == authz.RoleAdmin && in.Role == string(authz.RoleOwner) {
		return Member{}, apperr.ErrRoleInsufficient()
	}
	if in.Role != string(authz.RoleEditor) && len(in.CanvasIDs) > 0 {
		return Member{}, invalidGrants()
	}
	var out Member
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		if _, err := q.GetUserByID(ctx, in.UserID); pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		} else if err != nil {
			return err
		}
		if err := validateGrants(ctx, q, wsID, in.CanvasIDs); err != nil {
			return err
		}
		sensitive := in.CanViewSensitive || in.Role == string(authz.RoleOwner) || in.Role == string(authz.RoleAdmin)
		if err := q.InsertMember(ctx, dbgen.InsertMemberParams{WorkspaceID: wsID, UserID: in.UserID, Role: in.Role, CanViewSensitive: sensitive, AddedBy: &actor.Principal.UserID}); err != nil {
			if _, ok := pg.UniqueViolation(err); ok {
				return apperr.Conflict("member.already_exists", "That account is already a member.")
			}
			return err
		}
		if len(in.CanvasIDs) > 0 {
			if err := q.ReplaceCanvasGrants(ctx, dbgen.ReplaceCanvasGrantsParams{WorkspaceID: wsID, UserID: in.UserID, Column3: in.CanvasIDs, GrantedBy: &actor.Principal.UserID}); err != nil {
				return err
			}
		}
		out, err = load(ctx, q, wsID, in.UserID)
		if err != nil {
			return err
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &wsID, Action: authz.MemberAdd, TargetType: "member", TargetID: &in.UserID}); err != nil {
			return err
		}
		return events.Emit(ctx, q, events.Change{WorkspaceID: wsID, Type: events.MemberChanged, SubjectID: &in.UserID})
	})
	return out, err
}

func (s *Service) Update(ctx context.Context, wsID, userID uuid.UUID, in MemberUpdate) (Member, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.MemberUpdate)
	if err != nil {
		return Member{}, err
	}
	if in.Role == nil && in.CanViewSensitive == nil {
		return Member{}, apperr.Invalid(apperr.CodeValidation, "At least one field is required.")
	}
	if in.Role != nil && !validRole(*in.Role) {
		return Member{}, apperr.Invalid(apperr.CodeValidation, "Invalid workspace role.")
	}
	var out Member
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		current, err := q.LockMember(ctx, dbgen.LockMemberParams{WorkspaceID: wsID, UserID: userID})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if actor.Role == authz.RoleAdmin && (current.Role == string(authz.RoleOwner) || (in.Role != nil && *in.Role == string(authz.RoleOwner))) {
			return apperr.ErrRoleInsufficient()
		}
		role := current.Role
		if in.Role != nil {
			role = *in.Role
		}
		if current.Role == string(authz.RoleOwner) && role != current.Role {
			if err := requireAnotherOwner(ctx, q, wsID); err != nil {
				return err
			}
		}
		sensitive := current.CanViewSensitive
		if in.CanViewSensitive != nil {
			sensitive = *in.CanViewSensitive
		}
		if role == string(authz.RoleOwner) || role == string(authz.RoleAdmin) {
			sensitive = true
		}
		if _, err := q.UpdateMember(ctx, dbgen.UpdateMemberParams{WorkspaceID: wsID, UserID: userID, Role: role, CanViewSensitive: sensitive}); err != nil {
			return err
		}
		if role != string(authz.RoleEditor) && current.Role == string(authz.RoleEditor) {
			if err := q.ReplaceCanvasGrants(ctx, dbgen.ReplaceCanvasGrantsParams{WorkspaceID: wsID, UserID: userID, Column3: []uuid.UUID{}, GrantedBy: &actor.Principal.UserID}); err != nil {
				return err
			}
		}
		out, err = load(ctx, q, wsID, userID)
		if err != nil {
			return err
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &wsID, Action: authz.MemberUpdate, TargetType: "member", TargetID: &userID}); err != nil {
			return err
		}
		return events.Emit(ctx, q, events.Change{WorkspaceID: wsID, Type: events.MemberChanged, SubjectID: &userID})
	})
	return out, err
}

func (s *Service) Remove(ctx context.Context, wsID, userID uuid.UUID) error {
	actor, err := s.authz.Actor(ctx, wsID, authz.MemberRemove)
	if err != nil {
		return err
	}
	if !actor.IsManager() && actor.UserID() != userID {
		return apperr.ErrRoleInsufficient()
	}
	return s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		current, err := q.LockMember(ctx, dbgen.LockMemberParams{WorkspaceID: wsID, UserID: userID})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if actor.Role == authz.RoleAdmin && current.Role == string(authz.RoleOwner) {
			return apperr.ErrRoleInsufficient()
		}
		if current.Role == string(authz.RoleOwner) {
			if err := requireAnotherOwner(ctx, q, wsID); err != nil {
				return err
			}
		}
		if _, err := q.DeleteMember(ctx, dbgen.DeleteMemberParams{WorkspaceID: wsID, UserID: userID}); err != nil {
			return err
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &wsID, Action: authz.MemberRemove, TargetType: "member", TargetID: &userID}); err != nil {
			return err
		}
		return events.Emit(ctx, q, events.Change{WorkspaceID: wsID, Type: events.MemberChanged, SubjectID: &userID})
	})
}

func (s *Service) SetGrants(ctx context.Context, wsID, userID uuid.UUID, in CanvasGrants) (Member, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.MemberGrantsSet)
	if err != nil {
		return Member{}, err
	}
	var out Member
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		current, err := q.LockMember(ctx, dbgen.LockMemberParams{WorkspaceID: wsID, UserID: userID})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if current.Role != string(authz.RoleEditor) {
			return invalidGrants()
		}
		if err := validateGrants(ctx, q, wsID, in.CanvasIDs); err != nil {
			return err
		}
		if err := q.ReplaceCanvasGrants(ctx, dbgen.ReplaceCanvasGrantsParams{WorkspaceID: wsID, UserID: userID, Column3: in.CanvasIDs, GrantedBy: &actor.Principal.UserID}); err != nil {
			return err
		}
		out, err = load(ctx, q, wsID, userID)
		if err != nil {
			return err
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &wsID, Action: authz.MemberGrantsSet, TargetType: "member", TargetID: &userID}); err != nil {
			return err
		}
		return events.Emit(ctx, q, events.Change{WorkspaceID: wsID, Type: events.MemberChanged, SubjectID: &userID})
	})
	return out, err
}

func (s *Service) Get(ctx context.Context, wsID, userID uuid.UUID) (Member, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.MemberList)
	if err != nil {
		return Member{}, err
	}
	var out Member
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.GetMember(ctx, dbgen.GetMemberParams{WorkspaceID: wsID, UserID: userID})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		out, err = render(ctx, q, wsID, rowFromGet(row), actor.IsManager())
		return err
	})
	return out, err
}

func load(ctx context.Context, q *dbgen.Queries, wsID, userID uuid.UUID) (Member, error) {
	row, err := q.GetMember(ctx, dbgen.GetMemberParams{WorkspaceID: wsID, UserID: userID})
	if err != nil {
		return Member{}, fmt.Errorf("load member: %w", err)
	}
	return render(ctx, q, wsID, rowFromGet(row), true)
}

func requireAnotherOwner(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID) error {
	owners, err := q.LockOwners(ctx, wsID)
	if err != nil {
		return err
	}
	if len(owners) < 2 {
		return apperr.Conflict(apperr.CodeLastOwner, "A workspace must keep at least one owner.")
	}
	return nil
}

func validRole(role string) bool {
	switch role {
	case string(authz.RoleOwner), string(authz.RoleAdmin), string(authz.RoleEditor), string(authz.RoleViewer):
		return true
	}
	return false
}

func invalidGrants() error {
	return apperr.Invalid(apperr.CodeValidation, "Canvas grants must name distinct tab canvases in this workspace and target an editor.")
}

func validateGrants(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if id == uuid.Nil || seen[id] {
			return invalidGrants()
		}
		seen[id] = true
	}
	n, err := q.CountTabCanvases(ctx, dbgen.CountTabCanvasesParams{WorkspaceID: wsID, Column2: ids})
	if err != nil {
		return err
	}
	if n != int64(len(ids)) {
		return invalidGrants()
	}
	return nil
}
