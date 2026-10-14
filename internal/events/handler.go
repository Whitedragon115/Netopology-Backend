package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }
func (h *Handler) Register(api huma.API) {
	api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[Event](), true, "Event")
	op := httpx.Op(authz.EventsStream, http.MethodGet, "/workspaces/{wsId}/events", "streamEvents", "Events", "Live updates for this workspace (Server-Sent Events)", httpx.Errors(404))
	op.Responses = map[string]*huma.Response{"200": {Description: "Event stream", Content: map[string]*huma.MediaType{"text/event-stream": {Schema: &huma.Schema{Type: "string", Description: "SSE stream; see the Event schema for the payload of each data: line."}}}}}
	op.OperationID = "streamEvents"
	op.Extensions["x-sse-event-schema"] = &huma.Schema{Ref: "#/components/schemas/Event"}
	op.SkipValidateBody = true
	huma.Register(api, op, h.stream)
}

type streamInput struct {
	WsID     uuid.UUID `path:"wsId"`
	CanvasID string    `query:"canvasId" format:"uuid"`
	LastID   string    `header:"Last-Event-ID"`
}

func (h *Handler) stream(ctx context.Context, in *streamInput) (*huma.StreamResponse, error) {
	stream, err := h.svc.Open(ctx, in.WsID, in.CanvasID, in.LastID)
	if err != nil {
		return nil, err
	}
	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		defer stream.Close()
		hctx.SetHeader("Content-Type", "text/event-stream")
		hctx.SetHeader("Cache-Control", "no-cache")
		hctx.SetHeader("X-Accel-Buffering", "no")
		hctx.SetStatus(200)
		writer := hctx.BodyWriter()
		rw, ok := writer.(http.ResponseWriter)
		if !ok {
			return
		}
		controller := http.NewResponseController(rw)
		_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := fmt.Fprint(writer, ": ok\n\n"); err != nil {
			return
		}
		if controller.Flush() != nil {
			return
		}
		_ = stream.Run(ctx, func(event *Event) error {
			_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if event == nil {
				if _, err := fmt.Fprint(writer, ": heartbeat\n\n"); err != nil {
					return err
				}
			} else {
				raw, err := json.Marshal(event)
				if err != nil {
					return err
				}
				if _, err = fmt.Fprintf(writer, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, raw); err != nil {
					return err
				}
			}
			return controller.Flush()
		})
	}}, nil
}
