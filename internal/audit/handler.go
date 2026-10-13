package audit

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }
func (h *Handler) Register(api huma.API) {
	huma.Register(api, httpx.Op(authz.AuditList, http.MethodGet, "/workspaces/{wsId}/audit-log", "listAuditLog", "Audit", "Who did what, newest first", httpx.Errors(403, 404)), h.list)
}

type listInput struct {
	WsID     uuid.UUID `path:"wsId"`
	Cursor   string    `query:"cursor"`
	Limit    int       `query:"limit" default:"50" minimum:"1" maximum:"200"`
	ActorID  string    `query:"actorId" format:"uuid"`
	Action   string    `query:"action" doc:"Exact key or prefix, e.g. node."`
	TargetID string    `query:"targetId" format:"uuid"`
	Since    string    `query:"since" format:"date-time"`
}
type listOutput struct{ Body AuditPage }

func (h *Handler) list(ctx context.Context, in *listInput) (*listOutput, error) {
	page, err := h.svc.List(ctx, *in)
	if err != nil {
		return nil, err
	}
	return &listOutput{Body: page}, nil
}
