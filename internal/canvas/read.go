package canvas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func fromRow(row dbgen.Canvase) (Canvas, error) {
	out := Canvas{
		ID: row.ID, AreaID: row.AreaID, Name: row.Name, Description: row.Description,
		SortOrder: row.SortOrder, IsProtected: row.IsProtected, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
	if row.AreaID == nil {
		out.Kind = "tab"
	} else {
		out.Kind = "area"
	}
	if err := json.Unmarshal(row.Viewport, &out.CanvasViewport); err != nil {
		return Canvas{}, fmt.Errorf("decode canvas viewport: %w", err)
	}
	if err := json.Unmarshal(row.Settings, &out.CanvasSettings); err != nil {
		return Canvas{}, fmt.Errorf("decode canvas settings: %w", err)
	}
	if row.LockedBy != nil && row.LockedAt != nil {
		out.CanvasLock = &CanvasLock{UserID: *row.LockedBy, LockedAt: *row.LockedAt}
	}
	return out, nil
}

func (s *Service) canSeeCanvas(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, row dbgen.Canvase) error {
	if actor.Role != authz.RoleEditor {
		return nil
	}
	ids, err := q.VisibleInventoryIDs(ctx, dbgen.VisibleInventoryIDsParams{WorkspaceID: actor.WorkspaceID, UserID: actor.UserID()})
	if err != nil {
		return err
	}
	for _, id := range ids.CanvasIds {
		if id == row.ID {
			return nil
		}
	}
	return apperr.ErrNotFound()
}

func (s *Service) List(ctx context.Context, wsID uuid.UUID) (CanvasPage, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.CanvasList)
	if err != nil {
		return CanvasPage{}, err
	}
	out := CanvasPage{Items: []Canvas{}}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		var editorID *uuid.UUID
		if actor.Role == authz.RoleEditor {
			editorID = actorID(actor)
		}
		rows, err := q.ListTabCanvases(ctx, dbgen.ListTabCanvasesParams{WorkspaceID: wsID, EditorID: editorID})
		if err != nil {
			return err
		}
		for _, row := range rows {
			item, err := fromRow(row)
			if err != nil {
				return err
			}
			out.Items = append(out.Items, item)
		}
		return nil
	})
	return out, err
}

func (s *Service) Get(ctx context.Context, wsID, id uuid.UUID) (Canvas, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.CanvasRead)
	if err != nil {
		return Canvas{}, err
	}
	var out Canvas
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.GetCanvasRow(ctx, dbgen.GetCanvasRowParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := s.canSeeCanvas(ctx, q, actor, row); err != nil {
			return err
		}
		out, err = fromRow(row)
		return err
	})
	return out, err
}

func (s *Service) GetView(ctx context.Context, wsID, id uuid.UUID) (CanvasView, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.CanvasView)
	if err != nil {
		return CanvasView{}, err
	}
	var out CanvasView
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.GetCanvasRow(ctx, dbgen.GetCanvasRowParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if err := s.canSeeCanvas(ctx, q, actor, row); err != nil {
			return err
		}
		out, err = s.view(ctx, q, actor, row)
		return err
	})
	return out, err
}

