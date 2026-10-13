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
	huma.Register(api, httpx.Op(authz.MemberList, http.MethodGet, "/workspaces/{wsId}/members",
		"listMembers", "Members", "List members", httpx.Errors(404)), h.list)
	huma.Register(api, httpx.Op(authz.MemberAdd, http.MethodPost, "/workspaces/{wsId}/members",
		"addMember", "Members", "Add an existing account to this workspace",
		httpx.Status(201), httpx.Errors(403, 404, 409)), h.add)
	huma.Register(api, httpx.Op(authz.MemberUpdate, http.MethodPatch, "/workspaces/{wsId}/members/{userId}",
		"updateMember", "Members", "Change a member's role or sensitive-info access",
		httpx.Errors(403, 404, 409)), h.update)
	huma.Register(api, httpx.Op(authz.MemberRemove, http.MethodDelete, "/workspaces/{wsId}/members/{userId}",
		"removeMember", "Members", "Remove a member (or leave, when it is yourself)",
		httpx.Status(204), httpx.Errors(403, 404, 409)), h.remove)
	huma.Register(api, httpx.Op(authz.MemberGrantsSet, http.MethodPut, "/workspaces/{wsId}/members/{userId}/canvas-grants",
		"setMemberCanvasGrants", "Members", "Set which canvases an editor may use (replaces the list)",
		httpx.Errors(403, 404, 422)), h.setGrants)
}

type (
	listInput struct {
		WsID uuid.UUID `path:"wsId" format:"uuid"`
	}
	listOutput struct{ Body MemberPage }
)

func (h *Handler) list(ctx context.Context, in *listInput) (*listOutput, error) {
	page, err := h.svc.List(ctx, in.WsID)
	if err != nil {
		return nil, err
	}
	return &listOutput{Body: page}, nil
}

type addInput struct {
	WsID uuid.UUID `path:"wsId" format:"uuid"`
	Body MemberAdd
}
type memberOutput struct{ Body Member }

func (h *Handler) add(ctx context.Context, in *addInput) (*memberOutput, error) {
	m, err := h.svc.Add(ctx, in.WsID, in.Body)
	if err != nil {
		return nil, err
	}
	return &memberOutput{Body: m}, nil
}

type updateInput struct {
	WsID   uuid.UUID `path:"wsId" format:"uuid"`
	UserID uuid.UUID `path:"userId" format:"uuid"`
	Body   MemberUpdate
}

func (h *Handler) update(ctx context.Context, in *updateInput) (*memberOutput, error) {
	m, err := h.svc.Update(ctx, in.WsID, in.UserID, in.Body)
	if err != nil {
		return nil, err
	}
	return &memberOutput{Body: m}, nil
}

type removeInput struct {
	WsID   uuid.UUID `path:"wsId" format:"uuid"`
	UserID uuid.UUID `path:"userId" format:"uuid"`
}

func (h *Handler) remove(ctx context.Context, in *removeInput) (*struct{}, error) {
	if err := h.svc.Remove(ctx, in.WsID, in.UserID); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

type grantsInput struct {
	WsID   uuid.UUID `path:"wsId" format:"uuid"`
	UserID uuid.UUID `path:"userId" format:"uuid"`
	Body   CanvasGrants
}

func (h *Handler) setGrants(ctx context.Context, in *grantsInput) (*memberOutput, error) {
	m, err := h.svc.SetGrants(ctx, in.WsID, in.UserID, in.Body)
	if err != nil {
		return nil, err
	}
	return &memberOutput{Body: m}, nil
}
