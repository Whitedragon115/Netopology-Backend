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
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
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
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpx.Op(authz.AreaList, http.MethodGet, "/workspaces/{wsId}/areas",
		"listAreas", "Areas", "List areas", httpx.Errors(404)), h.listAreas)
	huma.Register(api, httpx.Op(authz.AreaRead, http.MethodGet, "/workspaces/{wsId}/areas/{areaId}",
		"getArea", "Areas", "Get an area", httpx.Errors(404)), h.getArea)
	huma.Register(api, httpx.Op(authz.NodeList, http.MethodGet, "/workspaces/{wsId}/nodes",
		"listNodes", "Nodes", "List nodes in the inventory", httpx.Errors(404)), h.listNodes)
	huma.Register(api, httpx.Op(authz.NodeRead, http.MethodGet, "/workspaces/{wsId}/nodes/{nodeId}",
		"getNode", "Nodes", "Get a node", httpx.Errors(404)), h.getNode)
	huma.Register(api, httpx.Op(authz.NodeUsage, http.MethodGet, "/workspaces/{wsId}/nodes/{nodeId}/usage",
		"getNodeUsage", "Nodes", "Where a node is used (for the delete warning)", httpx.Errors(404)), h.getNodeUsage)
	huma.Register(api, httpx.Op(authz.LinkList, http.MethodGet, "/workspaces/{wsId}/links",
		"listLinks", "Links", "List links", httpx.Errors(404)), h.listLinks)
	huma.Register(api, httpx.Op(authz.LinkRead, http.MethodGet, "/workspaces/{wsId}/links/{linkId}",
		"getLink", "Links", "Get a link", httpx.Errors(404)), h.getLink)
	huma.Register(api, httpx.Op(authz.NoteList, http.MethodGet, "/workspaces/{wsId}/notes",
		"listNotes", "Notes", "List notes", httpx.Describe("Filter by one target. Without a filter, returns workspace-level notes."), httpx.Errors(404)), h.listNotes)
	huma.Register(api, httpx.Op(authz.NoteRead, http.MethodGet, "/workspaces/{wsId}/notes/{noteId}",
		"getNote", "Notes", "Get a note", httpx.Errors(404)), h.getNote)
}

type listAreasInput struct {
	WsID uuid.UUID `path:"wsId" format:"uuid"`
	httpx.PageParams
	ParentID string `query:"parentId" doc:"Only direct children of this area. Use root for top-level areas."`
}
type areaPageOutput struct{ Body AreaPage }

func (h *Handler) listAreas(ctx context.Context, in *listAreasInput) (*areaPageOutput, error) {
	page, err := h.svc.ListAreas(ctx, in.WsID, AreaFilter{Cursor: in.Cursor, Limit: in.Limit, Parent: in.ParentID})
	if err != nil {
		return nil, err
	}
	return &areaPageOutput{Body: page}, nil
}

type (
	getAreaInput struct {
		WsID   uuid.UUID `path:"wsId" format:"uuid"`
		AreaID uuid.UUID `path:"areaId" format:"uuid"`
	}
	areaOutput struct {
		ETag string `header:"ETag"`
		Body Area
	}
)

