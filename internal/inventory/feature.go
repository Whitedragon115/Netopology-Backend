// Package inventory: The shared inventory of a workspace: areas, nodes, links, notes. Owns the editor visibility set (permissions.md rule 3) and redaction (permissions.md §4), which canvas, share and events reuse through the Reader interface.
//
// Operations (method, path under /api/v1, operationId, x-permission). Copy summaries, statuses
// and schemas from api/contract.yaml; follow internal/setup as the reference implementation.
//
//	WP-08  GET    /workspaces/{wsId}/areas                             listAreas                area.list
//	WP-08  GET    /workspaces/{wsId}/areas/{areaId}                    getArea                  area.read
//	WP-08  GET    /workspaces/{wsId}/nodes                             listNodes                node.list
//	WP-08  GET    /workspaces/{wsId}/nodes/{nodeId}                    getNode                  node.read
//	WP-08  GET    /workspaces/{wsId}/nodes/{nodeId}/usage              getNodeUsage             node.usage
//	WP-08  GET    /workspaces/{wsId}/links                             listLinks                link.list
//	WP-08  GET    /workspaces/{wsId}/links/{linkId}                    getLink                  link.read
//	WP-08  GET    /workspaces/{wsId}/notes                             listNotes                note.list
//	WP-08  GET    /workspaces/{wsId}/notes/{noteId}                    getNote                  note.read
//	WP-09  POST   /workspaces/{wsId}/areas                             createArea               area.create
//	WP-09  PATCH  /workspaces/{wsId}/areas/{areaId}                    updateArea               area.update
//	WP-09  DELETE /workspaces/{wsId}/areas/{areaId}                    deleteArea               area.delete
//	WP-09  POST   /workspaces/{wsId}/nodes                             createNode               node.create
//	WP-09  PATCH  /workspaces/{wsId}/nodes/{nodeId}                    updateNode               node.update
//	WP-09  DELETE /workspaces/{wsId}/nodes/{nodeId}                    deleteNode               node.delete
//	WP-09  PUT    /workspaces/{wsId}/nodes/{nodeId}/health             setNodeHealth            node.health_set
//	WP-09  POST   /workspaces/{wsId}/links                             createLink               link.create
//	WP-09  PATCH  /workspaces/{wsId}/links/{linkId}                    updateLink               link.update
//	WP-09  DELETE /workspaces/{wsId}/links/{linkId}                    deleteLink               link.delete
//	WP-09  POST   /workspaces/{wsId}/notes                             createNote               note.create
//	WP-09  PATCH  /workspaces/{wsId}/notes/{noteId}                    updateNote               note.update
//	WP-09  DELETE /workspaces/{wsId}/notes/{noteId}                    deleteNote               note.delete
package inventory

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