func (s *Service) view(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, row dbgen.Canvase) (CanvasView, error) {
	canvas, err := fromRow(row)
	if err != nil {
		return CanvasView{}, err
	}
	out := CanvasView{
		Canvas: canvas, Viewer: Viewer{
			Role: string(actor.Role), CanEdit: actor.Role != authz.RoleViewer && row.LockedBy == nil,
			CanViewSensitive: actor.SeesSensitive(),
		}, Breadcrumb: []BreadcrumbItem{}, Placements: []Placement{},
		LinkOverrides: []LinkOverride{}, Nodes: []apimodel.Node{}, Links: []apimodel.Link{}, Areas: []apimodel.Area{},
		Notes: []apimodel.Note{}, Shapes: []Shape{},
	}
	if row.AreaID != nil {
		chain, err := q.ListAreaBreadcrumb(ctx, dbgen.ListAreaBreadcrumbParams{WorkspaceID: actor.WorkspaceID, AreaID: *row.AreaID})
		if err != nil {
			return CanvasView{}, err
		}
		if len(chain) > 0 {
			tab, err := q.FindAreaStartingTab(ctx, dbgen.FindAreaStartingTabParams{WorkspaceID: actor.WorkspaceID, AreaID: &chain[0].ID})
			if err == nil {
				canSeeTab := true
				if actor.Role == authz.RoleEditor {
					canSeeTab = false
					visible, err := q.VisibleInventoryIDs(ctx, dbgen.VisibleInventoryIDsParams{WorkspaceID: actor.WorkspaceID, UserID: actor.UserID()})
					if err != nil {
						return CanvasView{}, err
					}
					for _, id := range visible.CanvasIds {
						if id == tab.ID {
							canSeeTab = true
							break
						}
					}
				}
				if canSeeTab {
					out.Breadcrumb = append(out.Breadcrumb, BreadcrumbItem{Name: tab.Name, CanvasID: tab.ID})
				}
			} else if !pg.IsNotFound(err) {
				return CanvasView{}, err
			}
		}
		for _, area := range chain {
			if area.CanvasID != nil {
				out.Breadcrumb = append(out.Breadcrumb, BreadcrumbItem{AreaID: &area.ID, Name: area.Name, CanvasID: *area.CanvasID})
			}
		}
	}
	rows, err := q.ListCanvasPlacements(ctx, dbgen.ListCanvasPlacementsParams{WorkspaceID: actor.WorkspaceID, CanvasID: row.ID})
	if err != nil {
		return CanvasView{}, err
	}
	ids := apimodel.IDs{}
	for _, p := range rows {
		if p.LinkID != nil {
			override, err := overrideFromRow(p)
			if err != nil {
				return CanvasView{}, err
			}
			if !override.Hidden {
				out.LinkOverrides = append(out.LinkOverrides, override)
			}
			continue
		}
		placement, err := placementFromRow(p)
		if err != nil {
			return CanvasView{}, err
		}
		out.Placements = append(out.Placements, placement)
		switch placement.Item.Type {
		case "node":
			ids.Nodes = append(ids.Nodes, placement.Item.ID)
		case "area":
			ids.Areas = append(ids.Areas, placement.Item.ID)
		case "note":
			ids.Notes = append(ids.Notes, placement.Item.ID)
		}
	}
	linkIDs, err := q.ListCanvasViewLinkIDs(ctx, dbgen.ListCanvasViewLinkIDsParams{CanvasID: row.ID, WorkspaceID: actor.WorkspaceID})
	if err != nil {
		return CanvasView{}, err
	}
	ids.Links = linkIDs
	if row.AreaID != nil {
		ids.Areas = append(ids.Areas, *row.AreaID)
	}
	snapshot, err := s.reader.ReadForActor(ctx, q, actor, ids)
	if err != nil {
		return CanvasView{}, err
	}
	// An area canvas draws boundary links with an unplaced outside endpoint.
	if row.AreaID != nil {
		placed := map[uuid.UUID]bool{}
		for _, n := range snapshot.Nodes {
			placed[n.ID] = true
		}
		extra := apimodel.IDs{}
		for _, link := range snapshot.Links {
			if !placed[link.SourceNodeID] {
				extra.Nodes = append(extra.Nodes, link.SourceNodeID)
			}
			if !placed[link.TargetNodeID] {
				extra.Nodes = append(extra.Nodes, link.TargetNodeID)
			}
		}
		if len(extra.Nodes) > 0 {
			outside, err := s.reader.ReadForActor(ctx, q, actor, extra)
			if err != nil {
				return CanvasView{}, err
			}
			snapshot.Nodes = append(snapshot.Nodes, outside.Nodes...)
		}
	}
	areaSeen := map[uuid.UUID]bool{}
	for _, area := range snapshot.Areas {
		areaSeen[area.ID] = true
	}
	extraAreas := apimodel.IDs{}
	for _, node := range snapshot.Nodes {
		if node.AreaID != nil && !areaSeen[*node.AreaID] {
			extraAreas.Areas = append(extraAreas.Areas, *node.AreaID)
			areaSeen[*node.AreaID] = true
		}
	}
	if len(extraAreas.Areas) > 0 {
		areas, err := s.reader.ReadForActor(ctx, q, actor, extraAreas)
		if err != nil {
			return CanvasView{}, err
		}
		snapshot.Areas = append(snapshot.Areas, areas.Areas...)
	}
	out.Nodes, out.Links, out.Areas, out.Notes = snapshot.Nodes, snapshot.Links, snapshot.Areas, snapshot.Notes
	shapes, err := q.ListCanvasShapes(ctx, dbgen.ListCanvasShapesParams{WorkspaceID: actor.WorkspaceID, CanvasID: row.ID})
	if err != nil {
		return CanvasView{}, err
	}
	for _, shape := range shapes {
		item, err := shapeFromRow(shape)
		if err != nil {
			return CanvasView{}, err
		}
		out.Shapes = append(out.Shapes, item)
	}
	return out, nil
}

func actorID(actor *authz.Actor) *uuid.UUID { id := actor.UserID(); return &id }

func placementFromRow(p dbgen.Placement) (Placement, error) {
	item := ItemRef{}
	switch {
	case p.NodeID != nil:
		item.Type = "node"
		item.ID = *p.NodeID
	case p.AreaID != nil:
		item.Type = "area"
		item.ID = *p.AreaID
	case p.NoteID != nil:
		item.Type = "note"
		item.ID = *p.NoteID
	}
	out := Placement{ID: p.ID, Item: item, ParentPlacementID: p.ParentPlacementID, X: p.X, Y: p.Y, Width: p.Width, Height: p.Height, ZIndex: p.ZIndex, Collapsed: p.Collapsed}
	if err := json.Unmarshal(p.Style, &out.Style); err != nil {
		return Placement{}, err
	}
	return out, nil
}

func overrideFromRow(p dbgen.Placement) (LinkOverride, error) {
	out := LinkOverride{LinkID: *p.LinkID, Hidden: p.Hidden, Waypoints: []Point{}}
	if err := json.Unmarshal(p.Style, &out.Style); err != nil {
		return LinkOverride{}, err
	}
	if len(p.Waypoints) > 0 {
		if err := json.Unmarshal(p.Waypoints, &out.Waypoints); err != nil {
			return LinkOverride{}, err
		}
	}
	return out, nil
}

func shapeFromRow(p dbgen.CanvasShape) (Shape, error) {
	out := Shape{ID: p.ID, Kind: p.Kind, X: p.X, Y: p.Y, Width: p.Width, Height: p.Height, Rotation: p.Rotation, ZIndex: p.ZIndex}
	if err := json.Unmarshal(p.Style, &out.Style); err != nil {
		return Shape{}, err
	}
	if err := json.Unmarshal(p.Data, &out.Data); err != nil {
		return Shape{}, err
	}
	return out, nil
}
