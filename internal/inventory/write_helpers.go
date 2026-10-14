package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

var fieldKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

func checkedFields(in []Field, old json.RawMessage) (json.RawMessage, error) {
	if in == nil {
		return json.RawMessage(`[]`), nil
	}
	previous := map[string]Field{}
	if len(old) > 0 {
		var fields []Field
		if err := json.Unmarshal(old, &fields); err != nil {
			return nil, err
		}
		for _, field := range fields {
			previous[field.Key] = field
		}
	}
	keys := map[string]bool{}
	for i := range in {
		f := &in[i]
		if !fieldKey.MatchString(f.Key) || len(f.Label) > 60 || f.Label == "" || keys[f.Key] || (f.Value != nil && len(*f.Value) > 2000) {
			return nil, apperr.Invalid(apperr.CodeValidation, "Invalid inventory field.")
		}
		keys[f.Key] = true
		if f.Redacted {
			oldField, ok := previous[f.Key]
			if !ok || !oldField.Sensitive || !f.Sensitive || f.Value != nil {
				return nil, apperr.Invalid(apperr.CodeValidation, "Invalid redacted field.")
			}
			f.Value = oldField.Value
			f.Redacted = false
		}
	}
	return json.Marshal(in)
}

func parsedCIDR(raw *string) (*netip.Prefix, error) {
	if raw == nil {
		return nil, nil
	}
	cidr, err := netip.ParsePrefix(*raw)
	if err != nil {
		return nil, apperr.Invalid(apperr.CodeValidation, "Invalid CIDR.")
	}
	return &cidr, nil
}

func choose(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func idIn(ids []uuid.UUID, id uuid.UUID) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func visibleItem(vis visibility, kind string, id uuid.UUID) bool {
	if vis.all {
		return true
	}
	switch kind {
	case "area":
		return idIn(vis.ids.AreaIds, id)
	case "node":
		return idIn(vis.ids.NodeIds, id)
	case "link":
		return idIn(vis.ids.LinkIds, id)
	case "note":
		return idIn(vis.ids.NoteIds, id)
	case "canvas":
		return idIn(vis.ids.CanvasIds, id)
	}
	return false
}

func requireVisible(vis visibility, kind string, id uuid.UUID) error {
	if !visibleItem(vis, kind, id) {
		return apperr.ErrNotFound()
	}
	return nil
}

func checkMatch(version int32, match *int32) error {
	if match != nil && version != *match {
		return apperr.PreconditionFailed()
	}
	return nil
}

func (s *Service) preparePlaceOn(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, place *PlaceOn) error {
	if place == nil {
		if actor.Role == authz.RoleEditor {
			return apperr.Invalid("inventory.place_on_required", "Editors must place new inventory on a granted canvas.")
		}
		return nil
	}
	if place.CanvasID == uuid.Nil {
		return apperr.Invalid(apperr.CodeValidation, "Invalid canvas id.")
	}
	if place.Width != nil && *place.Width <= 0 || place.Height != nil && *place.Height <= 0 {
		return apperr.Invalid(apperr.CodeValidation, "Placement size must be positive.")
	}
	_, err := q.GetInventoryCanvas(ctx, dbgen.GetInventoryCanvasParams{WorkspaceID: actor.WorkspaceID, ID: place.CanvasID})
	if pg.IsNotFound(err) {
		return apperr.ErrNotFound()
	}
	if err != nil {
		return err
	}
	if actor.Role == authz.RoleEditor {
		granted, err := q.IsGrantedTabCanvas(ctx, dbgen.IsGrantedTabCanvasParams{WorkspaceID: actor.WorkspaceID, CanvasID: place.CanvasID, UserID: actor.UserID()})
		if err != nil {
			return err
		}
		if !granted {
			return apperr.ErrNotFound()
		}
	}
	return nil
}

func insertPlaceOn(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, place *PlaceOn, kind string, id uuid.UUID) error {
	if place == nil {
		return nil
	}
	style := json.RawMessage(`{}`)
	if place.Style != nil {
		var err error
		style, err = json.Marshal(place.Style)
		if err != nil {
			return err
		}
	}
	args := dbgen.InsertInventoryPlacementParams{
		WorkspaceID: wsID, CanvasID: place.CanvasID,
		ParentPlacementID: place.ParentPlacementID, X: place.X, Y: place.Y, Width: place.Width, Height: place.Height, Style: style,
	}
	switch kind {
	case "area":
		args.AreaID = &id
	case "node":
		args.NodeID = &id
	case "note":
		args.NoteID = &id
	}
	return q.InsertInventoryPlacement(ctx, args)
}

func usedElsewhere(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, kind string, id uuid.UUID) error {
	if actor.Role != authz.RoleEditor {
		return nil
	}
	used, err := q.InventoryUsedOutsideGrants(ctx, dbgen.InventoryUsedOutsideGrantsParams{WorkspaceID: actor.WorkspaceID, Kind: kind, ID: &id, UserID: actor.UserID()})
	if err != nil {
		return err
	}
	if used {
		return apperr.Conflict("inventory.used_elsewhere", "This item is used on a canvas you cannot edit.")
	}
	return nil
}

func recordInventoryChange(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, permission authz.Permission, kind string, event events.Type, id uuid.UUID, name string, canvasIDs []uuid.UUID) error {
	if canvasIDs == nil {
		var err error
		canvasIDs, err = q.InventoryCanvasIDs(ctx, dbgen.InventoryCanvasIDsParams{WorkspaceID: actor.WorkspaceID, Kind: kind, ID: &id})
		if err != nil {
			return err
		}
	}
	if err := auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &actor.WorkspaceID, Action: permission, TargetType: kind, TargetID: &id, Metadata: map[string]any{"name": name}}); err != nil {
		return err
	}
	return events.Emit(ctx, q, events.Change{WorkspaceID: actor.WorkspaceID, Type: event, SubjectID: &id, CanvasIDs: canvasIDs})
}

func inventoryWriteError(err error) error {
	if err == nil {
		return nil
	}
	if constraint, ok := pg.UniqueViolation(err); ok && strings.Contains(constraint, "slug") {
		return apperr.Conflict(apperr.CodeSlugTaken, "That node slug is already used.")
	}
	if constraint, ok := pg.CheckViolation(err); ok {
		switch constraint {
		case "areas_no_cycle":
			return apperr.Conflict("area.cycle", "An area cannot be nested inside its descendant.")
		default:
			return apperr.Invalid(apperr.CodeValidation, fmt.Sprintf("Invalid inventory value (%s).", constraint))
		}
	}
	if _, ok := pg.ForeignKeyViolation(err); ok {
		return apperr.ErrNotFound()
	}
	return err
}
