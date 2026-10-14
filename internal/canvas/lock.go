package canvas

import (
	"context"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func (s *Service) SetLock(ctx context.Context, wsID, id uuid.UUID, locked bool) (Canvas, error) {
	permission := authz.CanvasLock
	if !locked {
		permission = authz.CanvasUnlock
	}
	actor, err := s.authz.Actor(ctx, wsID, permission)
	if err != nil {
		return Canvas{}, err
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
		if err := s.canSeeCanvas(ctx, q, actor, row); err != nil {
			return err
		}
		if locked == (row.LockedBy != nil) {
			out, err = fromRow(row)
			return err
		}
		var userID *uuid.UUID
		if locked {
			userID = actorID(actor)
		}
		row, err = q.SetCanvasLock(ctx, dbgen.SetCanvasLockParams{WorkspaceID: wsID, ID: id, LockedBy: userID})
		if err != nil {
			return err
		}
		out, err = fromRow(row)
		if err != nil {
			return err
		}
		return canvasChange(ctx, q, actor, permission, events.CanvasUpdated, id, row.Name)
	})
	return out, err
}
