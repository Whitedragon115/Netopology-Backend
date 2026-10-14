package inventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type visibility struct {
	all bool
	ids dbgen.VisibleInventoryIDsRow
}

// visibility is shared by the HTTP reads and the canvas reader. Only editors are filtered.
func visible(ctx context.Context, q *dbgen.Queries, actor *authz.Actor) (visibility, error) {
	if actor.Role != authz.RoleEditor {
		return visibility{all: true}, nil
	}
	ids, err := q.VisibleInventoryIDs(ctx, dbgen.VisibleInventoryIDsParams{WorkspaceID: actor.WorkspaceID, UserID: actor.UserID()})
	if err != nil {
		return visibility{}, fmt.Errorf("compute editor visibility: %w", err)
	}
	return visibility{ids: ids}, nil
}

func audience(actor *authz.Actor) Audience { return Audience{Sensitive: actor.SeesSensitive()} }

func pageCursor(cursor string) (*uuid.UUID, error) {
	if cursor == "" {
		return nil, nil
	}
	var raw string
	if err := httpx.DecodeCursor(cursor, &raw); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperr.Invalid(apperr.CodeValidation, "Invalid cursor.")
	}
	return &id, nil
}

func pageLimit(limit int) int32 {
	if limit < 1 {
		return 50
	}
	if limit > 200 {
		return 201
	}
	return int32(limit + 1)
}

func more[T any](rows []T, limit int, id func(T) uuid.UUID) ([]T, *string) {
	if limit < 1 {
		limit = 50
	}
	if len(rows) <= limit {
		return rows, nil
	}
	rows = rows[:limit]
	return rows, httpx.EncodeCursor(id(rows[len(rows)-1]).String())
}

type (
	AreaFilter struct {
		Cursor string
		Limit  int
		Parent string
	}
	NodeFilter struct {
		Cursor                   string
		Limit                    int
		AreaID, CanvasID         *uuid.UUID
		Category, Health, Search *string
	}
	LinkFilter struct {
		Cursor string
		Limit  int
		NodeID *uuid.UUID
		Kind   *string
	}
	NoteFilter struct {
		Cursor                           string
		Limit                            int
		NodeID, LinkID, AreaID, CanvasID *uuid.UUID
		Kind                             *string
	}
)

func (s *Service) ListAreas(ctx context.Context, wsID uuid.UUID, in AreaFilter) (AreaPage, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.AreaList)
	if err != nil {
		return AreaPage{}, err
	}
	after, err := pageCursor(in.Cursor)
	if err != nil {
		return AreaPage{}, err
	}
	var parent *uuid.UUID
	if in.Parent != "" && in.Parent != "root" {
		id, err := uuid.Parse(in.Parent)
		if err != nil {
			return AreaPage{}, apperr.Invalid(apperr.CodeValidation, "Invalid parentId.")
		}
		parent = &id
	}
	page := AreaPage{Items: []Area{}}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		rows, err := q.ListAreasVisible(ctx, dbgen.ListAreasVisibleParams{
			WorkspaceID: wsID, AllVisible: vis.all, VisibleIds: vis.ids.AreaIds, VisibleNodeIds: vis.ids.NodeIds,
			FilterParent: in.Parent != "", RootOnly: in.Parent == "root", ParentID: parent,
			AfterID: after, PageLimit: pageLimit(in.Limit),
		})
		if err != nil {
			return err
		}
		rows, page.NextCursor = more(rows, in.Limit, func(row dbgen.ListAreasVisibleRow) uuid.UUID { return row.ID })
		for _, row := range rows {
			area, err := areaFromRow(areaBase(row.ID, row.WorkspaceID, row.ParentID, row.Name, row.Kind, row.Color, row.Cidr, row.VlanID, row.Description, row.Properties, row.CreatedBy, row.CreatedAt, row.UpdatedAt, row.Version), row.NodeCount, audience(actor))
			if err != nil {
				return err
			}
			page.Items = append(page.Items, area)
		}
		return nil
	})
	return page, err
}

func (s *Service) GetArea(ctx context.Context, wsID, id uuid.UUID) (Area, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.AreaRead)
	if err != nil {
		return Area{}, err
	}
	var out Area
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		out, err = readArea(ctx, q, actor, id)
		return err
	})
	return out, err
}

