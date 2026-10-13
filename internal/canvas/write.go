package canvas

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func canvasChange(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, action authz.Permission, event events.Type, id uuid.UUID) error {
	if err := auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &actor.WorkspaceID, Action: action, TargetType: "canvas", TargetID: &id}); err != nil {
		return err
	}
	return events.Emit(ctx, q, events.Change{WorkspaceID: actor.WorkspaceID, Type: event, SubjectID: &id, CanvasIDs: []uuid.UUID{id}})
}

func canvasError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := pg.UniqueViolation(err); ok {
		return apperr.Conflict("canvas.name_taken", "That canvas name is already used.")
	}
	return err
}

func (s *Service) CanvasCreate(ctx context.Context, wsID uuid.UUID, req CanvasCreate) (Canvas, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.CanvasCreate)
	if err != nil {
		return Canvas{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		return Canvas{}, apperr.Invalid(apperr.CodeValidation, "Invalid canvas name.")
	}
	settings := json.RawMessage(`{}`)
	if req.CanvasSettings != nil {
		settings, err = json.Marshal(req.CanvasSettings)
		if err != nil {
			return Canvas{}, err
		}
	}
	var out Canvas
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.InsertTabCanvas(ctx, dbgen.InsertTabCanvasParams{
			WorkspaceID: wsID, Name: name,
			Description: req.Description, SortOrder: req.SortOrder, Settings: settings, IsProtected: req.IsProtected, CreatedBy: actorID(actor),
		})
		if err != nil {
			return err
		}
		out, err = fromRow(row)
		if err != nil {
			return err
		}
		return canvasChange(ctx, q, actor, authz.CanvasCreate, events.CanvasCreated, row.ID)
	})
	return out, canvasError(err)
}

func (s *Service) CanvasUpdate(ctx context.Context, wsID, id uuid.UUID, req CanvasUpdate, match *int32) (Canvas, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.CanvasUpdate)
	if err != nil {
		return Canvas{}, err
	}
	if req.Name == nil && req.Description == nil && req.SortOrder == nil && req.CanvasViewport == nil && req.CanvasSettings == nil && req.IsProtected == nil {
		return Canvas{}, apperr.Invalid(apperr.CodeValidation, "At least one field is required.")
	}
	var out Canvas
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.LockCanvasRow(ctx, dbgen.LockCanvasRowParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if match != nil && row.Version != *match {
			return apperr.PreconditionFailed()
		}
		if req.IsProtected != nil && row.IsProtected && !*req.IsProtected && actor.Role != authz.RoleOwner {
			return apperr.Forbidden(apperr.CodeProtected, "Only the owner can remove canvas protection.")
		}
		if req.Name != nil {
			row.Name = strings.TrimSpace(*req.Name)
		}
		if row.Name == "" || len(row.Name) > 100 {
			return apperr.Invalid(apperr.CodeValidation, "Invalid canvas name.")
		}
		if req.Description != nil {
			row.Description = *req.Description
		}
		if req.SortOrder != nil {
			row.SortOrder = *req.SortOrder
		}
		if req.CanvasViewport != nil {
			if req.CanvasViewport.Zoom <= 0 || req.CanvasViewport.Zoom > 8 {
				return apperr.Invalid(apperr.CodeValidation, "Invalid viewport zoom.")
			}
			row.Viewport, err = json.Marshal(req.CanvasViewport)
			if err != nil {
				return err
			}
		}
		if req.CanvasSettings != nil {
			row.Settings, err = json.Marshal(req.CanvasSettings)
			if err != nil {
				return err
			}
		}
		if req.IsProtected != nil {
			row.IsProtected = *req.IsProtected
		}
		updated, err := q.UpdateCanvasRow(ctx, dbgen.UpdateCanvasRowParams{
			WorkspaceID: wsID, ID: id, Version: row.Version,
			Name: row.Name, Description: row.Description, SortOrder: row.SortOrder, Viewport: row.Viewport,
			Settings: row.Settings, IsProtected: row.IsProtected,
		})
		if err != nil {
			return err
		}
		out, err = fromRow(updated)
		if err != nil {
			return err
		}
		return canvasChange(ctx, q, actor, authz.CanvasUpdate, events.CanvasUpdated, id)
	})
	return out, canvasError(err)
}

func (s *Service) Delete(ctx context.Context, wsID, id uuid.UUID, match *int32) error {
	actor, err := s.authz.Actor(ctx, wsID, authz.CanvasDelete)
	if err != nil {
		return err
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.LockCanvasRow(ctx, dbgen.LockCanvasRowParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if match != nil && row.Version != *match {
			return apperr.PreconditionFailed()
		}
		if row.IsProtected && actor.Role != authz.RoleOwner {
			return apperr.Forbidden(apperr.CodeProtected, "Only the owner can delete a protected canvas.")
		}
		count, err := q.DeleteCanvasRow(ctx, dbgen.DeleteCanvasRowParams{WorkspaceID: wsID, ID: id, Version: row.Version})
		if err != nil {
			return err
		}
		if count != 1 {
			return apperr.PreconditionFailed()
		}
		return canvasChange(ctx, q, actor, authz.CanvasDelete, events.CanvasDeleted, id)
	})
	return err
}

func (s *Service) GetAreaView(ctx context.Context, wsID, areaID uuid.UUID) (CanvasView, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.AreaCanvas)
	if err != nil {
		return CanvasView{}, err
	}
	var out CanvasView
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		if actor.Role == authz.RoleEditor {
			vis, err := q.VisibleInventoryIDs(ctx, dbgen.VisibleInventoryIDsParams{WorkspaceID: wsID, UserID: actor.UserID()})
			if err != nil {
				return err
			}
			found := false
			for _, id := range vis.AreaIds {
				if id == areaID {
					found = true
					break
				}
			}
			if !found {
				return apperr.ErrNotFound()
			}
		}
		created, err := q.EnsureAreaCanvas(ctx, dbgen.EnsureAreaCanvasParams{WorkspaceID: wsID, AreaID: areaID, CreatedBy: actor.UserID()})
		if err != nil {
			return err
		}
		row, err := q.GetAreaCanvasRow(ctx, dbgen.GetAreaCanvasRowParams{WorkspaceID: wsID, AreaID: &areaID})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if created > 0 {
			if err := q.SeedAreaCanvasNodes(ctx, dbgen.SeedAreaCanvasNodesParams{WorkspaceID: wsID, AreaID: &areaID, CanvasID: row.ID}); err != nil {
				return err
			}
			if err := q.SeedAreaCanvasAreas(ctx, dbgen.SeedAreaCanvasAreasParams{WorkspaceID: wsID, AreaID: &areaID, CanvasID: row.ID}); err != nil {
				return err
			}
			if err := canvasChange(ctx, q, actor, authz.AreaCanvas, events.CanvasCreated, row.ID); err != nil {
				return err
			}
		}
		out, err = s.view(ctx, q, actor, row)
		return err
	})
	return out, err
}
