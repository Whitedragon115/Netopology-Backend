package canvas

import (
	"context"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// ReadForGuest is called within the share service's transaction after rechecking its grant.
func (s *Service) ReadForGuest(ctx context.Context, q *dbgen.Queries) (apimodel.CanvasView, error) {
	grant := authz.PrincipalFrom(ctx).Guest
	if grant == nil {
		return CanvasView{}, apperr.Unauthenticated("auth.required", "Unlock the share link first.")
	}
	row, err := q.GetCanvasRow(ctx, dbgen.GetCanvasRowParams{WorkspaceID: grant.WorkspaceID, ID: grant.CanvasID})
	if pg.IsNotFound(err) {
		return CanvasView{}, apperr.ErrNotFound()
	}
	if err != nil {
		return CanvasView{}, err
	}
	actor := &authz.Actor{Principal: &authz.Principal{}, WorkspaceID: grant.WorkspaceID, Role: authz.RoleViewer}
	view, err := s.view(ctx, q, actor, row)
	if err != nil {
		return CanvasView{}, err
	}
	ids := apimodel.IDs{}
	placed := map[uuid.UUID]bool{}
	for _, p := range view.Placements {
		placed[p.Item.ID] = true
		switch p.Item.Type {
		case "node":
			ids.Nodes = append(ids.Nodes, p.Item.ID)
		case "area":
			ids.Areas = append(ids.Areas, p.Item.ID)
		case "note":
			ids.Notes = append(ids.Notes, p.Item.ID)
		}
	}
	// permissions.md §4: guests get no inventory or notes outside the shared canvas.
	for _, link := range view.Links {
		if placed[link.SourceNodeID] && placed[link.TargetNodeID] {
			ids.Links = append(ids.Links, link.ID)
		}
	}
	snapshot, err := s.reader.ReadForGuest(ctx, q, ids)
	if err != nil {
		return CanvasView{}, err
	}
	view.Nodes, view.Links, view.Areas, view.Notes = snapshot.Nodes, snapshot.Links, snapshot.Areas, snapshot.Notes
	allowed := map[uuid.UUID]bool{}
	for _, n := range view.Nodes {
		allowed[n.ID] = true
	}
	for _, a := range view.Areas {
		allowed[a.ID] = true
	}
	for _, n := range view.Notes {
		allowed[n.ID] = true
	}
	placements := []Placement{}
	for _, p := range view.Placements {
		if allowed[p.Item.ID] {
			placements = append(placements, p)
		}
	}
	view.Placements = placements
	links := map[uuid.UUID]bool{}
	for _, l := range view.Links {
		links[l.ID] = true
	}
	overrides := []LinkOverride{}
	for _, o := range view.LinkOverrides {
		if links[o.LinkID] {
			overrides = append(overrides, o)
		}
	}
	view.LinkOverrides = overrides
	for i := range view.Nodes {
		if view.Nodes[i].AreaID != nil && !allowed[*view.Nodes[i].AreaID] {
			view.Nodes[i].AreaID = nil
		}
	}
	for i := range view.Areas {
		if view.Areas[i].ParentID != nil && !allowed[*view.Areas[i].ParentID] {
			view.Areas[i].ParentID = nil
		}
	}
	view.Canvas.CanvasLock = nil
	view.Viewer = Viewer{Role: "guest", CanEdit: false, CanViewSensitive: false}
	view.Breadcrumb = []BreadcrumbItem{}
	return view, nil
}
