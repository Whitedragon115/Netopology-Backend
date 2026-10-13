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
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
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
	huma.Register(api, httpx.Op(authz.WorkspaceList, http.MethodGet, "/workspaces",
		"listWorkspaces", "Workspaces", "List workspaces I am a member of"), h.list)
	huma.Register(api, httpx.Op(authz.WorkspaceCreate, http.MethodPost, "/workspaces",
		"createWorkspace", "Workspaces", "Create a workspace (server owner/admin only; I become its owner)",
		httpx.Status(http.StatusCreated), httpx.Errors(403, 409, 422)), h.create)
	huma.Register(api, httpx.Op(authz.WorkspaceRead, http.MethodGet, "/workspaces/{wsId}",
		"getWorkspace", "Workspaces", "Get a workspace", httpx.Errors(404)), h.get)
	huma.Register(api, httpx.Op(authz.WorkspaceUpdate, http.MethodPatch, "/workspaces/{wsId}",
		"updateWorkspace", "Workspaces", "Update a workspace", httpx.Errors(403, 404, 412, 422)), h.update)
	huma.Register(api, httpx.Op(authz.WorkspaceDelete, http.MethodDelete, "/workspaces/{wsId}",
		"deleteWorkspace", "Workspaces", "Delete a workspace and everything in it",
		httpx.Status(http.StatusNoContent), httpx.Errors(403, 404, 412)), h.delete)
}

type listOutput struct{ Body Page }

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	page, err := h.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	return &listOutput{Body: page}, nil
}

type (
	createInput     struct{ Body WorkspaceCreate }
	workspaceOutput struct {
		ETag string `header:"ETag"`
		Body Workspace
	}
)

func (h *Handler) create(ctx context.Context, in *createInput) (*workspaceOutput, error) {
	row, err := h.svc.Create(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	return &workspaceOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

type workspaceInput struct {
	WsID uuid.UUID `path:"wsId" format:"uuid"`
}

func (h *Handler) get(ctx context.Context, in *workspaceInput) (*workspaceOutput, error) {
	row, err := h.svc.Get(ctx, in.WsID)
	if err != nil {
		return nil, err
	}
	return &workspaceOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

type updateInput struct {
	WsID    uuid.UUID `path:"wsId" format:"uuid"`
	IfMatch string    `header:"If-Match"`
	Body    WorkspaceUpdate
}

func (h *Handler) update(ctx context.Context, in *updateInput) (*workspaceOutput, error) {
	version, present, err := httpx.IfMatch(in.IfMatch)
	if err != nil {
		return nil, err
	}
	var match *int32
	if present {
		match = &version
	}
	row, err := h.svc.Update(ctx, in.WsID, in.Body, match)
	if err != nil {
		return nil, err
	}
	return &workspaceOutput{ETag: httpx.ETag(row.Version), Body: row}, nil
}

type deleteInput struct {
	WsID    uuid.UUID `path:"wsId" format:"uuid"`
	IfMatch string    `header:"If-Match"`
}

func (h *Handler) delete(ctx context.Context, in *deleteInput) (*struct{}, error) {
	version, present, err := httpx.IfMatch(in.IfMatch)
	if err != nil {
		return nil, err
	}
	var match *int32
	if present {
		match = &version
	}
	if err := h.svc.Delete(ctx, in.WsID, match); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}
