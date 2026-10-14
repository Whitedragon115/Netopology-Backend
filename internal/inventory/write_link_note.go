package inventory

import (
	"context"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func (s *Service) CreateLink(ctx context.Context, wsID uuid.UUID, req LinkCreate) (Link, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.LinkCreate)
	if err != nil {
		return Link{}, err
	}
	if req.SourceNodeID == req.TargetNodeID {
		return Link{}, apperr.Invalid(apperr.CodeValidation, "A link needs two different nodes.")
	}
	properties, err := checkedFields(req.Properties, nil)
	if err != nil {
		return Link{}, err
	}
	var id uuid.UUID
	var out Link
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "node", req.SourceNodeID); err != nil {
			return err
		}
		if err := requireVisible(vis, "node", req.TargetNodeID); err != nil {
			return err
		}
		for _, nodeID := range []uuid.UUID{req.SourceNodeID, req.TargetNodeID} {
			exists, err := q.NodeExists(ctx, dbgen.NodeExistsParams{WorkspaceID: wsID, ID: nodeID})
			if err != nil {
				return err
			}
			if !exists {
				return apperr.ErrNotFound()
			}
		}
		row, err := q.InsertLink(ctx, dbgen.InsertLinkParams{
			WorkspaceID: wsID, SourceNodeID: req.SourceNodeID,
			TargetNodeID: req.TargetNodeID, Kind: req.Kind, Directed: req.Directed, Label: req.Label, SourcePort: req.SourcePort,
			TargetPort: req.TargetPort, Properties: properties, CreatedBy: auditActor(actor),
		})
		if err != nil {
			return err
		}
		id = row.ID
		if err := recordInventoryChange(ctx, q, actor, authz.LinkCreate, "link", events.LinkCreated, id, row.Label, nil); err != nil {
			return err
		}
		out, err = readLink(ctx, q, actor, id)
		return err
	})
	if err != nil {
		return Link{}, inventoryWriteError(err)
	}
	return out, nil
}

func (s *Service) UpdateLink(ctx context.Context, wsID, id uuid.UUID, req LinkUpdate, match *int32) (Link, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.LinkUpdate)
	if err != nil {
		return Link{}, err
	}
	if !req.hasChange() {
		return Link{}, apperr.Invalid(apperr.CodeValidation, "At least one field is required.")
	}
	var out Link
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "link", id); err != nil {
			return err
		}
		current, err := q.LockLink(ctx, dbgen.LockLinkParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		if req.Kind != nil {
			current.Kind = *req.Kind
		}
		if req.Directed != nil {
			current.Directed = *req.Directed
		}
		if req.Label != nil {
			current.Label = *req.Label
		}
		if req.sourcePortSet {
			current.SourcePort = req.SourcePort
		}
		if req.targetPortSet {
			current.TargetPort = req.TargetPort
		}
		if req.Properties != nil {
			current.Properties, err = checkedFields(req.Properties, current.Properties)
			if err != nil {
				return err
			}
		}
		_, err = q.UpdateLinkInventory(ctx, dbgen.UpdateLinkInventoryParams{
			WorkspaceID: wsID, ID: id, Version: current.Version,
			Kind: current.Kind, Directed: current.Directed, Label: current.Label, SourcePort: current.SourcePort,
			TargetPort: current.TargetPort, Properties: current.Properties,
		})
		if err != nil {
			return err
		}
		if err := recordInventoryChange(ctx, q, actor, authz.LinkUpdate, "link", events.LinkUpdated, id, current.Label, nil); err != nil {
			return err
		}
		out, err = readLink(ctx, q, actor, id)
		return err
	})
	if err != nil {
		return Link{}, inventoryWriteError(err)
	}
	return out, nil
}

func (s *Service) DeleteLink(ctx context.Context, wsID, id uuid.UUID, match *int32) error {
	actor, err := s.authz.Actor(ctx, wsID, authz.LinkDelete)
	if err != nil {
		return err
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "link", id); err != nil {
			return err
		}
		current, err := q.LockLink(ctx, dbgen.LockLinkParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		canvases, err := q.InventoryCanvasIDs(ctx, dbgen.InventoryCanvasIDsParams{WorkspaceID: wsID, Kind: "link", ID: &id})
		if err != nil {
			return err
		}
		count, err := q.DeleteLinkInventory(ctx, dbgen.DeleteLinkInventoryParams{WorkspaceID: wsID, ID: id, Version: current.Version})
		if err != nil {
			return err
		}
		if count != 1 {
			return apperr.PreconditionFailed()
		}
		return recordInventoryChange(ctx, q, actor, authz.LinkDelete, "link", events.LinkDeleted, id, current.Label, canvases)
	})
	return inventoryWriteError(err)
}

func noteTargetIDs(target NoteTarget) (node, link, area, canvas *uuid.UUID, error error) {
	switch target.Type {
	case "workspace":
		if target.ID != nil {
			return nil, nil, nil, nil, apperr.Invalid(apperr.CodeValidation, "Workspace-level notes have no target id.")
		}
	case "node":
		node = target.ID
	case "link":
		link = target.ID
	case "area":
		area = target.ID
	case "canvas":
		canvas = target.ID
	default:
		return nil, nil, nil, nil, apperr.Invalid(apperr.CodeValidation, "Invalid note target.")
	}
	if target.Type != "workspace" && target.ID == nil {
		return nil, nil, nil, nil, apperr.Invalid(apperr.CodeValidation, "Note target id is required.")
	}
	return
}

