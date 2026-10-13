package canvas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func marshalObject(value map[string]any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage(`{}`), nil
	}
	return json.Marshal(value)
}

func number(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func integer(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func boolean(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func operationError(index int, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := pg.CheckViolation(err); ok {
		return apperr.Conflict("canvas.operation_conflict", fmt.Sprintf("Operation %d conflicts with canvas rules.", index))
	}
	if _, ok := pg.UniqueViolation(err); ok {
		return apperr.Conflict("canvas.operation_conflict", fmt.Sprintf("Operation %d conflicts with an existing item.", index))
	}
	if _, ok := pg.ForeignKeyViolation(err); ok {
		return apperr.Conflict("canvas.operation_conflict", fmt.Sprintf("Operation %d references an unavailable item.", index))
	}
	if app, ok := apperr.As(err); ok {
		copy := *app
		copy.Detail = fmt.Sprintf("Operation %d: %s", index, app.Detail)
		return &copy
	}
	return err
}

func (s *Service) ApplyOperations(ctx context.Context, wsID, canvasID uuid.UUID, batch CanvasOperationBatch) (CanvasOperationResult, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.CanvasEdit)
	if err != nil {
		return CanvasOperationResult{}, err
	}
	if len(batch.Operations) < 1 || len(batch.Operations) > 500 {
		return CanvasOperationResult{}, apperr.Invalid(apperr.CodeValidation, "Operations must contain 1 to 500 items.")
	}
	out := CanvasOperationResult{Results: make([]CanvasOperationResultItem, 0, len(batch.Operations))}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		canvasRow, err := q.LockCanvasRow(ctx, dbgen.LockCanvasRowParams{WorkspaceID: wsID, ID: canvasID})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := s.canSeeCanvas(ctx, q, actor, canvasRow); err != nil {
			return err
		}
		if canvasRow.LockedBy != nil {
			return apperr.Locked(apperr.CodeCanvasLocked, "This canvas is locked against edits.")
		}
		var editorVisible map[string]map[uuid.UUID]bool
		if actor.Role == authz.RoleEditor {
			visible, err := q.VisibleInventoryIDs(ctx, dbgen.VisibleInventoryIDsParams{WorkspaceID: wsID, UserID: actor.UserID()})
			if err != nil {
				return err
			}
			editorVisible = map[string]map[uuid.UUID]bool{"node": {}, "area": {}, "note": {}, "link": {}}
			for _, id := range visible.NodeIds {
				editorVisible["node"][id] = true
			}
			for _, id := range visible.AreaIds {
				editorVisible["area"][id] = true
			}
			for _, id := range visible.NoteIds {
				editorVisible["note"][id] = true
			}
			for _, id := range visible.LinkIds {
				editorVisible["link"][id] = true
			}
		}
		counts := map[string]int{}
		for index, op := range batch.Operations {
			counts[op.Op]++
			result, err := s.applyOne(ctx, q, actor, canvasID, editorVisible, index, op)
			if err != nil {
				return operationError(index, err)
			}
			out.Results = append(out.Results, result)
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{
			WorkspaceID: &wsID, Action: authz.CanvasEdit,
			TargetType: "canvas", TargetID: &canvasID, Metadata: map[string]any{"operationCounts": counts},
		}); err != nil {
			return err
		}
		return events.Emit(ctx, q, events.Change{
			WorkspaceID: wsID, Type: events.CanvasOperationsApplied,
			SubjectID: &canvasID, CanvasIDs: []uuid.UUID{canvasID},
		})
	})
	if err != nil {
		return CanvasOperationResult{}, err
	}
	view, err := s.GetView(ctx, wsID, canvasID)
	if err != nil {
		return CanvasOperationResult{}, err
	}
	out.View = &view
	return out, nil
}

