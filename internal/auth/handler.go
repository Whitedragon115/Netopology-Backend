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
	huma.Register(api, httpx.Op(authz.AuthOAuthStart, http.MethodGet, "/auth/oauth/{provider}/start",
		"startOAuthSignIn", "Setup & auth", "Begin Google/Discord sign-in (browser redirect)",
		httpx.Status(302), httpx.Errors(404)), h.startOAuth)
	huma.Register(api, httpx.Op(authz.AuthOAuthCB, http.MethodGet, "/auth/oauth/{provider}/callback",
		"finishOAuthSignIn", "Setup & auth", "Provider redirects back here",
		httpx.Describe("Signs in only if the identity is linked to an existing account, or if an account's email matches the provider-**verified** email and has no identity for this provider yet (it is then linked). Never creates an account. On failure redirects to the sign-in page with `?error=identity_not_linked` (or another `code`). Also completes `me.identity_link` flows."),
		httpx.Status(302)), h.finishOAuth)
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

type startOAuthInput struct {
	Provider string `path:"provider" enum:"google,discord"`
	ReturnTo string `query:"returnTo" default:"/"`
}
type startOAuthOutput struct {
	Location  string      `header:"Location"`
	SetCookie http.Cookie `header:"Set-Cookie"`
}

func (h *Handler) startOAuth(ctx context.Context, in *startOAuthInput) (*startOAuthOutput, error) {
	location, cookie, err := h.svc.StartSignIn(ctx, in.Provider, in.ReturnTo)
	if err != nil {
		return nil, err
	}
	return &startOAuthOutput{Location: location, SetCookie: cookie}, nil
}

type finishOAuthInput struct {
	Provider string `path:"provider" enum:"google,discord"`
	Code     string `query:"code"`
	State    string `query:"state"`
	Error    string `query:"error"`
}
type finishOAuthOutput struct {
	Location  string        `header:"Location"`
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

func (h *Handler) finishOAuth(ctx context.Context, in *finishOAuthInput) (*finishOAuthOutput, error) {
	result := h.svc.FinishOAuth(ctx, in.Provider, in.Code, in.State, in.Error, oauthCookieFrom(ctx))
	return &finishOAuthOutput{Location: result.Location, SetCookie: result.Cookies}, nil
}
