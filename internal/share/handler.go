package share

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpx.Op(authz.ShareLinkList, http.MethodGet, "/workspaces/{wsId}/canvases/{canvasId}/share-links", "listShareLinks", "Share links", "List share links of a canvas", httpx.Errors(403, 404)), h.list)
	huma.Register(api, httpx.Op(authz.ShareLinkCreate, http.MethodPost, "/workspaces/{wsId}/canvases/{canvasId}/share-links", "createShareLink", "Share links", "Create a guest link for this canvas", httpx.Describe("The token is returned **only once**, in this response. Only its hash is stored."), httpx.Status(201), httpx.Errors(403, 404, 422)), h.create)
	huma.Register(api, httpx.Op(authz.ShareLinkUpdate, http.MethodPatch, "/workspaces/{wsId}/share-links/{shareLinkId}", "updateShareLink", "Share links", "Change label, password or expiry", httpx.Errors(403, 404)), h.update)
	huma.Register(api, httpx.Op(authz.ShareLinkRevoke, http.MethodDelete, "/workspaces/{wsId}/share-links/{shareLinkId}", "revokeShareLink", "Share links", "Revoke a share link (it stops working immediately)", httpx.Status(204), httpx.Errors(403, 404)), h.revoke)
	huma.Register(api, httpx.Op(authz.ShareInfo, http.MethodGet, "/share/{token}", "getShareInfo", "Guest", "Check a share link before opening it", httpx.Describe("Reveals nothing about the canvas until the link is unlocked."), httpx.Errors(404)), h.info)
	huma.Register(api, httpx.Op(authz.ShareUnlock, http.MethodPost, "/share/{token}/unlock", "unlockShare", "Guest", "Open a share link (with its password, if it has one)", httpx.Status(204), httpx.Errors(401, 404, 429)), h.unlock)
	huma.Register(api, httpx.Op(authz.ShareView, http.MethodGet, "/share/{token}/view", "getSharedCanvasView", "Guest", "Read-only view of the shared canvas (guest redaction applied)", httpx.Errors(401, 404)), h.view)
}

type (
	CanvasInput struct {
		WsID     uuid.UUID `path:"wsId"`
		CanvasID uuid.UUID `path:"canvasId"`
	}
	listOutput struct{ Body ShareLinkPage }
)

func (h *Handler) list(ctx context.Context, in *CanvasInput) (*listOutput, error) {
	out, err := h.svc.List(ctx, in.WsID, in.CanvasID)
	if err != nil {
		return nil, err
	}
	return &listOutput{Body: out}, nil
}

type (
	createInput struct {
		CanvasInput
		Body ShareLinkCreate
	}
	createOutput struct{ Body ShareLinkCreated }
)

func (h *Handler) create(ctx context.Context, in *createInput) (*createOutput, error) {
	out, err := h.svc.Create(ctx, in.WsID, in.CanvasID, in.Body)
	if err != nil {
		return nil, err
	}
	return &createOutput{Body: out}, nil
}

type (
	LinkInput struct {
		WsID uuid.UUID `path:"wsId"`
		ID   uuid.UUID `path:"shareLinkId"`
	}
	updateInput struct {
		LinkInput
		Body ShareLinkUpdate
	}
	updateOutput struct{ Body ShareLink }
)

func (h *Handler) update(ctx context.Context, in *updateInput) (*updateOutput, error) {
	out, err := h.svc.Update(ctx, in.WsID, in.ID, in.Body)
	if err != nil {
		return nil, err
	}
	return &updateOutput{Body: out}, nil
}

func (h *Handler) revoke(ctx context.Context, in *LinkInput) (*struct{}, error) {
	return nil, h.svc.Revoke(ctx, in.WsID, in.ID)
}

type (
	TokenInput struct {
		Token string `path:"token" minLength:"32"`
	}
	infoOutput struct{ Body ShareInfo }
)

func (h *Handler) info(ctx context.Context, in *TokenInput) (*infoOutput, error) {
	out, err := h.svc.Info(ctx, in.Token)
	if err != nil {
		return nil, err
	}
	return &infoOutput{Body: out}, nil
}

type (
	unlockInput struct {
		TokenInput
		Body *ShareUnlock `required:"false"`
	}
	unlockOutput struct {
		SetCookie http.Cookie `header:"Set-Cookie"`
	}
)

func (h *Handler) unlock(ctx context.Context, in *unlockInput) (*unlockOutput, error) {
	body := ShareUnlock{}
	if in.Body != nil {
		body = *in.Body
	}
	cookie, err := h.svc.Unlock(ctx, in.Token, body)
	if err != nil {
		return nil, err
	}
	return &unlockOutput{SetCookie: cookie}, nil
}

type viewInput struct {
	TokenInput
	CookieValue string
}

func (in *viewInput) Resolve(ctx huma.Context) []error {
	request := &http.Request{Header: http.Header{"Cookie": []string{ctx.Header("Cookie")}}}
	if cookie, err := request.Cookie(httpx.GuestCookieName); err == nil {
		in.CookieValue = cookie.Value
	}
	return nil
}

type viewOutput struct{ Body apimodel.CanvasView }

func (h *Handler) view(ctx context.Context, in *viewInput) (*viewOutput, error) {
	out, err := h.svc.View(ctx, in.Token, in.CookieValue)
	if err != nil {
		return nil, err
	}
	return &viewOutput{Body: out}, nil
}