func (s *Service) applyOne(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID,
	visible map[string]map[uuid.UUID]bool, index int, op CanvasOperation,
) (CanvasOperationResultItem, error) {
	wsID := actor.WorkspaceID
	result := CanvasOperationResultItem{Index: index, ClientID: op.ClientID}
	switch op.Op {
	case "place":
		if op.Item == nil || op.Item.ID == uuid.Nil || op.Item.Type != "node" && op.Item.Type != "area" && op.Item.Type != "note" {
			return result, apperr.Invalid(apperr.CodeValidation, "Invalid item to place.")
		}
		if visible != nil && !visible[op.Item.Type][op.Item.ID] {
			return result, apperr.ErrNotFound()
		}
		exists, err := q.CanvasItemExists(ctx, dbgen.CanvasItemExistsParams{WorkspaceID: wsID, ID: op.Item.ID, Kind: op.Item.Type})
		if err != nil {
			return result, err
		}
		if !exists {
			return result, apperr.ErrNotFound()
		}
		style, err := marshalObject(op.Style)
		if err != nil {
			return result, err
		}
		args := dbgen.InsertCanvasPlacementParams{
			WorkspaceID: wsID, CanvasID: canvasID, ParentPlacementID: op.ParentPlacementID,
			X: number(op.X), Y: number(op.Y), Width: op.Width, Height: op.Height, ZIndex: integer(op.ZIndex), Collapsed: boolean(op.Collapsed), Style: style,
		}
		switch op.Item.Type {
		case "node":
			args.NodeID = &op.Item.ID
		case "area":
			args.AreaID = &op.Item.ID
		case "note":
			args.NoteID = &op.Item.ID
		}
		id, err := q.InsertCanvasPlacement(ctx, args)
		if err != nil {
			return result, err
		}
		result.ID = &id
		if op.Item.Type == "area" {
			if err := q.AdoptAreaNodes(ctx, dbgen.AdoptAreaNodesParams{WorkspaceID: wsID, CanvasID: canvasID, AreaID: &op.Item.ID, AreaPlacementID: &id}); err != nil {
				return result, err
			}
		}
		return result, nil
	case "update":
		if op.PlacementID == nil {
			return result, apperr.Invalid(apperr.CodeValidation, "placementId is required.")
		}
		p, err := q.GetCanvasPlacement(ctx, dbgen.GetCanvasPlacementParams{WorkspaceID: wsID, CanvasID: canvasID, ID: *op.PlacementID})
		if pg.IsNotFound(err) {
			return result, apperr.ErrNotFound()
		}
		if err != nil {
			return result, err
		}
		if p.LinkID != nil {
			return result, apperr.Invalid(apperr.CodeValidation, "Use link.override to edit a link.")
		}
		if op.parentSet {
			p.ParentPlacementID = op.ParentPlacementID
			if p.NodeID != nil {
				var areaID *uuid.UUID
				if p.ParentPlacementID != nil {
					areaID, err = q.GetCanvasParentArea(ctx, dbgen.GetCanvasParentAreaParams{WorkspaceID: wsID, CanvasID: canvasID, ID: *p.ParentPlacementID})
					if pg.IsNotFound(err) {
						return result, apperr.Invalid(apperr.CodeValidation, "Parent must be an area placement.")
					}
					if err != nil {
						return result, err
					}
				}
				if err := q.ReparentCanvasNode(ctx, dbgen.ReparentCanvasNodeParams{WorkspaceID: wsID, NodeID: *p.NodeID, AreaID: areaID}); err != nil {
					return result, err
				}
			}
		}
		if op.X != nil {
			p.X = *op.X
		}
		if op.Y != nil {
			p.Y = *op.Y
		}
		if op.widthSet {
			p.Width = op.Width
		}
		if op.heightSet {
			p.Height = op.Height
		}
		if op.ZIndex != nil {
			p.ZIndex = *op.ZIndex
		}
		if op.Collapsed != nil {
			p.Collapsed = *op.Collapsed
		}
		if op.Style != nil {
			p.Style, err = marshalObject(op.Style)
			if err != nil {
				return result, err
			}
		}
		_, err = q.UpdateCanvasPlacement(ctx, dbgen.UpdateCanvasPlacementParams{
			WorkspaceID: wsID, CanvasID: canvasID, ID: p.ID,
			ParentPlacementID: p.ParentPlacementID, X: p.X, Y: p.Y, Width: p.Width, Height: p.Height, ZIndex: p.ZIndex, Collapsed: p.Collapsed, Style: p.Style,
		})
		if err != nil {
			return result, err
		}
		result.ID = &p.ID
		return result, nil
	case "remove":
		if op.PlacementID == nil {
			return result, apperr.Invalid(apperr.CodeValidation, "placementId is required.")
		}
		count, err := q.DeleteCanvasPlacement(ctx, dbgen.DeleteCanvasPlacementParams{WorkspaceID: wsID, CanvasID: canvasID, ID: *op.PlacementID})
		if err != nil {
			return result, err
		}
		if count == 0 {
			return result, apperr.ErrNotFound()
		}
		removed, err := q.DeleteOrphanLinkOverrides(ctx, dbgen.DeleteOrphanLinkOverridesParams{WorkspaceID: wsID, CanvasID: canvasID})
		if err != nil {
			return result, err
		}
		result.RemovedPlacementIDs = removed
		return result, nil
	case "shape.create":
		return s.createShape(ctx, q, actor, canvasID, result, op)
	case "shape.update":
		return s.updateShape(ctx, q, actor, canvasID, result, op)
	case "shape.delete":
		if op.ShapeID == nil {
			return result, apperr.Invalid(apperr.CodeValidation, "shapeId is required.")
		}
		count, err := q.DeleteCanvasShape(ctx, dbgen.DeleteCanvasShapeParams{WorkspaceID: wsID, CanvasID: canvasID, ID: *op.ShapeID})
		if err != nil {
			return result, err
		}
		if count == 0 {
			return result, apperr.ErrNotFound()
		}
		return result, nil
	case "link.override":
		return s.overrideLink(ctx, q, actor, canvasID, result, op)
	case "link.reset":
		if op.LinkID == nil {
			return result, apperr.Invalid(apperr.CodeValidation, "linkId is required.")
		}
		_, err := q.DeleteCanvasLinkOverride(ctx, dbgen.DeleteCanvasLinkOverrideParams{WorkspaceID: wsID, CanvasID: canvasID, LinkID: op.LinkID})
		return result, err
	default:
		return result, apperr.Invalid(apperr.CodeValidation, "Unknown canvas operation.")
	}
}

