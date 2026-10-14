// Package events streams outbox changes through the same visibility and redaction as reads.
package events

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	platformevents "github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type (
	InventoryReader interface {
		ReadForActor(context.Context, *dbgen.Queries, *authz.Actor, apimodel.IDs) (apimodel.Snapshot, error)
	}
	ResourceReader func(context.Context, uuid.UUID, uuid.UUID, bool) (any, error)
	MemberReader   func(context.Context, uuid.UUID, uuid.UUID) (any, error)
	Service        struct {
		db        *pg.DB
		authz     *authz.Resolver
		broker    *platformevents.Broker
		inventory InventoryReader
		canvas    ResourceReader
		member    MemberReader
	}
)

func NewService(db *pg.DB, az *authz.Resolver, broker *platformevents.Broker, inventory InventoryReader, canvas ResourceReader, member MemberReader) *Service {
	return &Service{db: db, authz: az, broker: broker, inventory: inventory, canvas: canvas, member: member}
}

type Event struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type" enum:"node.created,node.updated,node.deleted,link.created,link.updated,link.deleted,area.created,area.updated,area.deleted,note.created,note.updated,note.deleted,canvas.created,canvas.updated,canvas.deleted,canvas.operations_applied,health.changed,member.changed"`
	WorkspaceID uuid.UUID  `json:"workspaceId"`
	CanvasID    *uuid.UUID `json:"canvasId,omitempty"`
	ActorID     *uuid.UUID `json:"actorId,omitempty"`
	OccurredAt  time.Time  `json:"occurredAt"`
	Data        any        `json:"data"`
}
type Stream struct {
	svc      *Service
	wsID     uuid.UUID
	canvasID *uuid.UUID
	after    uuid.UUID
	sub      *platformevents.Subscription
}

func (s *Service) Open(ctx context.Context, wsID uuid.UUID, canvasID string, lastID string) (*Stream, error) {
	if _, err := s.authz.Actor(ctx, wsID, authz.EventsStream); err != nil {
		return nil, err
	}
	stream := &Stream{svc: s, wsID: wsID}
	if canvasID != "" {
		id, err := uuid.Parse(canvasID)
		if err != nil {
			return nil, apperr.Invalid("validation", "Invalid canvas ID.")
		}
		if _, err = s.canvas(ctx, wsID, id, false); err != nil {
			return nil, err
		}
		stream.canvasID = &id
	}
	if lastID != "" {
		id, err := uuid.Parse(lastID)
		if err != nil {
			return nil, apperr.Invalid("validation", "Invalid event ID.")
		}
		stream.after = id
	}
	stream.sub = s.broker.Subscribe(wsID)
	return stream, nil
}
func (s *Stream) Close() { s.sub.Close() }

func (s *Stream) Run(ctx context.Context, send func(*Event) error) error {
	defer s.Close()
	replayed := map[uuid.UUID]bool{}
	// Subscribe before replay, so a commit between the two is buffered rather than lost.
	if s.after != uuid.Nil {
		for {
			var rows []dbgen.WorkspaceEvent
			err := s.svc.db.InWorkspace(ctx, s.wsID, func(q *dbgen.Queries) error {
				var err error
				rows, err = q.ListWorkspaceEventsAfter(ctx, dbgen.ListWorkspaceEventsAfterParams{WorkspaceID: s.wsID, ID: s.after})
				return err
			})
			if err != nil {
				return err
			}
			for _, row := range rows {
				replayed[row.ID] = true
				event, err := s.svc.project(ctx, row, s.canvasID)
				if err != nil {
					return err
				}
				if event != nil {
					if err = send(event); err != nil {
						return err
					}
				}
				s.after = row.ID
			}
			if len(rows) < 500 {
				break
			}
		}
	}
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ref, ok := <-s.sub.C:
			if !ok {
				return nil
			}
			// Transactions can commit out of UUID order. Skip only rows actually replayed.
			if replayed[ref.EventID] {
				delete(replayed, ref.EventID)
				continue
			}
			var row dbgen.WorkspaceEvent
			err := s.svc.db.InWorkspace(ctx, s.wsID, func(q *dbgen.Queries) error {
				var err error
				row, err = q.GetWorkspaceEvent(ctx, dbgen.GetWorkspaceEventParams{WorkspaceID: s.wsID, ID: ref.EventID})
				return err
			})
			if pg.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			event, err := s.svc.project(ctx, row, s.canvasID)
			if err != nil {
				return err
			}
			if event != nil {
				if err = send(event); err != nil {
					return err
				}
			}
			s.after = row.ID
		case <-heartbeat.C:
			if _, err := s.svc.authz.Actor(ctx, s.wsID, authz.EventsStream); err != nil {
				return err
			}
			if s.canvasID != nil {
				if _, err := s.svc.canvas(ctx, s.wsID, *s.canvasID, false); err != nil {
					return err
				}
			}
			if err := send(nil); err != nil {
				return err
			}
		}
	}
}