func (h *Handler) getArea(ctx context.Context, in *getAreaInput) (*areaOutput, error) {
	row, err := h.svc.GetArea(ctx, in.WsID, in.AreaID)
	if err != nil {
		return nil, err
	}
	return &areaOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

type listNodesInput struct {
	WsID uuid.UUID `path:"wsId" format:"uuid"`
	httpx.PageParams
	AreaID   string `query:"areaId" format:"uuid"`
	Category string `query:"category" enum:"network,compute,service,storage,external,generic"`
	CanvasID string `query:"canvasId" format:"uuid"`
	Health   string `query:"health" enum:"healthy,degraded,down,unknown"`
	Q        string `query:"q" doc:"Search name, slug and non-sensitive field values"`
}
type nodePageOutput struct{ Body NodePage }

func (h *Handler) listNodes(ctx context.Context, in *listNodesInput) (*nodePageOutput, error) {
	areaID, err := optionalID(in.AreaID)
	if err != nil {
		return nil, err
	}
	canvasID, err := optionalID(in.CanvasID)
	if err != nil {
		return nil, err
	}
	page, err := h.svc.ListNodes(ctx, in.WsID, NodeFilter{
		Cursor: in.Cursor, Limit: in.Limit, AreaID: areaID, Category: cleanSearch(in.Category),
		CanvasID: canvasID, Health: cleanSearch(in.Health), Search: cleanSearch(in.Q),
	})
	if err != nil {
		return nil, err
	}
	return &nodePageOutput{Body: page}, nil
}

type (
	getNodeInput struct {
		WsID   uuid.UUID `path:"wsId" format:"uuid"`
		NodeID uuid.UUID `path:"nodeId" format:"uuid"`
	}
	nodeOutput struct {
		ETag string `header:"ETag"`
		Body Node
	}
)

func (h *Handler) getNode(ctx context.Context, in *getNodeInput) (*nodeOutput, error) {
	row, err := h.svc.GetNode(ctx, in.WsID, in.NodeID)
	if err != nil {
		return nil, err
	}
	return &nodeOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

type nodeUsageOutput struct{ Body NodeUsage }

func (h *Handler) getNodeUsage(ctx context.Context, in *getNodeInput) (*nodeUsageOutput, error) {
	usage, err := h.svc.GetNodeUsage(ctx, in.WsID, in.NodeID)
	if err != nil {
		return nil, err
	}
	return &nodeUsageOutput{Body: usage}, nil
}

type listLinksInput struct {
	WsID uuid.UUID `path:"wsId" format:"uuid"`
	httpx.PageParams
	NodeID string `query:"nodeId" format:"uuid"`
	Kind   string `query:"kind" enum:"physical,vpn_tunnel,proxy_route,dns_dependency,storage_mount,logical"`
}
type linkPageOutput struct{ Body LinkPage }

func (h *Handler) listLinks(ctx context.Context, in *listLinksInput) (*linkPageOutput, error) {
	nodeID, err := optionalID(in.NodeID)
	if err != nil {
		return nil, err
	}
	page, err := h.svc.ListLinks(ctx, in.WsID, LinkFilter{Cursor: in.Cursor, Limit: in.Limit, NodeID: nodeID, Kind: cleanSearch(in.Kind)})
	if err != nil {
		return nil, err
	}
	return &linkPageOutput{Body: page}, nil
}

type (
	getLinkInput struct {
		WsID   uuid.UUID `path:"wsId" format:"uuid"`
		LinkID uuid.UUID `path:"linkId" format:"uuid"`
	}
	linkOutput struct {
		ETag string `header:"ETag"`
		Body Link
	}
)

func (h *Handler) getLink(ctx context.Context, in *getLinkInput) (*linkOutput, error) {
	row, err := h.svc.GetLink(ctx, in.WsID, in.LinkID)
	if err != nil {
		return nil, err
	}
	return &linkOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

type listNotesInput struct {
	WsID uuid.UUID `path:"wsId" format:"uuid"`
	httpx.PageParams
	NodeID   string `query:"nodeId" format:"uuid"`
	LinkID   string `query:"linkId" format:"uuid"`
	AreaID   string `query:"areaId" format:"uuid"`
	CanvasID string `query:"canvasId" format:"uuid"`
	Kind     string `query:"kind" enum:"note,rule"`
}
type notePageOutput struct{ Body NotePage }

func (h *Handler) listNotes(ctx context.Context, in *listNotesInput) (*notePageOutput, error) {
	nodeID, err := optionalID(in.NodeID)
	if err != nil {
		return nil, err
	}
	linkID, err := optionalID(in.LinkID)
	if err != nil {
		return nil, err
	}
	areaID, err := optionalID(in.AreaID)
	if err != nil {
		return nil, err
	}
	canvasID, err := optionalID(in.CanvasID)
	if err != nil {
		return nil, err
	}
	page, err := h.svc.ListNotes(ctx, in.WsID, NoteFilter{
		Cursor: in.Cursor, Limit: in.Limit, NodeID: nodeID, LinkID: linkID,
		AreaID: areaID, CanvasID: canvasID, Kind: cleanSearch(in.Kind),
	})
	if err != nil {
		return nil, err
	}
	return &notePageOutput{Body: page}, nil
}

func optionalID(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperr.Invalid(apperr.CodeValidation, "Invalid UUID query parameter.")
	}
	return &id, nil
}

type (
	getNoteInput struct {
		WsID   uuid.UUID `path:"wsId" format:"uuid"`
		NoteID uuid.UUID `path:"noteId" format:"uuid"`
	}
	noteOutput struct {
		ETag string `header:"ETag"`
		Body Note
	}
)

func (h *Handler) getNote(ctx context.Context, in *getNoteInput) (*noteOutput, error) {
	row, err := h.svc.GetNote(ctx, in.WsID, in.NoteID)
	if err != nil {
		return nil, err
	}
	return &noteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}
