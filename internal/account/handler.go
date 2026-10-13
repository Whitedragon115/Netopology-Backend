package account

import (
	"context"
	"net/http"

	"github.com/google/uuid"

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
	huma.Register(api, httpx.Op(authz.MeUpdate, http.MethodPatch, "/me",
		"updateMe", "Me", "Update my profile", httpx.Errors(401, 422)), h.updateMe)
	huma.Register(api, httpx.Op(authz.MePassword, http.MethodPost, "/me/password",
		"changeMyPassword", "Me", "Change my password", httpx.Describe("Requires the current password. Ends all other sessions."),
		httpx.Status(204), httpx.Errors(401, 422)), h.changeMyPassword)
	huma.Register(api, httpx.Op(authz.MeIdentityList, http.MethodGet, "/me/identities",
		"listMyIdentities", "Me", "My linked Google/Discord sign-ins", httpx.Errors(401)), h.listMyIdentities)
	huma.Register(api, httpx.Op(authz.MeIdentityUnlink, http.MethodDelete, "/me/identities/{identityId}",
		"unlinkMyIdentity", "Me", "Unlink a sign-in (not the last way to sign in)", httpx.Status(204), httpx.Errors(401, 404, 409)), h.unlinkMyIdentity)
	huma.Register(api, httpx.Op(authz.UserList, http.MethodGet, "/users",
		"listUsers", "Instance", "List accounts", httpx.Errors(403)), h.listUsers)
	huma.Register(api, httpx.Op(authz.UserCreate, http.MethodPost, "/users",
		"createUser", "Instance", "Create an account", httpx.Status(201), httpx.Errors(403, 409, 422)), h.createUser)
	huma.Register(api, httpx.Op(authz.UserLookup, http.MethodGet, "/users/lookup",
		"lookupUser", "Instance", "Find a user by exact username (to add as a member)",
		httpx.Describe("Available to owners and admins of any workspace. Returns public profile only."), httpx.Errors(403, 404)), h.lookupUser)
	huma.Register(api, httpx.Op(authz.UserRead, http.MethodGet, "/users/{userId}",
		"getUser", "Instance", "Get an account", httpx.Errors(403, 404)), h.getUser)
	huma.Register(api, httpx.Op(authz.UserUpdate, http.MethodPatch, "/users/{userId}",
		"updateUser", "Instance", "Update an account (disable, change server role, ...)", httpx.Errors(403, 404, 409)), h.updateUser)
	huma.Register(api, httpx.Op(authz.UserPasswordReset, http.MethodPost, "/users/{userId}/password",
		"resetUserPassword", "Instance", "Set a new password for an account",
		httpx.Describe("Ends all of that user's sessions."), httpx.Status(204), httpx.Errors(403, 404)), h.resetUserPassword)
	huma.Register(api, httpx.Op(authz.InstanceSettingsRead, http.MethodGet, "/instance/settings",
		"getInstanceSettings", "Instance", "Get server settings", httpx.Errors(403)), h.getInstanceSettings)
	huma.Register(api, httpx.Op(authz.InstanceSettingsWrite, http.MethodPatch, "/instance/settings",
		"updateInstanceSettings", "Instance", "Update server settings", httpx.Errors(403)), h.updateInstanceSettings)
}

type getMeOutput struct{ Body apimodel.Account }

func (h *Handler) getMe(ctx context.Context, _ *struct{}) (*getMeOutput, error) {
	account, err := h.svc.GetMe(ctx)
	if err != nil {
		return nil, err
	}
	return &getMeOutput{Body: account}, nil
}

type updateMeInput struct{ Body MeUpdate }

func (h *Handler) updateMe(ctx context.Context, in *updateMeInput) (*getMeOutput, error) {
	account, err := h.svc.UpdateMe(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	return &getMeOutput{Body: account}, nil
}

type changeMyPasswordInput struct{ Body PasswordChange }

func (h *Handler) changeMyPassword(ctx context.Context, in *changeMyPasswordInput) (*struct{}, error) {
	if err := h.svc.ChangeMyPassword(ctx, in.Body); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

type listMyIdentitiesOutput struct {
	Body struct {
		Items       []Identity `json:"items"`
		HasPassword bool       `json:"hasPassword"`
	}
}

func (h *Handler) listMyIdentities(ctx context.Context, _ *struct{}) (*listMyIdentitiesOutput, error) {
	items, hasPassword, err := h.svc.ListMyIdentities(ctx)
	if err != nil {
		return nil, err
	}
	out := &listMyIdentitiesOutput{}
	out.Body.Items, out.Body.HasPassword = items, hasPassword
	return out, nil
}

type unlinkMyIdentityInput struct {
	IdentityID uuid.UUID `path:"identityId"`
}

func (h *Handler) unlinkMyIdentity(ctx context.Context, in *unlinkMyIdentityInput) (*struct{}, error) {
	if err := h.svc.UnlinkMyIdentity(ctx, in.IdentityID); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

type listUsersInput struct {
	httpx.PageParams
	Search string `query:"q"`
}
type listUsersOutput struct{ Body AccountPage }

func (h *Handler) listUsers(ctx context.Context, in *listUsersInput) (*listUsersOutput, error) {
	page, err := h.svc.ListUsers(ctx, in.PageParams, in.Search)
	if err != nil {
		return nil, err
	}
	return &listUsersOutput{Body: page}, nil
}

type (
	createUserInput struct{ Body AccountCreate }
	accountOutput   struct{ Body apimodel.Account }
)

func (h *Handler) createUser(ctx context.Context, in *createUserInput) (*accountOutput, error) {
	account, err := h.svc.CreateUser(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	return &accountOutput{Body: account}, nil
}

type lookupUserInput struct {
	Username string `query:"username" required:"true"`
}
type lookupUserOutput struct{ Body User }

func (h *Handler) lookupUser(ctx context.Context, in *lookupUserInput) (*lookupUserOutput, error) {
	user, err := h.svc.LookupUser(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	return &lookupUserOutput{Body: user}, nil
}

type userIDInput struct {
	UserID uuid.UUID `path:"userId"`
}

func (h *Handler) getUser(ctx context.Context, in *userIDInput) (*accountOutput, error) {
	account, err := h.svc.GetUser(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	return &accountOutput{Body: account}, nil
}

type updateUserInput struct {
	UserID uuid.UUID `path:"userId"`
	Body   AccountUpdate
}

func (h *Handler) updateUser(ctx context.Context, in *updateUserInput) (*accountOutput, error) {
	account, err := h.svc.UpdateUser(ctx, in.UserID, in.Body)
	if err != nil {
		return nil, err
	}
	return &accountOutput{Body: account}, nil
}

type resetUserPasswordInput struct {
	UserID uuid.UUID `path:"userId"`
	Body   PasswordReset
}

func (h *Handler) resetUserPassword(ctx context.Context, in *resetUserPasswordInput) (*struct{}, error) {
	if err := h.svc.ResetUserPassword(ctx, in.UserID, in.Body); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

type instanceSettingsOutput struct{ Body InstanceSettings }

func (h *Handler) getInstanceSettings(ctx context.Context, _ *struct{}) (*instanceSettingsOutput, error) {
	value, err := h.svc.GetInstanceSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &instanceSettingsOutput{Body: value}, nil
}

type updateInstanceSettingsInput struct{ Body InstanceSettings }

func (h *Handler) updateInstanceSettings(ctx context.Context, in *updateInstanceSettingsInput) (*instanceSettingsOutput, error) {
	value, err := h.svc.UpdateInstanceSettings(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	return &instanceSettingsOutput{Body: value}, nil
}
