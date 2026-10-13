package canvas_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/canvas"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func canvasWorkspace(t *testing.T, c *testutil.Client) uuid.UUID {
	t.Helper()
	var page struct{ Items []struct{ ID uuid.UUID } }
	c.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200).JSON(t, &page)
	return page.Items[0].ID
}

func TestCanvasListAndView(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := canvasWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	var tab canvas.Canvas
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Overview"}).Expect(t, 201).JSON(t, &tab)
	if tab.Kind != "tab" || tab.Version != 1 {
		t.Fatalf("canvas %+v", tab)
	}
	var area struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/areas", map[string]any{"name": "Rack"}).Expect(t, 201).JSON(t, &area)
	var node struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{
		"name": "Router", "category": "network", "areaId": area.ID,
		"placeOn": map[string]any{"canvasId": tab.ID, "x": 100, "y": 200},
	}).Expect(t, 201).JSON(t, &node)
	var page canvas.CanvasPage
	owner.Do(http.MethodGet, root+"/canvases", nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 || page.Items[0].ID != tab.ID {
		t.Fatalf("canvas list %+v", page)
	}
	var view canvas.CanvasView
	owner.Do(http.MethodGet, root+"/canvases/"+tab.ID.String()+"/view", nil).Expect(t, 200).JSON(t, &view)
	if len(view.Nodes) != 1 || view.Nodes[0].ID != node.ID || len(view.Placements) != 1 || view.Placements[0].X != 100 {
		t.Fatalf("canvas view %+v", view)
	}
	var inner canvas.CanvasView
	owner.Do(http.MethodGet, root+"/areas/"+area.ID.String()+"/canvas", nil).Expect(t, 200).JSON(t, &inner)
	if inner.Canvas.Kind != "area" || len(inner.Nodes) != 1 || inner.Nodes[0].ID != node.ID {
		t.Fatalf("inner canvas %+v", inner)
	}
}

func TestAreaCanvasBoundaryAndBreadcrumb(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := canvasWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	var tab canvas.Canvas
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Map"}).Expect(t, 201).JSON(t, &tab)
	var area struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/areas", map[string]any{"name": "Rack"}).Expect(t, 201).JSON(t, &area)
	var inside, outside struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{
		"name": "Inside", "category": "network", "areaId": area.ID,
		"placeOn": map[string]any{"canvasId": tab.ID, "x": 10, "y": 20},
	}).Expect(t, 201).JSON(t, &inside)
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{
		"name": "Outside", "category": "network",
		"placeOn": map[string]any{"canvasId": tab.ID, "x": 300, "y": 20},
	}).Expect(t, 201).JSON(t, &outside)
	var link struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/links", map[string]any{
		"sourceNodeId": inside.ID, "targetNodeId": outside.ID, "kind": "physical",
	}).Expect(t, 201).JSON(t, &link)
	owner.Do(http.MethodPost, root+"/canvases/"+tab.ID.String()+"/operations", map[string]any{
		"operations": []any{map[string]any{"op": "place", "item": map[string]any{"type": "area", "id": area.ID}}},
	}).Expect(t, 200)
	var inner canvas.CanvasView
	owner.Do(http.MethodGet, root+"/areas/"+area.ID.String()+"/canvas", nil).Expect(t, 200).JSON(t, &inner)
	if len(inner.Breadcrumb) != 2 || inner.Breadcrumb[0].CanvasID != tab.ID || inner.Breadcrumb[1].AreaID == nil || *inner.Breadcrumb[1].AreaID != area.ID {
		t.Fatalf("breadcrumb %+v", inner.Breadcrumb)
	}
	if len(inner.Links) != 1 || inner.Links[0].ID != link.ID || len(inner.Nodes) != 2 || len(inner.Placements) != 1 {
		t.Fatalf("area boundary view: links=%+v nodes=%+v placements=%+v", inner.Links, inner.Nodes, inner.Placements)
	}
}
