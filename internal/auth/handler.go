package auth

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
	huma.Register(api, httpx.Op(authz.AuthLogin, http.MethodPost, "/auth/login",
		"login", "Setup & auth", "Sign in", httpx.Errors(http.StatusUnauthorized, http.StatusTooManyRequests)), h.login)
	huma.Register(api, httpx.Op(authz.AuthProviders, http.MethodGet, "/auth/providers",
		"listAuthProviders", "Setup & auth", "Which sign-in methods are enabled",
		httpx.Describe("Google and Discord appear only when the server has OAuth client credentials configured.")), h.providers)
	huma.Register(api, httpx.Op(authz.AuthLogout, http.MethodPost, "/auth/logout",
		"logout", "Setup & auth", "Sign out (ends the current session)",
		httpx.Status(http.StatusNoContent), httpx.Errors(http.StatusUnauthorized)), h.logout)
}

type (
	loginInput  struct{ Body LoginRequest }
	loginOutput struct {
		SetCookie http.Cookie `header:"Set-Cookie"`
		Body      apimodel.Account
	}
)

func (h *Handler) login(ctx context.Context, in *loginInput) (*loginOutput, error) {
	account, cookie, err := h.svc.Login(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	return &loginOutput{SetCookie: cookie, Body: account}, nil
}

type providersOutput struct{ Body AuthProviders }

func (h *Handler) providers(ctx context.Context, _ *struct{}) (*providersOutput, error) {
	return &providersOutput{Body: h.svc.Providers(ctx)}, nil
}

type logoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

func (h *Handler) logout(ctx context.Context, _ *struct{}) (*logoutOutput, error) {
	cookie, err := h.svc.Logout(ctx)
	if err != nil {
		return nil, err
	}
	return &logoutOutput{SetCookie: cookie}, nil
}
