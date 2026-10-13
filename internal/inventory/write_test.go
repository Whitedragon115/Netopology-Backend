package inventory_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Whitedragon115/Netopology-Backend/internal/inventory"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestInventoryWriteLifecycle(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := firstWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	var area inventory.Area
	created := owner.Do(http.MethodPost, root+"/areas", map[string]any{"name": "Rack A", "kind": "site", "cidr": "192.0.2.0/24"}).Expect(t, 201)
	created.JSON(t, &area)
	if area.ID == uuid.Nil || created.Header.Get("ETag") != `"1"` {
		t.Fatalf("created area %+v, ETag %s", area, created.Header.Get("ETag"))
	}
	owner.Do(http.MethodPatch, root+"/areas/"+area.ID.String(), map[string]any{}).Expect(t, 422)
	owner.Do(http.MethodPatch, root+"/areas/"+area.ID.String(), map[string]any{"name": "Rack B"}, `If-Match: "0"`).Expect(t, 422)
	var updatedArea inventory.Area
	owner.Do(http.MethodPatch, root+"/areas/"+area.ID.String(), map[string]any{"name": "Rack B"}, `If-Match: "1"`).Expect(t, 200).JSON(t, &updatedArea)
	if updatedArea.Version != 2 || updatedArea.Name != "Rack B" {
		t.Fatalf("updated area %+v", updatedArea)
	}
	var clearedArea inventory.Area
	owner.Do(http.MethodPatch, root+"/areas/"+area.ID.String(), `{"parentId":null,"cidr":null,"vlanId":null}`).Expect(t, 200).JSON(t, &clearedArea)
	if clearedArea.Version != 3 || clearedArea.CIDR != nil || clearedArea.VlanID != nil {
		t.Fatalf("nullable area fields %+v", clearedArea)
	}
	owner.Do(http.MethodPatch, root+"/areas/"+area.ID.String(), map[string]any{"name": "stale"}, `If-Match: "1"`).Expect(t, 412)
	var nodeA, nodeB inventory.Node
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Router One", "category": "network", "areaId": area.ID, "properties": []map[string]any{{"key": "ip", "label": "IP", "value": "192.0.2.1", "sensitive": true}}}).Expect(t, 201).JSON(t, &nodeA)
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Router One", "category": "network"}).Expect(t, 201).JSON(t, &nodeB)
	if nodeA.Slug != "router-one" || nodeB.Slug != "router-one-2" {
		t.Fatalf("generated slugs %s %s", nodeA.Slug, nodeB.Slug)
	}
	var health inventory.Node
	owner.Do(http.MethodPut, root+"/nodes/"+nodeA.ID.String()+"/health", map[string]any{"status": "down", "message": "offline"}).Expect(t, 200).JSON(t, &health)
	if health.Health.Status != "down" || health.Health.Message == nil {
		t.Fatalf("health %+v", health.Health)
	}
	owner.Do(http.MethodPatch, root+"/nodes/"+nodeA.ID.String(), map[string]any{"name": "Router Two"}, `If-Match: "1"`).Expect(t, 412)
	var changedNode inventory.Node
	owner.Do(http.MethodPatch, root+"/nodes/"+nodeA.ID.String(), map[string]any{"name": "Router Two"}).Expect(t, 200).JSON(t, &changedNode)
	if changedNode.Name != "Router Two" {
		t.Fatalf("node %+v", changedNode)
	}
	var link inventory.Link
	owner.Do(http.MethodPost, root+"/links", map[string]any{"sourceNodeId": nodeA.ID, "targetNodeId": nodeB.ID, "kind": "physical"}).Expect(t, 201).JSON(t, &link)
	var changedLink inventory.Link
	owner.Do(http.MethodPatch, root+"/links/"+link.ID.String(), map[string]any{"label": "uplink"}, `If-Match: "1"`).Expect(t, 200).JSON(t, &changedLink)
	if changedLink.Label != "uplink" {
		t.Fatalf("link %+v", changedLink)
	}
	var note inventory.Note
	owner.Do(http.MethodPost, root+"/notes", map[string]any{"target": map[string]any{"type": "link", "id": link.ID}, "body": "inspect", "isSensitive": true}).Expect(t, 201).JSON(t, &note)
	var changedNote inventory.Note
	owner.Do(http.MethodPatch, root+"/notes/"+note.ID.String(), map[string]any{"title": "Important"}, `If-Match: "1"`).Expect(t, 200).JSON(t, &changedNote)
	if changedNote.Title != "Important" {
		t.Fatalf("note %+v", changedNote)
	}
	owner.Do(http.MethodDelete, root+"/notes/"+note.ID.String(), nil, `If-Match: "2"`).Expect(t, 204)
	owner.Do(http.MethodDelete, root+"/links/"+link.ID.String(), nil, `If-Match: "2"`).Expect(t, 204)
	owner.Do(http.MethodDelete, root+"/nodes/"+nodeA.ID.String(), nil).Expect(t, 204)
	owner.Do(http.MethodDelete, root+"/nodes/"+nodeB.ID.String(), nil).Expect(t, 204)
	owner.Do(http.MethodDelete, root+"/areas/"+area.ID.String(), nil, `If-Match: "3"`).Expect(t, 204)
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	var auditCount, eventCount int
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE workspace_id=$1 AND action IN
		('area.create','area.update','area.delete','node.create','node.update','node.delete','node.health_set',
		'link.create','link.update','link.delete','note.create','note.update','note.delete')`, wsID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM workspace_events WHERE workspace_id=$1`, wsID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 16 || eventCount != 16 {
		t.Fatalf("transactional audit/events: %d/%d, want 16/16", auditCount, eventCount)
	}
}

