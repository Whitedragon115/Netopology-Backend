package inventory

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func (s *Service) CreateArea(ctx context.Context, wsID uuid.UUID, req AreaCreate) (Area, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.AreaCreate)
	if err != nil {
		return Area{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		return Area{}, apperr.Invalid(apperr.CodeValidation, "Invalid area name.")
	}
	properties, err := checkedFields(req.Properties, nil)
	if err != nil {
		return Area{}, err
	}
	cidr, err := parsedCIDR(req.CIDR)
	if err != nil {
		return Area{}, err
	}
	var id uuid.UUID
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if req.ParentID != nil {
			if err := requireVisible(vis, "area", *req.ParentID); err != nil {
				return err
			}
			exists, err := q.AreaExists(ctx, dbgen.AreaExistsParams{WorkspaceID: wsID, ID: *req.ParentID})
			if err != nil {
				return err
			}
			if !exists {
				return apperr.ErrNotFound()
			}
		}
		if err := s.preparePlaceOn(ctx, q, actor, req.PlaceOn); err != nil {
			return err
		}
		row, err := q.InsertArea(ctx, dbgen.InsertAreaParams{
			WorkspaceID: wsID, ParentID: req.ParentID,
			Name: name, Kind: choose(req.Kind, "custom"), Color: choose(req.Color, "yellow"), Cidr: cidr,
			VlanID: req.VlanID, Description: req.Description, Properties: properties, CreatedBy: auditActor(actor),
		})
		if err != nil {
			return err
		}
		id = row.ID
		if err := insertPlaceOn(ctx, q, wsID, req.PlaceOn, "area", id); err != nil {
			return err
		}
		return recordInventoryChange(ctx, q, actor, authz.AreaCreate, "area", events.AreaCreated, id, nil)
	})
	if err != nil {
		return Area{}, inventoryWriteError(err)
	}
	return s.GetArea(ctx, wsID, id)
}

func (s *Service) UpdateArea(ctx context.Context, wsID, id uuid.UUID, req AreaUpdate, match *int32) (Area, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.AreaUpdate)
	if err != nil {
		return Area{}, err
	}
	if !req.hasChange() {
		return Area{}, apperr.Invalid(apperr.CodeValidation, "At least one field is required.")
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "area", id); err != nil {
			return err
		}
		current, err := q.LockArea(ctx, dbgen.LockAreaParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.Kind != nil {
			current.Kind = *req.Kind
		}
		if req.Color != nil {
			current.Color = *req.Color
		}
		if req.cidrSet {
			current.Cidr, err = parsedCIDR(req.CIDR)
			if err != nil {
				return err
			}
		}
		if req.vlanSet {
			current.VlanID = req.VlanID
		}
		if req.Description != nil {
			current.Description = *req.Description
		}
		if req.Properties != nil {
			current.Properties, err = checkedFields(req.Properties, current.Properties)
			if err != nil {
				return err
			}
		}
		if req.parentSet {
			if req.ParentID != nil {
				if err := requireVisible(vis, "area", *req.ParentID); err != nil {
					return err
				}
			}
			current.ParentID = req.ParentID
		}
		if current.Name == "" || len(current.Name) > 100 {
			return apperr.Invalid(apperr.CodeValidation, "Invalid area name.")
		}
		_, err = q.UpdateAreaInventory(ctx, dbgen.UpdateAreaInventoryParams{
			WorkspaceID: wsID, ID: id, Version: current.Version,
			ParentID: current.ParentID, Name: current.Name, Kind: current.Kind, Color: current.Color, Cidr: current.Cidr,
			VlanID: current.VlanID, Description: current.Description, Properties: current.Properties,
		})
		if err != nil {
			return err
		}
		return recordInventoryChange(ctx, q, actor, authz.AreaUpdate, "area", events.AreaUpdated, id, nil)
	})
	if err != nil {
		return Area{}, inventoryWriteError(err)
	}
	return s.GetArea(ctx, wsID, id)
}

func (s *Service) DeleteArea(ctx context.Context, wsID, id uuid.UUID, match *int32) error {
	actor, err := s.authz.Actor(ctx, wsID, authz.AreaDelete)
	if err != nil {
		return err
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "area", id); err != nil {
			return err
		}
		current, err := q.LockArea(ctx, dbgen.LockAreaParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		if err := usedElsewhere(ctx, q, actor, "area", id); err != nil {
			return err
		}
		canvases, err := q.InventoryCanvasIDs(ctx, dbgen.InventoryCanvasIDsParams{WorkspaceID: wsID, Kind: "area", ID: &id})
		if err != nil {
			return err
		}
		count, err := q.DeleteAreaInventory(ctx, dbgen.DeleteAreaInventoryParams{WorkspaceID: wsID, ID: id, Version: current.Version})
		if err != nil {
			return err
		}
		if count != 1 {
			return apperr.PreconditionFailed()
		}
		return recordInventoryChange(ctx, q, actor, authz.AreaDelete, "area", events.AreaDeleted, id, canvases)
	})
	return inventoryWriteError(err)
}

func auditActor(actor *authz.Actor) *uuid.UUID { id := actor.UserID(); return &id }
