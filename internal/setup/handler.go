package setup

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

// Handler is the HTTP layer: parse (Huma does it from the input structs), call the service,
// return. No business rules and no SQL here. Errors from the service are returned as-is;
// httpx turns them into problem+json.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register adds this feature's operations. Paths are relative to /api/v1.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpx.Op(authz.SetupStatus, http.MethodGet, "/setup",
		"getSetupStatus", "Setup & auth", "Check whether first-run setup is needed"), h.status)

	huma.Register(api, httpx.Op(authz.SetupComplete, http.MethodPost, "/setup",
		"completeSetup", "Setup & auth", "Create the first account (becomes server owner)",
		httpx.Describe("Only works while no account exists. Optionally creates the first workspace. Signs the new user in."),
		httpx.Status(http.StatusCreated), httpx.Errors(http.StatusConflict, http.StatusUnprocessableEntity)), h.complete)
}

type statusOutput struct {
	Body SetupStatus
}

func (h *Handler) status(ctx context.Context, _ *struct{}) (*statusOutput, error) {
	st, err := h.svc.Status(ctx)
	if err != nil {
		return nil, err
	}
	return &statusOutput{Body: st}, nil
}

type completeInput struct {
	Body SetupRequest
}

type completeOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      apimodel.Account
}

func (h *Handler) complete(ctx context.Context, in *completeInput) (*completeOutput, error) {
	acct, cookie, err := h.svc.Complete(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	return &completeOutput{SetCookie: cookie, Body: acct}, nil
}
