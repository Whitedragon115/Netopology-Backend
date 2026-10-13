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
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// InventoryReader is supplied by the server; canvas depends only on shared wire models.
type InventoryReader interface {
	ReadForActor(context.Context, *dbgen.Queries, *authz.Actor, apimodel.IDs) (apimodel.Snapshot, error)
}

// Service holds the business rules and AUTHORIZATION of this feature. Every exported method takes
// ctx first, gets the caller from it (authz.PrincipalFrom / s.authz.Actor), and returns
// *apperr.Error for expected failures.
type Service struct {
	db     *pg.DB
	authz  *authz.Resolver
	reader InventoryReader
}

func NewService(db *pg.DB, az *authz.Resolver, reader InventoryReader) *Service {
	return &Service{db: db, authz: az, reader: reader}
}

// Handler is the HTTP layer (Huma). No business rules, no SQL.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds this feature's operations to the API.
// TODO: implement the operations listed in the package comment (see docs/WORK_PACKAGES.md).
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpx.Op(authz.CanvasList, http.MethodGet, "/workspaces/{wsId}/canvases",
		"listCanvases", "Canvases", "List the tab canvases I can see (area inner canvases are opened through their area)", httpx.Errors(404)), h.list)
	huma.Register(api, httpx.Op(authz.CanvasCreate, http.MethodPost, "/workspaces/{wsId}/canvases",
		"createCanvas", "Canvases", "CanvasCreate a canvas", httpx.Status(201), httpx.Errors(403, 404, 409, 422)), h.create)
	huma.Register(api, httpx.Op(authz.CanvasRead, http.MethodGet, "/workspaces/{wsId}/canvases/{canvasId}",
		"getCanvas", "Canvases", "Get canvas metadata", httpx.Errors(404)), h.get)
	huma.Register(api, httpx.Op(authz.CanvasUpdate, http.MethodPatch, "/workspaces/{wsId}/canvases/{canvasId}",
		"updateCanvas", "Canvases", "CanvasUpdate canvas metadata (name, order, settings, default viewport)", httpx.Errors(403, 404, 409, 412)), h.update)
	huma.Register(api, httpx.Op(authz.CanvasDelete, http.MethodDelete, "/workspaces/{wsId}/canvases/{canvasId}",
		"deleteCanvas", "Canvases", "Delete a canvas (the inventory is untouched)", httpx.Status(204), httpx.Errors(403, 404, 412)), h.delete)
	huma.Register(api, httpx.Op(authz.CanvasView, http.MethodGet, "/workspaces/{wsId}/canvases/{canvasId}/view",
		"getCanvasView", "Canvases", "Everything needed to draw a canvas, redacted for the caller", httpx.Errors(404)), h.getView)
	huma.Register(api, httpx.Op(authz.AreaCanvas, http.MethodGet, "/workspaces/{wsId}/areas/{areaId}/canvas",
		"getAreaCanvasView", "Areas", "Enter an area - its inner canvas, ready to draw",
		httpx.Describe("Double-clicking a collapsed area opens this. The inner canvas is created on first open, with the area's member nodes and child areas pre-placed. Edit it with `applyCanvasOperations` using `canvas.id` from the response. `breadcrumb` lists the path of areas from the top."), httpx.Errors(404)), h.getAreaView)
	h.registerOperations(api)
}
