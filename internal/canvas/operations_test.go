package canvas_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Whitedragon115/Netopology-Backend/internal/canvas"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestOperationViewFailureRollsBackBatch(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := canvasWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	var tab canvas.Canvas
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Projection failure"}).Expect(t, 201).JSON(t, &tab)
	var node struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Router", "category": "network", "placeOn": map[string]any{"canvasId": tab.ID}}).Expect(t, 201).JSON(t, &node)
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(t.Context(), `UPDATE nodes SET properties='[{"key":"bad","label":"Bad","value":{}}]'::jsonb WHERE workspace_id=$1 AND id=$2`, wsID, node.ID); err != nil {
		t.Fatal(err)
	}
	owner.Do(http.MethodPost, root+"/canvases/"+tab.ID.String()+"/operations", map[string]any{"operations": []any{map[string]any{"op": "shape.create", "kind": "rectangle", "x": 1, "y": 2}}}).Expect(t, 500)
	var shapes, audits, events int
	if err := conn.QueryRow(t.Context(), `SELECT
	(SELECT count(*) FROM canvas_shapes WHERE workspace_id=$1 AND canvas_id=$2),
	(SELECT count(*) FROM audit_log WHERE workspace_id=$1 AND target_id=$2 AND action='canvas.edit'),
	(SELECT count(*) FROM workspace_events WHERE workspace_id=$1 AND subject_id=$2 AND type='canvas.operations_applied')`, wsID, tab.ID).Scan(&shapes, &audits, &events); err != nil {
		t.Fatal(err)
	}
	if shapes != 0 || audits != 0 || events != 0 {
		t.Fatalf("view failure committed batch: %d %d %d", shapes, audits, events)
	}
}

func TestCanvasOperationsAtomicAndLock(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := canvasWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	var tab canvas.Canvas
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Operations"}).Expect(t, 201).JSON(t, &tab)
	var node struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{
		"name": "Router", "category": "network",
		"placeOn": map[string]any{"canvasId": tab.ID, "x": 1, "y": 2},
	}).Expect(t, 201).JSON(t, &node)
	path := root + "/canvases/" + tab.ID.String()
	item := map[string]any{"type": "node", "id": node.ID}
	resp := owner.Do(http.MethodPost, path+"/operations", map[string]any{"operations": []any{
		map[string]any{"op": "shape.create", "kind": "rectangle", "x": 4, "y": 5},
		map[string]any{"op": "place", "item": item},
	}}).Expect(t, 409)
	if !strings.Contains(string(resp.Body), "Operation 1") {
		t.Fatalf("missing failing index: %s", resp.Body)
	}
	var view canvas.CanvasView
	owner.Do(http.MethodGet, path+"/view", nil).Expect(t, 200).JSON(t, &view)
	if len(view.Shapes) != 0 {
		t.Fatalf("batch did not roll back: %+v", view.Shapes)
	}
	var result canvas.CanvasOperationResult
	owner.Do(http.MethodPost, path+"/operations", map[string]any{"operations": []any{
		map[string]any{"op": "shape.create", "kind": "rectangle", "x": 4, "y": 5},
	}}).Expect(t, 200).JSON(t, &result)
	if len(result.Results) != 1 || result.Results[0].ID == nil || result.View == nil || len(result.View.Shapes) != 1 {
		t.Fatalf("operation result %+v", result)
	}
	var locked canvas.Canvas
	owner.Do(http.MethodPost, path+"/lock", nil).Expect(t, 200).JSON(t, &locked)
	if locked.CanvasLock == nil {
		t.Fatal("canvas was not locked")
	}
	owner.Do(http.MethodPost, path+"/operations", map[string]any{"operations": []any{
		map[string]any{"op": "shape.delete", "shapeId": result.Results[0].ID},
	}}).Expect(t, 423)
	owner.Do(http.MethodDelete, path+"/lock", nil).Expect(t, 200).JSON(t, &locked)
	if locked.CanvasLock != nil {
		t.Fatal("canvas was not unlocked")
	}
	owner.Do(http.MethodPost, path+"/operations", map[string]any{"operations": []any{
		map[string]any{"op": "shape.delete", "shapeId": result.Results[0].ID},
	}}).Expect(t, 200)
}

func TestCanvasEditorGrant(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := canvasWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	var tab canvas.Canvas
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Granted"}).Expect(t, 201).JSON(t, &tab)
	editor := api.ClientAs("editor")
	path := root + "/canvases/" + tab.ID.String()
	editor.Do(http.MethodGet, path+"/view", nil).Expect(t, 404)
	editor.Do(http.MethodPost, path+"/lock", nil).Expect(t, 404)
	var me struct{ ID uuid.UUID }
	editor.Do(http.MethodGet, "/me", nil).Expect(t, 200).JSON(t, &me)
	owner.Do(http.MethodPut, root+"/members/"+me.ID.String()+"/canvas-grants",
		map[string]any{"canvasIds": []uuid.UUID{tab.ID}}).Expect(t, 200)
	editor.Do(http.MethodGet, path+"/view", nil).Expect(t, 200)
	editor.Do(http.MethodPost, path+"/lock", nil).Expect(t, 200)
	editor.Do(http.MethodDelete, path+"/lock", nil).Expect(t, 200)
}
