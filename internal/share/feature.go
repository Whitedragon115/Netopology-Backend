// Package share: Share links (owner/admin manage them) and the guest endpoints people use when opening a link. Guests are strictly read-only and get the strictest redaction.
//
// Operations (method, path under /api/v1, operationId, x-permission). Copy summaries, statuses
// and schemas from api/contract.yaml; follow internal/setup as the reference implementation.
//
//	WP-12  GET    /workspaces/{wsId}/canvases/{canvasId}/share-links   listShareLinks           share_link.list
//	WP-12  POST   /workspaces/{wsId}/canvases/{canvasId}/share-links   createShareLink          share_link.create
//	WP-12  PATCH  /workspaces/{wsId}/share-links/{shareLinkId}         updateShareLink          share_link.update
//	WP-12  DELETE /workspaces/{wsId}/share-links/{shareLinkId}         revokeShareLink          share_link.revoke
//	WP-12  GET    /share/{token}                                       getShareInfo             share.info
//	WP-12  POST   /share/{token}/unlock                                unlockShare              share.unlock
//	WP-12  GET    /share/{token}/view                                  getSharedCanvasView      share.view
package share

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
