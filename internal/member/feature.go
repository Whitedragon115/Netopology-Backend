// Package member: Workspace membership: roles, canViewSensitive, editor canvas grants.
//
// Operations (method, path under /api/v1, operationId, x-permission). Copy summaries, statuses
// and schemas from api/contract.yaml; follow internal/setup as the reference implementation.
//
//	WP-07  GET    /workspaces/{wsId}/members                           listMembers              member.list
//	WP-07  POST   /workspaces/{wsId}/members                           addMember                member.add
//	WP-07  PATCH  /workspaces/{wsId}/members/{userId}                  updateMember             member.update
//	WP-07  DELETE /workspaces/{wsId}/members/{userId}                  removeMember             member.remove
//	WP-07  PUT    /workspaces/{wsId}/members/{userId}/canvas-grants    setMemberCanvasGrants    member.grants_set
package member

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
)

// Service holds the business rules and AUTHORIZATION of this feature. Every exported method takes
// ctx first, gets the caller from it (authz.PrincipalFrom / s.authz.Actor), and returns
// *apperr.Error for expected failures.
type Service struct {
	db    *pg.DB
	authz *authz.Resolver
}

func NewService(db *pg.DB, az *authz.Resolver) *Service {
	return &Service{db: db, authz: az}
}

// Handler is the HTTP layer (Huma). No business rules, no SQL.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds this feature's operations to the API.
// TODO: implement the operations listed in the package comment (see docs/WORK_PACKAGES.md).
func (h *Handler) Register(api huma.API) {
	_ = api
}