func readArea(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, id uuid.UUID) (Area, error) {
	vis, err := visible(ctx, q, actor)
	if err != nil {
		return Area{}, err
	}
	row, err := q.GetAreaVisible(ctx, dbgen.GetAreaVisibleParams{WorkspaceID: actor.WorkspaceID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.AreaIds, VisibleNodeIds: vis.ids.NodeIds})
	if pg.IsNotFound(err) {
		return Area{}, apperr.ErrNotFound()
	}
	if err != nil {
		return Area{}, err
	}
	return areaFromRow(areaBase(row.ID, row.WorkspaceID, row.ParentID, row.Name, row.Kind, row.Color, row.Cidr, row.VlanID, row.Description, row.Properties, row.CreatedBy, row.CreatedAt, row.UpdatedAt, row.Version), row.NodeCount, audience(actor))
}

func (s *Service) ListNodes(ctx context.Context, wsID uuid.UUID, in NodeFilter) (NodePage, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NodeList)
	if err != nil {
		return NodePage{}, err
	}
	after, err := pageCursor(in.Cursor)
	if err != nil {
		return NodePage{}, err
	}
	page := NodePage{Items: []Node{}}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		rows, err := q.ListNodesVisible(ctx, dbgen.ListNodesVisibleParams{
			WorkspaceID: wsID, AllVisible: vis.all, VisibleIds: vis.ids.NodeIds,
			AreaID: in.AreaID, Category: in.Category, CanvasID: in.CanvasID,
			Health: in.Health, Search: in.Search, AfterID: after, PageLimit: pageLimit(in.Limit),
		})
		if err != nil {
			return err
		}
		rows, page.NextCursor = more(rows, in.Limit, func(row dbgen.Node) uuid.UUID { return row.ID })
		for _, row := range rows {
			node, err := nodeFromRow(row, audience(actor))
			if err != nil {
				return err
			}
			page.Items = append(page.Items, node)
		}
		return nil
	})
	return page, err
}

func (s *Service) GetNode(ctx context.Context, wsID, id uuid.UUID) (Node, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NodeRead)
	if err != nil {
		return Node{}, err
	}
	var out Node
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		out, err = readNode(ctx, q, actor, id)
		return err
	})
	return out, err
}

func readNode(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, id uuid.UUID) (Node, error) {
	vis, err := visible(ctx, q, actor)
	if err != nil {
		return Node{}, err
	}
	row, err := q.GetNodeVisible(ctx, dbgen.GetNodeVisibleParams{WorkspaceID: actor.WorkspaceID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.NodeIds})
	if pg.IsNotFound(err) {
		return Node{}, apperr.ErrNotFound()
	}
	if err != nil {
		return Node{}, err
	}
	return nodeFromRow(row, audience(actor))
}

func (s *Service) GetNodeUsage(ctx context.Context, wsID, id uuid.UUID) (NodeUsage, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NodeUsage)
	if err != nil {
		return NodeUsage{}, err
	}
	out := NodeUsage{Canvases: []NodeUsageCanvas{}}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		if _, err := q.GetNodeVisible(ctx, dbgen.GetNodeVisibleParams{WorkspaceID: wsID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.NodeIds}); pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		} else if err != nil {
			return err
		}
		rows, err := q.ListNodeUsageCanvases(ctx, dbgen.ListNodeUsageCanvasesParams{WorkspaceID: wsID, NodeID: &id})
		if err != nil {
			return err
		}
		visibleCanvas := map[uuid.UUID]bool{}
		for _, canvasID := range vis.ids.CanvasIds {
			visibleCanvas[canvasID] = true
		}
		for _, row := range rows {
			if vis.all || visibleCanvas[row.ID] {
				out.Canvases = append(out.Canvases, NodeUsageCanvas{ID: row.ID, Name: row.Name})
			} else {
				out.HiddenCanvasCount++
			}
		}
		out.LinkCount, err = q.CountNodeLinks(ctx, dbgen.CountNodeLinksParams{WorkspaceID: wsID, NodeID: id, AllVisible: vis.all, VisibleIds: vis.ids.LinkIds})
		if err != nil {
			return err
		}
		out.NoteCount, err = q.CountNodeNotes(ctx, dbgen.CountNodeNotesParams{WorkspaceID: wsID, NodeID: &id, AllVisible: vis.all, VisibleIds: vis.ids.NoteIds})
		return err
	})
	return out, err
}

