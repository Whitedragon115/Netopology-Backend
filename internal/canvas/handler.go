package canvas

import (
	"context"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

type (
	workspaceInput struct {
		WsID uuid.UUID `path:"wsId"`
	}
	canvasInput struct {
		WsID     uuid.UUID `path:"wsId"`
		CanvasID uuid.UUID `path:"canvasId"`
	}
	areaInput struct {
		WsID   uuid.UUID `path:"wsId"`
		AreaID uuid.UUID `path:"areaId"`
	}
	listOutput   struct{ Body CanvasPage }
	canvasOutput struct {
		ETag string `header:"ETag"`
		Body Canvas
	}
	viewOutput struct{ Body CanvasView }
)

func (h *Handler) list(ctx context.Context, in *workspaceInput) (*listOutput, error) {
	page, err := h.svc.List(ctx, in.WsID)
	if err != nil {
		return nil, err
	}
	return &listOutput{Body: page}, nil
}

func (h *Handler) get(ctx context.Context, in *canvasInput) (*canvasOutput, error) {
	item, err := h.svc.Get(ctx, in.WsID, in.CanvasID)
	if err != nil {
		return nil, err
	}
	return &canvasOutput{ETag: httpx.ETag(item.Version), Body: item}, nil
}

func (h *Handler) getView(ctx context.Context, in *canvasInput) (*viewOutput, error) {
	item, err := h.svc.GetView(ctx, in.WsID, in.CanvasID)
	if err != nil {
		return nil, err
	}
	return &viewOutput{Body: item}, nil
}

func (h *Handler) getAreaView(ctx context.Context, in *areaInput) (*viewOutput, error) {
	item, err := h.svc.GetAreaView(ctx, in.WsID, in.AreaID)
	if err != nil {
		return nil, err
	}
	return &viewOutput{Body: item}, nil
}

type (
	createInput struct {
		WsID uuid.UUID `path:"wsId"`
		Body CanvasCreate
	}
	updateInput struct {
		WsID     uuid.UUID `path:"wsId"`
		CanvasID uuid.UUID `path:"canvasId"`
		IfMatch  string    `header:"If-Match"`
		Body     CanvasUpdate
	}
	deleteInput struct {
		WsID     uuid.UUID `path:"wsId"`
		CanvasID uuid.UUID `path:"canvasId"`
		IfMatch  string    `header:"If-Match"`
	}
)

func (h *Handler) create(ctx context.Context, in *createInput) (*canvasOutput, error) {
	item, err := h.svc.CanvasCreate(ctx, in.WsID, in.Body)
	if err != nil {
		return nil, err
	}
	return &canvasOutput{ETag: httpx.ETag(item.Version), Body: item}, nil
}

func (h *Handler) update(ctx context.Context, in *updateInput) (*canvasOutput, error) {
	version, present, err := httpx.IfMatch(in.IfMatch)
	if err != nil {
		return nil, err
	}
	var match *int32
	if present {
		match = &version
	}
	item, err := h.svc.CanvasUpdate(ctx, in.WsID, in.CanvasID, in.Body, match)
	if err != nil {
		return nil, err
	}
	return &canvasOutput{ETag: httpx.ETag(item.Version), Body: item}, nil
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
	if err := h.svc.Delete(ctx, in.WsID, in.CanvasID, match); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}
