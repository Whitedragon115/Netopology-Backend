package inventory

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

func (h *Handler) registerWrites(api huma.API) {
	huma.Register(api, httpx.Op(authz.AreaCreate, http.MethodPost, "/workspaces/{wsId}/areas",
		"createArea", "Areas", "Create an area", httpx.Describe("Editors must include `placeOn` with a canvas granted to them."), httpx.Status(201), httpx.Errors(403, 404, 422)), h.createArea)
	huma.Register(api, httpx.Op(authz.AreaUpdate, http.MethodPatch, "/workspaces/{wsId}/areas/{areaId}",
		"updateArea", "Areas", "Update an area (including moving it under another parent)", httpx.Errors(403, 404, 409, 412, 422)), h.updateArea)
	huma.Register(api, httpx.Op(authz.AreaDelete, http.MethodDelete, "/workspaces/{wsId}/areas/{areaId}",
		"deleteArea", "Areas", "Delete an area (child areas and nodes move to the top level)", httpx.Status(204), httpx.Errors(403, 404, 409, 412)), h.deleteArea)
	huma.Register(api, httpx.Op(authz.NodeCreate, http.MethodPost, "/workspaces/{wsId}/nodes",
		"createNode", "Nodes", "Create a node", httpx.Describe("Editors must include `placeOn` with a canvas granted to them. `slug` is generated from the name if omitted."), httpx.Status(201), httpx.Errors(403, 404, 409, 422)), h.createNode)
	huma.Register(api, httpx.Op(authz.NodeUpdate, http.MethodPatch, "/workspaces/{wsId}/nodes/{nodeId}",
		"updateNode", "Nodes", "Update a node (visible on every canvas it is placed on)", httpx.Errors(403, 404, 409, 412, 422)), h.updateNode)
	huma.Register(api, httpx.Op(authz.NodeDelete, http.MethodDelete, "/workspaces/{wsId}/nodes/{nodeId}",
		"deleteNode", "Nodes", "Delete a node from the inventory (removes it, its links and notes from every canvas)", httpx.Status(204), httpx.Errors(403, 404, 409, 412)), h.deleteNode)
	huma.Register(api, httpx.Op(authz.NodeHealthSet, http.MethodPut, "/workspaces/{wsId}/nodes/{nodeId}/health",
		"setNodeHealth", "Nodes", "Set a node's health manually", httpx.Describe("Plugin-driven health arrives in phase C3 and takes precedence when present."), httpx.Errors(403, 404)), h.setNodeHealth)
	huma.Register(api, httpx.Op(authz.LinkCreate, http.MethodPost, "/workspaces/{wsId}/links",
		"createLink", "Links", "Create a link between two nodes", httpx.Describe("The link appears automatically on every canvas where both nodes are placed (unless that canvas hides it)."), httpx.Status(201), httpx.Errors(403, 404, 422)), h.createLink)
	huma.Register(api, httpx.Op(authz.LinkUpdate, http.MethodPatch, "/workspaces/{wsId}/links/{linkId}",
		"updateLink", "Links", "Update a link", httpx.Errors(403, 404, 412, 422)), h.updateLink)
	huma.Register(api, httpx.Op(authz.LinkDelete, http.MethodDelete, "/workspaces/{wsId}/links/{linkId}",
		"deleteLink", "Links", "Delete a link from the inventory (removes it from every canvas)", httpx.Status(204), httpx.Errors(403, 404, 412)), h.deleteLink)
	huma.Register(api, httpx.Op(authz.NoteCreate, http.MethodPost, "/workspaces/{wsId}/notes",
		"createNote", "Notes", "Create a note", httpx.Status(201), httpx.Errors(403, 404, 422)), h.createNote)
	huma.Register(api, httpx.Op(authz.NoteUpdate, http.MethodPatch, "/workspaces/{wsId}/notes/{noteId}",
		"updateNote", "Notes", "Update a note", httpx.Errors(403, 404, 412, 422)), h.updateNote)
	huma.Register(api, httpx.Op(authz.NoteDelete, http.MethodDelete, "/workspaces/{wsId}/notes/{noteId}",
		"deleteNote", "Notes", "Delete a note", httpx.Status(204), httpx.Errors(403, 404, 412)), h.deleteNote)
}

func matchHeader(header string) (*int32, error) {
	v, present, err := httpx.IfMatch(header)
	if err != nil || !present {
		return nil, err
	}
	return &v, nil
}

type (
	areaCreateInput struct {
		WsID uuid.UUID `path:"wsId"`
		Body AreaCreate
	}
	areaUpdateInput struct {
		WsID    uuid.UUID `path:"wsId"`
		AreaID  uuid.UUID `path:"areaId"`
		IfMatch string    `header:"If-Match"`
		Body    AreaUpdate
	}
	areaDeleteInput struct {
		WsID    uuid.UUID `path:"wsId"`
		AreaID  uuid.UUID `path:"areaId"`
		IfMatch string    `header:"If-Match"`
	}
	areaWriteOutput struct {
		ETag string `header:"ETag"`
		Body Area
	}
)