func TestInventoryEditorWriteRules(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := firstWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	editor := api.ClientAs("editor")
	viewer := api.ClientAs("viewer")
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	var granted, hidden uuid.UUID
	for _, target := range []*uuid.UUID{&granted, &hidden} {
		if err := conn.QueryRow(t.Context(), `INSERT INTO canvases (workspace_id,name) VALUES ($1,$2) RETURNING id`, wsID, uuid.NewString()).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	owner.Do(http.MethodPut, root+"/members/"+userID(t, editor).String()+"/canvas-grants", map[string]any{"canvasIds": []uuid.UUID{granted}}).Expect(t, 200)
	viewer.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Denied", "category": "generic"}).Expect(t, 403)
	editor.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "No place", "category": "generic"}).Expect(t, 422)
	editor.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Bad place", "category": "generic", "placeOn": map[string]any{"canvasId": hidden}}).Expect(t, 404)
	var node inventory.Node
	editor.Do(http.MethodPost, root+"/nodes", map[string]any{
		"name": "Placed", "category": "generic", "placeOn": map[string]any{"canvasId": granted},
		"properties": []map[string]any{{"key": "token", "label": "Token", "value": "secret-value", "sensitive": true}},
	}).Expect(t, 201).JSON(t, &node)
	if len(node.Properties) != 1 || node.Properties[0].Value != nil || !node.Properties[0].Redacted {
		t.Fatalf("editor secret %+v", node.Properties)
	}
	var revised inventory.Node
	editor.Do(http.MethodPatch, root+"/nodes/"+node.ID.String(), map[string]any{"properties": []map[string]any{{"key": "token", "label": "Token", "sensitive": true, "redacted": true}}}).Expect(t, 200).JSON(t, &revised)
	var ownerRead inventory.Node
	owner.Do(http.MethodGet, root+"/nodes/"+node.ID.String(), nil).Expect(t, 200).JSON(t, &ownerRead)
	if ownerRead.Properties[0].Value == nil || *ownerRead.Properties[0].Value != "secret-value" {
		t.Fatalf("redacted field lost %+v", ownerRead.Properties)
	}
	var hiddenNode inventory.Node
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Hidden endpoint", "category": "generic", "placeOn": map[string]any{"canvasId": hidden}}).Expect(t, 201).JSON(t, &hiddenNode)
	editor.Do(http.MethodPost, root+"/links", map[string]any{"sourceNodeId": node.ID, "targetNodeId": hiddenNode.ID, "kind": "physical"}).Expect(t, 404)
	editor.Do(http.MethodPost, root+"/notes", map[string]any{"target": map[string]any{"type": "workspace"}, "body": "forbidden", "placeOn": map[string]any{"canvasId": granted}}).Expect(t, 403)
	if _, err := conn.Exec(t.Context(), `INSERT INTO placements (workspace_id,canvas_id,node_id) VALUES ($1,$2,$3)`, wsID, hidden, node.ID); err != nil {
		t.Fatal(err)
	}
	editor.Do(http.MethodDelete, root+"/nodes/"+node.ID.String(), nil).Expect(t, 409)
	owner.Do(http.MethodPatch, root+"/nodes/"+node.ID.String(), map[string]any{"isProtected": true}).Expect(t, 200)
	editor.Do(http.MethodPatch, root+"/nodes/"+node.ID.String(), map[string]any{"isProtected": false}).Expect(t, 403)
	editor.Do(http.MethodDelete, root+"/nodes/"+node.ID.String(), nil).Expect(t, 403)
}

func TestAreaCycleValidation(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	root := "/workspaces/" + firstWorkspace(t, owner).String()
	var outer, inner inventory.Area
	owner.Do(http.MethodPost, root+"/areas", map[string]any{"name": "Outer"}).Expect(t, 201).JSON(t, &outer)
	owner.Do(http.MethodPost, root+"/areas", map[string]any{"name": "Inner", "parentId": outer.ID}).Expect(t, 201).JSON(t, &inner)
	res := owner.Do(http.MethodPatch, root+"/areas/"+outer.ID.String(), map[string]any{"parentId": inner.ID}).Expect(t, 409)
	if res.Code() != "area.cycle" {
		t.Fatalf("cycle code %s", res.Code())
	}
}
