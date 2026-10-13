// Package events is the write side and the fan-out of live updates (SSE).
// Design: docs/ARCHITECTURE.md §9.
//
//	service (in pg.InWorkspace) --Emit--> workspace_events row --trigger--> pg_notify("netopology_events")
//	Broker (one per api process) LISTENs --> Subscriber channels (one per open SSE connection)
//	internal/events handler: for each Ref, re-reads the subject AS THE SUBSCRIBER (same
//	visibility + redaction as a GET) and writes `id/event/data` to the stream.
package events

import (
	"context"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// Type is Event.type in the contract (components.schemas.Event.type enum).
type Type string

const (
	NodeCreated   Type = "node.created"
	NodeUpdated   Type = "node.updated"
	NodeDeleted   Type = "node.deleted"
	LinkCreated   Type = "link.created"
	LinkUpdated   Type = "link.updated"
	LinkDeleted   Type = "link.deleted"
	AreaCreated   Type = "area.created"
	AreaUpdated   Type = "area.updated"
	AreaDeleted   Type = "area.deleted"
	NoteCreated   Type = "note.created"
	NoteUpdated   Type = "note.updated"
	NoteDeleted   Type = "note.deleted"
	CanvasCreated Type = "canvas.created"
	CanvasUpdated Type = "canvas.updated" // also lock/unlock and settings changes
	CanvasDeleted Type = "canvas.deleted"
	// CanvasOperationsApplied: placements/shapes changed; data = the applied operations result.
	CanvasOperationsApplied Type = "canvas.operations_applied"
	HealthChanged           Type = "health.changed"
	MemberChanged           Type = "member.changed"
	// Keep this list identical to the contract's Event.type enum (checked by the contract test).
)

// AllTypes is used by the contract test.
var AllTypes = []Type{
	NodeCreated, NodeUpdated, NodeDeleted, LinkCreated, LinkUpdated, LinkDeleted,
	AreaCreated, AreaUpdated, AreaDeleted, NoteCreated, NoteUpdated, NoteDeleted,
	CanvasCreated, CanvasUpdated, CanvasDeleted, CanvasOperationsApplied, HealthChanged, MemberChanged,
}

// Change describes one change, written in the same transaction as the change itself.
type Change struct {
	WorkspaceID uuid.UUID
	Type        Type
	SubjectID   *uuid.UUID  // the changed item
	CanvasIDs   []uuid.UUID // canvases that show it (for ?canvasId= filtering); nil = workspace-wide
}

// Emit records c. Call it inside pg.InWorkspace, after the change succeeded.
func Emit(ctx context.Context, q *dbgen.Queries, c Change) error {
	var actor *uuid.UUID
	if p := authz.PrincipalFrom(ctx); p.SignedIn() {
		id := p.UserID
		actor = &id
	}
	ids := c.CanvasIDs
	if ids == nil {
		ids = []uuid.UUID{}
	}
	_, err := q.InsertWorkspaceEvent(ctx, dbgen.InsertWorkspaceEventParams{
		WorkspaceID: c.WorkspaceID,
		Type:        string(c.Type),
		SubjectID:   c.SubjectID,
		CanvasIds:   ids,
		ActorID:     actor,
	})
	return err
}

// Ref is what the broker delivers: just ids. The subscriber loads the rest as itself.
type Ref struct {
	WorkspaceID uuid.UUID
	EventID     uuid.UUID
}