func (s *Service) createShape(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID, result CanvasOperationResultItem, op CanvasOperation) (CanvasOperationResultItem, error) {
	if op.Kind == nil {
		return result, apperr.Invalid(apperr.CodeValidation, "Shape kind is required.")
	}
	style, err := marshalObject(op.Style)
	if err != nil {
		return result, err
	}
	data, err := marshalObject(op.Data)
	if err != nil {
		return result, err
	}
	id, err := q.InsertCanvasShape(ctx, dbgen.InsertCanvasShapeParams{
		WorkspaceID: actor.WorkspaceID, CanvasID: canvasID,
		Kind: *op.Kind, X: number(op.X), Y: number(op.Y), Width: op.Width, Height: op.Height, Rotation: number(op.Rotation),
		ZIndex: integer(op.ZIndex), Style: style, Data: data, CreatedBy: actorID(actor),
	})
	if err != nil {
		return result, err
	}
	result.ID = &id
	return result, nil
}

func (s *Service) updateShape(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID, result CanvasOperationResultItem, op CanvasOperation) (CanvasOperationResultItem, error) {
	if op.ShapeID == nil {
		return result, apperr.Invalid(apperr.CodeValidation, "shapeId is required.")
	}
	p, err := q.GetCanvasShape(ctx, dbgen.GetCanvasShapeParams{WorkspaceID: actor.WorkspaceID, CanvasID: canvasID, ID: *op.ShapeID})
	if pg.IsNotFound(err) {
		return result, apperr.ErrNotFound()
	}
	if err != nil {
		return result, err
	}
	if op.X != nil {
		p.X = *op.X
	}
	if op.Y != nil {
		p.Y = *op.Y
	}
	if op.widthSet {
		p.Width = op.Width
	}
	if op.heightSet {
		p.Height = op.Height
	}
	if op.Rotation != nil {
		p.Rotation = *op.Rotation
	}
	if op.ZIndex != nil {
		p.ZIndex = *op.ZIndex
	}
	if op.Style != nil {
		p.Style, err = marshalObject(op.Style)
		if err != nil {
			return result, err
		}
	}
	if op.Data != nil {
		p.Data, err = marshalObject(op.Data)
		if err != nil {
			return result, err
		}
	}
	_, err = q.UpdateCanvasShape(ctx, dbgen.UpdateCanvasShapeParams{
		WorkspaceID: actor.WorkspaceID, CanvasID: canvasID, ID: p.ID,
		X: p.X, Y: p.Y, Width: p.Width, Height: p.Height, Rotation: p.Rotation, ZIndex: p.ZIndex, Style: p.Style, Data: p.Data,
	})
	if err != nil {
		return result, err
	}
	result.ID = &p.ID
	return result, nil
}