func (h *Handler) createArea(ctx context.Context, in *areaCreateInput) (*areaWriteOutput, error) {
	row, err := h.svc.CreateArea(ctx, in.WsID, in.Body)
	if err != nil {
		return nil, err
	}
	return &areaWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) updateArea(ctx context.Context, in *areaUpdateInput) (*areaWriteOutput, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateArea(ctx, in.WsID, in.AreaID, in.Body, match)
	if err != nil {
		return nil, err
	}
	return &areaWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) deleteArea(ctx context.Context, in *areaDeleteInput) (*struct{}, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	if err = h.svc.DeleteArea(ctx, in.WsID, in.AreaID, match); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

type (
	nodeCreateInput struct {
		WsID uuid.UUID `path:"wsId"`
		Body NodeCreate
	}
	nodeUpdateInput struct {
		WsID    uuid.UUID `path:"wsId"`
		NodeID  uuid.UUID `path:"nodeId"`
		IfMatch string    `header:"If-Match"`
		Body    NodeUpdate
	}
	nodeDeleteInput struct {
		WsID    uuid.UUID `path:"wsId"`
		NodeID  uuid.UUID `path:"nodeId"`
		IfMatch string    `header:"If-Match"`
	}
	nodeHealthInput struct {
		WsID   uuid.UUID `path:"wsId"`
		NodeID uuid.UUID `path:"nodeId"`
		Body   HealthSet
	}
	nodeWriteOutput struct {
		ETag string `header:"ETag"`
		Body Node
	}
	nodeHealthOutput struct{ Body Node }
)

func (h *Handler) createNode(ctx context.Context, in *nodeCreateInput) (*nodeWriteOutput, error) {
	row, err := h.svc.CreateNode(ctx, in.WsID, in.Body)
	if err != nil {
		return nil, err
	}
	return &nodeWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) updateNode(ctx context.Context, in *nodeUpdateInput) (*nodeWriteOutput, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateNode(ctx, in.WsID, in.NodeID, in.Body, match)
	if err != nil {
		return nil, err
	}
	return &nodeWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) deleteNode(ctx context.Context, in *nodeDeleteInput) (*struct{}, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	if err = h.svc.DeleteNode(ctx, in.WsID, in.NodeID, match); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

func (h *Handler) setNodeHealth(ctx context.Context, in *nodeHealthInput) (*nodeHealthOutput, error) {
	row, err := h.svc.SetNodeHealth(ctx, in.WsID, in.NodeID, in.Body)
	if err != nil {
		return nil, err
	}
	return &nodeHealthOutput{Body: row}, nil
}

type (
	linkCreateInput struct {
		WsID uuid.UUID `path:"wsId"`
		Body LinkCreate
	}
	linkUpdateInput struct {
		WsID    uuid.UUID `path:"wsId"`
		LinkID  uuid.UUID `path:"linkId"`
		IfMatch string    `header:"If-Match"`
		Body    LinkUpdate
	}
	linkDeleteInput struct {
		WsID    uuid.UUID `path:"wsId"`
		LinkID  uuid.UUID `path:"linkId"`
		IfMatch string    `header:"If-Match"`
	}
	linkWriteOutput struct {
		ETag string `header:"ETag"`
		Body Link
	}
)

func (h *Handler) createLink(ctx context.Context, in *linkCreateInput) (*linkWriteOutput, error) {
	row, err := h.svc.CreateLink(ctx, in.WsID, in.Body)
	if err != nil {
		return nil, err
	}
	return &linkWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) updateLink(ctx context.Context, in *linkUpdateInput) (*linkWriteOutput, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateLink(ctx, in.WsID, in.LinkID, in.Body, match)
	if err != nil {
		return nil, err
	}
	return &linkWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) deleteLink(ctx context.Context, in *linkDeleteInput) (*struct{}, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	if err = h.svc.DeleteLink(ctx, in.WsID, in.LinkID, match); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

type (
	noteCreateInput struct {
		WsID uuid.UUID `path:"wsId"`
		Body NoteCreate
	}
	noteUpdateInput struct {
		WsID    uuid.UUID `path:"wsId"`
		NoteID  uuid.UUID `path:"noteId"`
		IfMatch string    `header:"If-Match"`
		Body    NoteUpdate
	}
	noteDeleteInput struct {
		WsID    uuid.UUID `path:"wsId"`
		NoteID  uuid.UUID `path:"noteId"`
		IfMatch string    `header:"If-Match"`
	}
	noteWriteOutput struct {
		ETag string `header:"ETag"`
		Body Note
	}
)

func (h *Handler) createNote(ctx context.Context, in *noteCreateInput) (*noteWriteOutput, error) {
	row, err := h.svc.CreateNote(ctx, in.WsID, in.Body)
	if err != nil {
		return nil, err
	}
	return &noteWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) updateNote(ctx context.Context, in *noteUpdateInput) (*noteWriteOutput, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateNote(ctx, in.WsID, in.NoteID, in.Body, match)
	if err != nil {
		return nil, err
	}
	return &noteWriteOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

func (h *Handler) deleteNote(ctx context.Context, in *noteDeleteInput) (*struct{}, error) {
	match, err := matchHeader(in.IfMatch)
	if err != nil {
		return nil, err
	}
	if err = h.svc.DeleteNote(ctx, in.WsID, in.NoteID, match); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}
