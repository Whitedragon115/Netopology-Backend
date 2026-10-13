package account

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpx.Op(authz.MeRead, http.MethodGet, "/me",
		"getMe", "Me", "Get the signed-in account", httpx.Errors(http.StatusUnauthorized)), h.getMe)
}

type getMeOutput struct{ Body apimodel.Account }

func (h *Handler) getMe(ctx context.Context, _ *struct{}) (*getMeOutput, error) {
	account, err := h.svc.GetMe(ctx)
	if err != nil {
		return nil, err
	}
	return &getMeOutput{Body: account}, nil
}