func (s *Service) project(ctx context.Context, row dbgen.WorkspaceEvent, canvasID *uuid.UUID) (*Event, error) {
	actor, err := s.authz.Actor(ctx, row.WorkspaceID, authz.EventsStream)
	if err != nil {
		return nil, err
	}
	if canvasID != nil {
		if _, err = s.canvas(ctx, row.WorkspaceID, *canvasID, false); err != nil {
			return nil, err
		}
		if len(row.CanvasIds) > 0 && !slices.Contains(row.CanvasIds, *canvasID) {
			return nil, nil
		}
	}
	event := &Event{ID: row.ID, Type: row.Type, WorkspaceID: row.WorkspaceID, ActorID: row.ActorID, OccurredAt: row.CreatedAt, Data: map[string]any{}}
	if canvasID != nil {
		event.CanvasID = canvasID
	} else if len(row.CanvasIds) == 1 {
		event.CanvasID = &row.CanvasIds[0]
	}
	if row.SubjectID == nil {
		return event, nil
	}
	id := *row.SubjectID
	if strings.HasSuffix(row.Type, ".deleted") {
		// Deleted subjects cannot be re-read. Never reveal their existence outside editor grants.
		if actor.Role == authz.RoleEditor {
			visible := false
			for _, canvas := range row.CanvasIds {
				if _, err := s.canvas(ctx, row.WorkspaceID, canvas, false); err == nil {
					visible = true
					break
				} else if !apperr.IsKind(err, apperr.KindNotFound) {
					return nil, err
				}
			}
			if !visible {
				return nil, nil
			}
		}
		event.Data = map[string]any{"id": id}
		return event, nil
	}
	switch {
	case strings.HasPrefix(row.Type, "canvas."):
		event.Data, err = s.canvas(ctx, row.WorkspaceID, id, row.Type == "canvas.operations_applied")
	case row.Type == "member.changed":
		event.Data, err = s.member(ctx, row.WorkspaceID, id)
	default:
		ids := apimodel.IDs{}
		switch {
		case strings.HasPrefix(row.Type, "node."), row.Type == "health.changed":
			ids.Nodes = []uuid.UUID{id}
		case strings.HasPrefix(row.Type, "area."):
			ids.Areas = []uuid.UUID{id}
		case strings.HasPrefix(row.Type, "link."):
			ids.Links = []uuid.UUID{id}
		case strings.HasPrefix(row.Type, "note."):
			ids.Notes = []uuid.UUID{id}
		}
		var snapshot apimodel.Snapshot
		err = s.db.InWorkspace(ctx, row.WorkspaceID, func(q *dbgen.Queries) error {
			var err error
			snapshot, err = s.inventory.ReadForActor(ctx, q, actor, ids)
			return err
		})
		if err == nil {
			switch {
			case len(snapshot.Nodes) > 0:
				event.Data = snapshot.Nodes[0]
			case len(snapshot.Areas) > 0:
				event.Data = snapshot.Areas[0]
			case len(snapshot.Links) > 0:
				event.Data = snapshot.Links[0]
			case len(snapshot.Notes) > 0:
				event.Data = snapshot.Notes[0]
			default:
				return nil, nil
			}
		}
	}
	if apperr.IsKind(err, apperr.KindNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Avoid exposing canvas IDs that an editor cannot GET.
	if actor.Role == authz.RoleEditor && event.CanvasID != nil {
		if _, err = s.canvas(ctx, row.WorkspaceID, *event.CanvasID, false); apperr.IsKind(err, apperr.KindNotFound) {
			event.CanvasID = nil
		} else if err != nil {
			return nil, err
		}
	}
	return event, nil
}