func (s *Service) ListLinks(ctx context.Context, wsID uuid.UUID, in LinkFilter) (LinkPage, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.LinkList)
	if err != nil {
		return LinkPage{}, err
	}
	after, err := pageCursor(in.Cursor)
	if err != nil {
		return LinkPage{}, err
	}
	page := LinkPage{Items: []Link{}}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		rows, err := q.ListLinksVisible(ctx, dbgen.ListLinksVisibleParams{
			WorkspaceID: wsID, AllVisible: vis.all, VisibleIds: vis.ids.LinkIds,
			NodeID: in.NodeID, Kind: in.Kind, AfterID: after, PageLimit: pageLimit(in.Limit),
		})
		if err != nil {
			return err
		}
		rows, page.NextCursor = more(rows, in.Limit, func(row dbgen.Link) uuid.UUID { return row.ID })
		for _, row := range rows {
			link, err := linkFromRow(row, audience(actor))
			if err != nil {
				return err
			}
			page.Items = append(page.Items, link)
		}
		return nil
	})
	return page, err
}

func (s *Service) GetLink(ctx context.Context, wsID, id uuid.UUID) (Link, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.LinkRead)
	if err != nil {
		return Link{}, err
	}
	var out Link
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		out, err = readLink(ctx, q, actor, id)
		return err
	})
	return out, err
}

func readLink(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, id uuid.UUID) (Link, error) {
	vis, err := visible(ctx, q, actor)
	if err != nil {
		return Link{}, err
	}
	row, err := q.GetLinkVisible(ctx, dbgen.GetLinkVisibleParams{WorkspaceID: actor.WorkspaceID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.LinkIds})
	if pg.IsNotFound(err) {
		return Link{}, apperr.ErrNotFound()
	}
	if err != nil {
		return Link{}, err
	}
	return linkFromRow(row, audience(actor))
}

func (s *Service) ListNotes(ctx context.Context, wsID uuid.UUID, in NoteFilter) (NotePage, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NoteList)
	if err != nil {
		return NotePage{}, err
	}
	after, err := pageCursor(in.Cursor)
	if err != nil {
		return NotePage{}, err
	}
	targetType, targetID, err := noteFilterTarget(in)
	if err != nil {
		return NotePage{}, err
	}
	page := NotePage{Items: []Note{}}
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		vis, err := visible(ctx, q, actor)
		if err != nil {
			return err
		}
		rows, err := q.ListNotesVisible(ctx, dbgen.ListNotesVisibleParams{
			WorkspaceID: wsID, AllVisible: vis.all, VisibleIds: vis.ids.NoteIds,
			TargetType: targetType, TargetID: targetID, Kind: in.Kind,
			AfterID: after, PageLimit: pageLimit(in.Limit),
		})
		if err != nil {
			return err
		}
		rows, page.NextCursor = more(rows, in.Limit, func(row dbgen.Note) uuid.UUID { return row.ID })
		for _, row := range rows {
			note, ok := noteFromRow(row, audience(actor))
			if ok {
				page.Items = append(page.Items, note)
			}
		}
		return nil
	})
	return page, err
}

func (s *Service) GetNote(ctx context.Context, wsID, id uuid.UUID) (Note, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.NoteRead)
	if err != nil {
		return Note{}, err
	}
	var out Note
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		out, err = readNote(ctx, q, actor, id)
		return err
	})
	return out, err
}

func readNote(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, id uuid.UUID) (Note, error) {
	vis, err := visible(ctx, q, actor)
	if err != nil {
		return Note{}, err
	}
	row, err := q.GetNoteVisible(ctx, dbgen.GetNoteVisibleParams{WorkspaceID: actor.WorkspaceID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.NoteIds})
	if pg.IsNotFound(err) {
		return Note{}, apperr.ErrNotFound()
	}
	if err != nil {
		return Note{}, err
	}
	out, ok := noteFromRow(row, audience(actor))
	if !ok {
		return Note{}, apperr.ErrNotFound()
	}
	return out, nil
}

func noteFilterTarget(in NoteFilter) (string, *uuid.UUID, error) {
	var selected string
	var id *uuid.UUID
	for _, item := range []struct {
		kind string
		id   *uuid.UUID
	}{{"node", in.NodeID}, {"link", in.LinkID}, {"area", in.AreaID}, {"canvas", in.CanvasID}} {
		if item.id == nil {
			continue
		}
		if selected != "" {
			return "", nil, apperr.Invalid(apperr.CodeValidation, "Filter notes by only one target.")
		}
		selected, id = item.kind, item.id
	}
	if selected == "" {
		selected = "workspace"
	}
	return selected, id, nil
}

func cleanSearch(query string) *string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	return &query
}