func (s *Service) overrideLink(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID, result CanvasOperationResultItem, op CanvasOperation) (CanvasOperationResultItem, error) {
	if op.LinkID == nil {
		return result, apperr.Invalid(apperr.CodeValidation, "linkId is required.")
	}
	wsID := actor.WorkspaceID
	endpoints, err := q.CanvasLinkEndpoints(ctx, dbgen.CanvasLinkEndpointsParams{WorkspaceID: wsID, ID: *op.LinkID})
	if pg.IsNotFound(err) {
		return result, apperr.ErrNotFound()
	}
	if err != nil {
		return result, err
	}
	for _, nodeID := range []uuid.UUID{endpoints.SourceNodeID, endpoints.TargetNodeID} {
		placed, err := q.CanvasHasNode(ctx, dbgen.CanvasHasNodeParams{WorkspaceID: wsID, CanvasID: canvasID, NodeID: &nodeID})
		if err != nil {
			return result, err
		}
		if !placed {
			return result, apperr.Invalid(apperr.CodeValidation, "Both link endpoints must be on this canvas.")
		}
	}
	hidden := boolean(op.Hidden)
	style := json.RawMessage(`{}`)
	var waypoints json.RawMessage
	current, err := q.GetCanvasLinkOverride(ctx, dbgen.GetCanvasLinkOverrideParams{WorkspaceID: wsID, CanvasID: canvasID, LinkID: op.LinkID})
	if err == nil {
		hidden = current.Hidden
		style = current.Style
		waypoints = current.Waypoints
	} else if !pg.IsNotFound(err) {
		return result, err
	}
	if op.Hidden != nil {
		hidden = *op.Hidden
	}
	if op.Style != nil {
		style, err = marshalObject(op.Style)
		if err != nil {
			return result, err
		}
	}
	if op.waypointsSet {
		waypoints = nil
		if op.Waypoints != nil {
			waypoints, err = json.Marshal(op.Waypoints)
			if err != nil {
				return result, err
			}
		}
	}
	if err := q.UpsertCanvasLinkOverride(ctx, dbgen.UpsertCanvasLinkOverrideParams{
		WorkspaceID: wsID, CanvasID: canvasID,
		LinkID: op.LinkID, Hidden: hidden, Style: style, Waypoints: waypoints,
	}); err != nil {
		return result, err
	}
	return result, nil
}
