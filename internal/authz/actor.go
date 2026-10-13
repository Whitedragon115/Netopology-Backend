package authz

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// Role is a workspace role (workspace_members.role).
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

func (r Role) bit() roleSet {
	switch r {
	case RoleOwner:
		return o
	case RoleAdmin:
		return a
	case RoleEditor:
		return e
	case RoleViewer:
		return v
	}
	return 0
}

// InstanceRole is the server-level role (users.instance_role).
type InstanceRole string

const (
	InstanceOwner  InstanceRole = "owner"
	InstanceAdmin  InstanceRole = "admin"
	InstanceMember InstanceRole = "member"
)

// Principal is WHO is calling, resolved from cookies by auth.Middleware (or share.Middleware).
// It says nothing about a workspace yet; see Actor.
type Principal struct {
	UserID       uuid.UUID // uuid.Nil for anonymous and guests
	Username     string
	InstanceRole InstanceRole // "" for anonymous and guests
	SessionID    uuid.UUID
	Guest        *GuestGrant // non-nil only for share-link guests
}

// GuestGrant is what a share-link cookie unlocks: one canvas, read-only.
type GuestGrant struct {
	ShareLinkID uuid.UUID
	WorkspaceID uuid.UUID
	CanvasID    uuid.UUID
}

func (p *Principal) SignedIn() bool { return p != nil && p.UserID != uuid.Nil }

// IsInstanceStaff: server owner or server admin.
func (p *Principal) IsInstanceStaff() bool {
	return p.SignedIn() && (p.InstanceRole == InstanceOwner || p.InstanceRole == InstanceAdmin)
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom never returns nil: no cookie means an anonymous principal.
func PrincipalFrom(ctx context.Context) *Principal {
	if p, ok := ctx.Value(principalKey{}).(*Principal); ok && p != nil {
		return p
	}
	return &Principal{}
}

// RequireSignedIn is the check for ScopeAccount permissions.
func RequireSignedIn(ctx context.Context) (*Principal, error) {
	p := PrincipalFrom(ctx)
	if !p.SignedIn() {
		return nil, apperr.Unauthenticated(apperr.CodeNotSignedIn, "Sign in first.")
	}
	return p, nil
}

// RequireInstanceStaff is the check for ScopeInstance permissions.
func RequireInstanceStaff(ctx context.Context) (*Principal, error) {
	p, err := RequireSignedIn(ctx)
	if err != nil {
		return nil, err
	}
	if !p.IsInstanceStaff() {
		return nil, apperr.Forbidden(apperr.CodeRoleInsufficient, "Only the server owner or a server admin can do this.")
	}
	return p, nil
}

// Actor is a signed-in member acting inside one workspace. Every workspace-scoped service
// method starts with:
//
//	actor, err := s.authz.Actor(ctx, wsID, authz.NodeUpdate)
//	if err != nil { return nil, err }
//
// which returns 404 for non-members (rule 2) and 403 when the role lacks the permission.
type Actor struct {
	Principal        *Principal
	WorkspaceID      uuid.UUID
	Role             Role
	CanViewSensitive bool
	// WorkspaceProtected mirrors workspaces.is_protected (rule 6).
	WorkspaceProtected bool

	db      *pg.DB
	granted map[uuid.UUID]bool // editor canvas grants, loaded on first use
}

func (a *Actor) UserID() uuid.UUID { return a.Principal.UserID }

// IsManager: owner or admin (sees everything in the workspace).
func (a *Actor) IsManager() bool { return a.Role == RoleOwner || a.Role == RoleAdmin }

// SeesSensitive: redaction rule from permissions.md §4.
func (a *Actor) SeesSensitive() bool { return a.IsManager() || a.CanViewSensitive }

// Can reports whether the role allows p (before the service rules).
func (a *Actor) Can(p Permission) bool { return RoleAllows(a.Role, p) }

// Require returns 403 role.insufficient unless the role allows p.
func (a *Actor) Require(p Permission) error {
	if !a.Can(p) {
		return apperr.ErrRoleInsufficient()
	}
	return nil
}

// CanvasGranted reports whether an EDITOR was granted the canvas. Owners, admins and viewers see
// every tab canvas, so this returns true for them. Area inner canvases follow rule 11 and are
// decided by the canvas service, not here.
func (a *Actor) CanvasGranted(ctx context.Context, canvasID uuid.UUID) (bool, error) {
	if a.Role != RoleEditor {
		return true, nil
	}
	ids, err := a.GrantedCanvasIDs(ctx)
	if err != nil {
		return false, err
	}
	return ids[canvasID], nil
}

// GrantedCanvasIDs returns the editor's granted canvases (empty for other roles: they need none).
func (a *Actor) GrantedCanvasIDs(ctx context.Context) (map[uuid.UUID]bool, error) {
	if a.granted != nil || a.Role != RoleEditor {
		return a.granted, nil
	}
	var ids []uuid.UUID
	err := a.db.InWorkspace(ctx, a.WorkspaceID, func(q *dbgen.Queries) error {
		var err error
		ids, err = q.ListGrantedCanvasIDs(ctx, dbgen.ListGrantedCanvasIDsParams{WorkspaceID: a.WorkspaceID, UserID: a.UserID()})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("load canvas grants: %w", err)
	}
	a.granted = make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		a.granted[id] = true
	}
	return a.granted, nil
}

// Resolver turns (principal, workspace) into an Actor.
type Resolver struct{ db *pg.DB }

func NewResolver(db *pg.DB) *Resolver { return &Resolver{db: db} }

// Actor loads the caller's membership and checks perm. Non-members and guests get 404, never 403.
func (r *Resolver) Actor(ctx context.Context, workspaceID uuid.UUID, perm Permission) (*Actor, error) {
	p := PrincipalFrom(ctx)
	if !p.SignedIn() {
		return nil, apperr.Unauthenticated(apperr.CodeNotSignedIn, "Sign in first.")
	}
	m, err := r.db.Q().GetMembership(ctx, dbgen.GetMembershipParams{WorkspaceID: workspaceID, UserID: p.UserID})
	if pg.IsNotFound(err) {
		return nil, apperr.ErrNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("load membership: %w", err)
	}
	act := &Actor{
		Principal:          p,
		WorkspaceID:        workspaceID,
		Role:               Role(m.Role),
		CanViewSensitive:   m.CanViewSensitive,
		WorkspaceProtected: m.WorkspaceProtected,
		db:                 r.db,
	}
	if err := act.Require(perm); err != nil {
		return nil, err
	}
	return act, nil
}
