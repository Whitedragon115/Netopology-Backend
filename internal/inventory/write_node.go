package inventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/slug"
)

func generatedNodeSlug(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, name string) (string, error) {
	base := slug.Make(name)
	for n := 1; n <= 1000; n++ {
		candidate := base
		if n > 1 {
			suffix := fmt.Sprintf("-%d", n)
			candidate = strings.TrimRight(base[:min(len(base), 63-len(suffix))], "-") + suffix
		}
		used, err := q.NodeSlugExists(ctx, dbgen.NodeSlugExistsParams{WorkspaceID: wsID, Slug: candidate})
		if err != nil {
			return "", err
		}
		if !used {
			return candidate, nil
		}
	}
	return "", apperr.Conflict(apperr.CodeSlugTaken, "No available slug for that node name.")
}

func (s *Service) CreateNode(ctx context.Context, wsID uuid.UUID, req NodeCreate) (Node, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NodeCreate)
	if err != nil {
		return Node{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		return Node{}, apperr.Invalid(apperr.CodeValidation, "Invalid node name.")
	}
	if req.IsProtected && actor.Role == authz.RoleEditor {
		return Node{}, apperr.Forbidden(apperr.CodeProtected, "Editors cannot protect nodes.")
	}
	properties, err := checkedFields(req.Properties, nil)
	if err != nil {
		return Node{}, err
	}
	var id uuid.UUID
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if req.AreaID != nil {
			if err := requireVisible(vis, "area", *req.AreaID); err != nil {
				return err
			}
			exists, err := q.AreaExists(ctx, dbgen.AreaExistsParams{WorkspaceID: wsID, ID: *req.AreaID})
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
		nodeSlug := ""
		if req.Slug != nil {
			nodeSlug = *req.Slug
		} else {
			nodeSlug, err = generatedNodeSlug(ctx, q, wsID, name)
			if err != nil {
				return err
			}
		}
		row, err := q.InsertNode(ctx, dbgen.InsertNodeParams{
			WorkspaceID: wsID, AreaID: req.AreaID, Slug: nodeSlug, Name: name,
			Category: req.Category, Type: choose(req.Type, "builtin.generic"), Icon: req.Icon, Description: req.Description,
			Properties: properties, IsProtected: req.IsProtected, CreatedBy: auditActor(actor),
		})
		if err != nil {
			return err
		}
		id = row.ID
		if err := insertPlaceOn(ctx, q, wsID, req.PlaceOn, "node", id); err != nil {
			return err
		}
		return recordInventoryChange(ctx, q, actor, authz.NodeCreate, "node", events.NodeCreated, id, nil)
	})
	if err != nil {
		return Node{}, inventoryWriteError(err)
	}
	return s.GetNode(ctx, wsID, id)
}

func (s *Service) UpdateNode(ctx context.Context, wsID, id uuid.UUID, req NodeUpdate, match *int32) (Node, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NodeUpdate)
	if err != nil {
		return Node{}, err
	}
	if !req.hasChange() {
		return Node{}, apperr.Invalid(apperr.CodeValidation, "At least one field is required.")
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "node", id); err != nil {
			return err
		}
		current, err := q.LockNode(ctx, dbgen.LockNodeParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		if req.IsProtected != nil {
			if actor.Role == authz.RoleEditor || actor.Role == authz.RoleAdmin && !*req.IsProtected && current.IsProtected {
				return apperr.Forbidden(apperr.CodeProtected, "Only the owner can remove node protection.")
			}
			current.IsProtected = *req.IsProtected
		}
		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.Slug != nil {
			current.Slug = *req.Slug
		}
		if req.Category != nil {
			current.Category = *req.Category
		}
		if req.Type != nil {
			current.Type = *req.Type
		}
		if req.iconSet {
			current.Icon = req.Icon
		}
		if req.Description != nil {
			current.Description = *req.Description
		}
		if req.areaSet {
			if req.AreaID != nil {
				if err := requireVisible(vis, "area", *req.AreaID); err != nil {
					return err
				}
			}
			current.AreaID = req.AreaID
		}
		if req.Properties != nil {
			current.Properties, err = checkedFields(req.Properties, current.Properties)
			if err != nil {
				return err
			}
		}
		if current.Name == "" || len(current.Name) > 100 {
			return apperr.Invalid(apperr.CodeValidation, "Invalid node name.")
		}
		_, err = q.UpdateNodeInventory(ctx, dbgen.UpdateNodeInventoryParams{
			WorkspaceID: wsID, ID: id, Version: current.Version,
			AreaID: current.AreaID, Slug: current.Slug, Name: current.Name, Category: current.Category, Type: current.Type,
			Icon: current.Icon, Description: current.Description, Properties: current.Properties, IsProtected: current.IsProtected,
		})
		if err != nil {
			return err
		}
		return recordInventoryChange(ctx, q, actor, authz.NodeUpdate, "node", events.NodeUpdated, id, nil)
	})
	if err != nil {
		return Node{}, inventoryWriteError(err)
	}
	return s.GetNode(ctx, wsID, id)
}

func (s *Service) DeleteNode(ctx context.Context, wsID, id uuid.UUID, match *int32) error {
	actor, err := s.authz.Actor(ctx, wsID, authz.NodeDelete)
	if err != nil {
		return err
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "node", id); err != nil {
			return err
		}
		current, err := q.LockNode(ctx, dbgen.LockNodeParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		if current.IsProtected && actor.Role != authz.RoleOwner {
			return apperr.Forbidden(apperr.CodeProtected, "Only the owner can delete a protected node.")
		}
		if err := usedElsewhere(ctx, q, actor, "node", id); err != nil {
			return err
		}
		canvases, err := q.InventoryCanvasIDs(ctx, dbgen.InventoryCanvasIDsParams{WorkspaceID: wsID, Kind: "node", ID: &id})
		if err != nil {
			return err
		}
		count, err := q.DeleteNodeInventory(ctx, dbgen.DeleteNodeInventoryParams{WorkspaceID: wsID, ID: id, Version: current.Version})
		if err != nil {
			return err
		}
		if count != 1 {
			return apperr.PreconditionFailed()
		}
		return recordInventoryChange(ctx, q, actor, authz.NodeDelete, "node", events.NodeDeleted, id, canvases)
	})
	return inventoryWriteError(err)
}

func (s *Service) SetNodeHealth(ctx context.Context, wsID, id uuid.UUID, req HealthSet) (Node, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NodeHealthSet)
	if err != nil {
		return Node{}, err
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "node", id); err != nil {
			return err
		}
		_, err = q.SetNodeHealthInventory(ctx, dbgen.SetNodeHealthInventoryParams{WorkspaceID: wsID, ID: id, Status: req.Status, Message: req.Message})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		return recordInventoryChange(ctx, q, actor, authz.NodeHealthSet, "node", events.HealthChanged, id, nil)
	})
	if err != nil {
		return Node{}, inventoryWriteError(err)
	}
	return s.GetNode(ctx, wsID, id)
}
