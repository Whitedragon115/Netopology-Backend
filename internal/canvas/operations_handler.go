package canvas

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

type operationsInput struct {
	WsID     uuid.UUID `path:"wsId"`
	CanvasID uuid.UUID `path:"canvasId"`
	Body     CanvasOperationBatch
}

type (
	operationsOutput struct{ Body CanvasOperationResult }
	lockOutput       struct{ Body Canvas }
)

func (h *Handler) applyOperations(ctx context.Context, in *operationsInput) (*operationsOutput, error) {
	result, err := h.svc.ApplyOperations(ctx, in.WsID, in.CanvasID, in.Body)
	if err != nil {
		return nil, err
	}
	return &operationsOutput{Body: result}, nil
}

func (h *Handler) lock(ctx context.Context, in *canvasInput) (*lockOutput, error) {
	result, err := h.svc.SetLock(ctx, in.WsID, in.CanvasID, true)
	if err != nil {
		return nil, err
	}
	return &lockOutput{Body: result}, nil
}

func (h *Handler) unlock(ctx context.Context, in *canvasInput) (*lockOutput, error) {
	result, err := h.svc.SetLock(ctx, in.WsID, in.CanvasID, false)
	if err != nil {
		return nil, err
	}
	return &lockOutput{Body: result}, nil
}

func (h *Handler) registerOperations(api huma.API) {
	huma.Register(api, httpx.Op(authz.CanvasEdit, http.MethodPost, "/workspaces/{wsId}/canvases/{canvasId}/operations",
		"applyCanvasOperations", "Canvases", "Apply a batch of canvas edits atomically",
		httpx.Describe("Place, move, restyle and remove items, and create/update/delete free shapes.\nAll operations succeed or none do. Used for drag-and-drop, multi-select moves and undo/redo.\nOther viewers receive the result as SSE events."),
		httpx.Errors(403, 404, 409, 422, 423)), h.applyOperations)
	huma.Register(api, httpx.Op(authz.CanvasLock, http.MethodPost, "/workspaces/{wsId}/canvases/{canvasId}/lock",
		"lockCanvas", "Canvases", "Lock a canvas against edits", httpx.Errors(403, 404)), h.lock)
	huma.Register(api, httpx.Op(authz.CanvasUnlock, http.MethodDelete, "/workspaces/{wsId}/canvases/{canvasId}/lock",
		"unlockCanvas", "Canvases", "Unlock a canvas", httpx.Errors(403, 404)), h.unlock)
}
