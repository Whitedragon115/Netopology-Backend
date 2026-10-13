// Package workspace: Workspaces: list (only yours), create (server owner/admin only), read, update, delete.
//
// Operations (method, path under /api/v1, operationId, x-permission). Copy summaries, statuses
// and schemas from api/contract.yaml; follow internal/setup as the reference implementation.
//
//	WP-06  GET    /workspaces                                          listWorkspaces           workspace.list
//	WP-06  POST   /workspaces                                          createWorkspace          workspace.create
//	WP-06  GET    /workspaces/{wsId}                                   getWorkspace             workspace.read
//	WP-06  PATCH  /workspaces/{wsId}                                   updateWorkspace          workspace.update
//	WP-06  DELETE /workspaces/{wsId}                                   deleteWorkspace          workspace.delete
package workspace

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
