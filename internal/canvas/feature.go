// Package canvas: Canvases (views over the inventory): CRUD, the canvas view (everything needed to draw it), area inner canvases, canvas operations (place/move/remove/link overrides), lock/unlock.
//
// Operations (method, path under /api/v1, operationId, x-permission). Copy summaries, statuses
// and schemas from api/contract.yaml; follow internal/setup as the reference implementation.
//
//	WP-10  GET    /workspaces/{wsId}/canvases                          listCanvases             canvas.list
//	WP-10  POST   /workspaces/{wsId}/canvases                          createCanvas             canvas.create
//	WP-10  GET    /workspaces/{wsId}/canvases/{canvasId}               getCanvas                canvas.read
//	WP-10  PATCH  /workspaces/{wsId}/canvases/{canvasId}               updateCanvas             canvas.update
//	WP-10  DELETE /workspaces/{wsId}/canvases/{canvasId}               deleteCanvas             canvas.delete
//	WP-10  GET    /workspaces/{wsId}/canvases/{canvasId}/view          getCanvasView            canvas.view
//	WP-10  GET    /workspaces/{wsId}/areas/{areaId}/canvas             getAreaCanvasView        area.canvas
//	WP-10  POST   /workspaces/{wsId}/canvases/{canvasId}/operations    applyCanvasOperations    canvas.edit
//	WP-10  POST   /workspaces/{wsId}/canvases/{canvasId}/lock          lockCanvas               canvas.lock
//	WP-10  DELETE /workspaces/{wsId}/canvases/{canvasId}/lock          unlockCanvas             canvas.unlock
package canvas

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
