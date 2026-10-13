package inventory

import (
	"context"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// ReadForActor reads requested IDs through the same visibility and redaction path as the API.
// The caller supplies a workspace-scoped transaction and an Actor authorized for that workspace.
func (s *Service) ReadForActor(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, ids IDs) (Snapshot, error) {
	if actor == nil || actor.WorkspaceID == uuid.Nil {
		return Snapshot{}, apperr.ErrNotFound()
	}
	vis, err := visible(ctx, q, actor)
	if err != nil {
		return Snapshot{}, err
	}
	return readSnapshot(ctx, q, actor.WorkspaceID, vis, audience(actor), ids)
}

// ReadForGuest is for a share view after the share service has derived IDs from its unlocked
// canvas. It refuses calls without a guest principal and always applies guest redaction.
func (s *Service) ReadForGuest(ctx context.Context, q *dbgen.Queries, ids IDs) (Snapshot, error) {
	grant := authz.PrincipalFrom(ctx).Guest
	if grant == nil {
		return Snapshot{}, apperr.ErrNotFound()
	}
	return readSnapshot(ctx, q, grant.WorkspaceID, visibility{all: true}, Audience{Guest: true}, ids)
}

func readSnapshot(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, vis visibility, audience Audience, ids IDs) (Snapshot, error) {
	out := Snapshot{Areas: []Area{}, Nodes: []Node{}, Links: []Link{}, Notes: []Note{}}
	for _, id := range ids.Areas {
		row, err := q.GetAreaVisible(ctx, dbgen.GetAreaVisibleParams{WorkspaceID: wsID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.AreaIds, VisibleNodeIds: vis.ids.NodeIds})
		if pg.IsNotFound(err) {
			continue
		}
		if err != nil {
			return Snapshot{}, err
		}
		area, err := areaFromRow(areaBase(row.ID, row.WorkspaceID, row.ParentID, row.Name, row.Kind, row.Color, row.Cidr, row.VlanID, row.Description, row.Properties, row.CreatedBy, row.CreatedAt, row.UpdatedAt, row.Version), row.NodeCount, audience)
		if err != nil {
			return Snapshot{}, err
		}
		out.Areas = append(out.Areas, area)
	}
	for _, id := range ids.Nodes {
		row, err := q.GetNodeVisible(ctx, dbgen.GetNodeVisibleParams{WorkspaceID: wsID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.NodeIds})
		if pg.IsNotFound(err) {
			continue
		}
		if err != nil {
			return Snapshot{}, err
		}
		node, err := nodeFromRow(row, audience)
		if err != nil {
			return Snapshot{}, err
		}
		out.Nodes = append(out.Nodes, node)
	}
	for _, id := range ids.Links {
		row, err := q.GetLinkVisible(ctx, dbgen.GetLinkVisibleParams{WorkspaceID: wsID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.LinkIds})
		if pg.IsNotFound(err) {
			continue
		}
		if err != nil {
			return Snapshot{}, err
		}
		link, err := linkFromRow(row, audience)
		if err != nil {
			return Snapshot{}, err
		}
		out.Links = append(out.Links, link)
	}
	for _, id := range ids.Notes {
		row, err := q.GetNoteVisible(ctx, dbgen.GetNoteVisibleParams{WorkspaceID: wsID, ID: id, AllVisible: vis.all, VisibleIds: vis.ids.NoteIds})
		if pg.IsNotFound(err) {
			continue
		}
		if err != nil {
			return Snapshot{}, err
		}
		note, ok := noteFromRow(row, audience)
		if ok {
			out.Notes = append(out.Notes, note)
		}
	}
	return out, nil
}