func (s *Service) CreateNote(ctx context.Context, wsID uuid.UUID, req NoteCreate) (Note, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NoteCreate)
	if err != nil {
		return Note{}, err
	}
	node, link, area, canvas, err := noteTargetIDs(req.Target)
	if err != nil {
		return Note{}, err
	}
	if req.Target.Type == "workspace" && actor.Role == authz.RoleEditor {
		return Note{}, apperr.Forbidden(apperr.CodeRoleInsufficient, "Only owners and admins can create workspace notes.")
	}
	var id uuid.UUID
	var out Note
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if req.Target.ID != nil {
			if err := requireVisible(vis, req.Target.Type, *req.Target.ID); err != nil {
				return err
			}
		}
		if err := s.preparePlaceOn(ctx, q, actor, req.PlaceOn); err != nil {
			return err
		}
		row, err := q.InsertNote(ctx, dbgen.InsertNoteParams{
			WorkspaceID: wsID, NodeID: node, LinkID: link, AreaID: area,
			CanvasID: canvas, Kind: choose(req.Kind, "note"), Title: req.Title, Body: req.Body, IsSensitive: req.IsSensitive,
			Color: choose(req.Color, "default"), CreatedBy: auditActor(actor), UpdatedBy: auditActor(actor),
		})
		if err != nil {
			return err
		}
		id = row.ID
		if err := insertPlaceOn(ctx, q, wsID, req.PlaceOn, "note", id); err != nil {
			return err
		}
		if err := recordInventoryChange(ctx, q, actor, authz.NoteCreate, "note", events.NoteCreated, id, row.Title, nil); err != nil {
			return err
		}
		out, err = readNote(ctx, q, actor, id)
		return err
	})
	if err != nil {
		return Note{}, inventoryWriteError(err)
	}
	return out, nil
}

func (s *Service) UpdateNote(ctx context.Context, wsID, id uuid.UUID, req NoteUpdate, match *int32) (Note, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NoteUpdate)
	if err != nil {
		return Note{}, err
	}
	if !req.hasChange() {
		return Note{}, apperr.Invalid(apperr.CodeValidation, "At least one field is required.")
	}
	var out Note
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "note", id); err != nil {
			return err
		}
		current, err := q.LockNote(ctx, dbgen.LockNoteParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if current.NodeID == nil && current.LinkID == nil && current.AreaID == nil && current.CanvasID == nil && actor.Role == authz.RoleEditor {
			return apperr.Forbidden(apperr.CodeRoleInsufficient, "Only owners and admins can edit workspace notes.")
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		if req.Kind != nil {
			current.Kind = *req.Kind
		}
		if req.Title != nil {
			current.Title = *req.Title
		}
		if req.Body != nil {
			current.Body = *req.Body
		}
		if req.IsSensitive != nil {
			current.IsSensitive = *req.IsSensitive
		}
		if req.Color != nil {
			current.Color = *req.Color
		}
		_, err = q.UpdateNoteInventory(ctx, dbgen.UpdateNoteInventoryParams{
			WorkspaceID: wsID, ID: id, Version: current.Version,
			Kind: current.Kind, Title: current.Title, Body: current.Body, IsSensitive: current.IsSensitive, Color: current.Color, UpdatedBy: auditActor(actor),
		})
		if err != nil {
			return err
		}
		if err := recordInventoryChange(ctx, q, actor, authz.NoteUpdate, "note", events.NoteUpdated, id, current.Title, nil); err != nil {
			return err
		}
		out, err = readNote(ctx, q, actor, id)
		return err
	})
	if err != nil {
		return Note{}, inventoryWriteError(err)
	}
	return out, nil
}

func (s *Service) DeleteNote(ctx context.Context, wsID, id uuid.UUID, match *int32) error {
	actor, err := s.authz.Actor(ctx, wsID, authz.NoteDelete)
	if err != nil {
		return err
	}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if err := requireVisible(vis, "note", id); err != nil {
			return err
		}
		current, err := q.LockNote(ctx, dbgen.LockNoteParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if current.NodeID == nil && current.LinkID == nil && current.AreaID == nil && current.CanvasID == nil && actor.Role == authz.RoleEditor {
			return apperr.Forbidden(apperr.CodeRoleInsufficient, "Only owners and admins can delete workspace notes.")
		}
		if err := checkMatch(current.Version, match); err != nil {
			return err
		}
		if err := usedElsewhere(ctx, q, actor, "note", id); err != nil {
			return err
		}
		canvases, err := q.InventoryCanvasIDs(ctx, dbgen.InventoryCanvasIDsParams{WorkspaceID: wsID, Kind: "note", ID: &id})
		if err != nil {
			return err
		}
		count, err := q.DeleteNoteInventory(ctx, dbgen.DeleteNoteInventoryParams{WorkspaceID: wsID, ID: id, Version: current.Version})
		if err != nil {
			return err
		}
		if count != 1 {
			return apperr.PreconditionFailed()
		}
		return recordInventoryChange(ctx, q, actor, authz.NoteDelete, "note", events.NoteDeleted, id, current.Title, canvases)
	})
	return inventoryWriteError(err)
}
