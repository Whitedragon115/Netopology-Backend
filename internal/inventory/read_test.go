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

func firstWorkspace(t *testing.T, c *testutil.Client) uuid.UUID {
	t.Helper()
	var page struct{ Items []struct{ ID uuid.UUID } }
	c.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200).JSON(t, &page)
	return page.Items[0].ID
}

func userID(t *testing.T, c *testutil.Client) uuid.UUID {
	t.Helper()
	var me struct{ ID uuid.UUID }
	c.Do(http.MethodGet, "/me", nil).Expect(t, 200).JSON(t, &me)
	return me.ID
}

type seed struct{ area, visibleNode, secondNode, hiddenNode, visibleLink, hiddenLink, sensitiveNote, canvasA, canvasB uuid.UUID }

func seedInventory(t *testing.T, api *testutil.API, wsID uuid.UUID) seed {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	s := seed{}
	for _, item := range []struct {
		target *uuid.UUID
		sql    string
		args   []any
	}{
		{&s.area, `INSERT INTO areas (workspace_id,name,kind,cidr,vlan_id) VALUES ($1,'LAN','network','192.0.2.0/24',42) RETURNING id`, []any{wsID}},
		{&s.canvasA, `INSERT INTO canvases (workspace_id,name) VALUES ($1,'Visible') RETURNING id`, []any{wsID}},
		{&s.canvasB, `INSERT INTO canvases (workspace_id,name) VALUES ($1,'Hidden') RETURNING id`, []any{wsID}},
	} {
		if err := conn.QueryRow(t.Context(), item.sql, item.args...).Scan(item.target); err != nil {
			t.Fatal(err)
		}
	}
	fields := `[ {"key":"ip","label":"IP","value":"192.0.2.1","sensitive":true}, {"key":"os","label":"OS","value":"Linux"} ]`
	for _, item := range []struct {
		target *uuid.UUID
		slug   string
	}{{&s.visibleNode, "node-a"}, {&s.secondNode, "node-b"}, {&s.hiddenNode, "node-c"}} {
		if err := conn.QueryRow(t.Context(), `INSERT INTO nodes (workspace_id,area_id,slug,name,category,properties) VALUES ($1,$2,$3,$3,'compute',$4) RETURNING id`, wsID, s.area, item.slug, fields).Scan(item.target); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct{ canvasID, nodeID uuid.UUID }{{s.canvasA, s.visibleNode}, {s.canvasA, s.secondNode}, {s.canvasB, s.visibleNode}, {s.canvasB, s.hiddenNode}} {
		if _, err := conn.Exec(t.Context(), `INSERT INTO placements (workspace_id,canvas_id,node_id) VALUES ($1,$2,$3)`, wsID, item.canvasID, item.nodeID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.Exec(t.Context(), `INSERT INTO placements (workspace_id,canvas_id,area_id) VALUES ($1,$2,$3)`, wsID, s.canvasA, s.area); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		target *uuid.UUID
		second uuid.UUID
	}{{&s.visibleLink, s.secondNode}, {&s.hiddenLink, s.hiddenNode}} {
		if err := conn.QueryRow(t.Context(), `INSERT INTO links (workspace_id,source_node_id,target_node_id,kind) VALUES ($1,$2,$3,'physical') RETURNING id`, wsID, s.visibleNode, item.second).Scan(item.target); err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.QueryRow(t.Context(), `INSERT INTO notes (workspace_id,node_id,body,title,is_sensitive) VALUES ($1,$2,'secret body','secret title',true) RETURNING id`, wsID, s.visibleNode).Scan(&s.sensitiveNote); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestInventoryReadsAndVisibility(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := firstWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	seed := seedInventory(t, api, wsID)
	editor := api.CreateUser("editor", "member")
	viewer := api.CreateUser("viewer", "member")
	owner.Do(http.MethodPost, root+"/members", map[string]any{"userId": userID(t, editor), "role": "editor"}).Expect(t, 201)
	owner.Do(http.MethodPost, root+"/members", map[string]any{"userId": userID(t, viewer), "role": "viewer"}).Expect(t, 201)
	owner.Do(http.MethodPut, root+"/members/"+userID(t, editor).String()+"/canvas-grants", map[string]any{"canvasIds": []uuid.UUID{seed.canvasA}}).Expect(t, 200)
	var nodes inventory.NodePage
	owner.Do(http.MethodGet, root+"/nodes", nil).Expect(t, 200).JSON(t, &nodes)
	if len(nodes.Items) != 3 {
		t.Fatalf("owner nodes %+v", nodes)
	}
	nodes = inventory.NodePage{}
	editor.Do(http.MethodGet, root+"/nodes", nil).Expect(t, 200).JSON(t, &nodes)
	if len(nodes.Items) != 2 {
		t.Fatalf("editor nodes %+v", nodes)
	}
	if nodes.Items[0].Properties[0].Value != nil || !nodes.Items[0].Properties[0].Redacted {
		t.Fatalf("sensitive field leaked %+v", nodes.Items[0].Properties)
	}
	editor.Do(http.MethodGet, root+"/nodes/"+seed.hiddenNode.String(), nil).Expect(t, 404)
	visible := editor.Do(http.MethodGet, root+"/nodes/"+seed.visibleNode.String(), nil).Expect(t, 200)
	if visible.Header.Get("ETag") != `"1"` {
		t.Fatalf("node etag %s", visible.Header.Get("ETag"))
	}
	var usage inventory.NodeUsage
	editor.Do(http.MethodGet, root+"/nodes/"+seed.visibleNode.String()+"/usage", nil).Expect(t, 200).JSON(t, &usage)
	if len(usage.Canvases) != 1 || usage.HiddenCanvasCount != 1 || usage.LinkCount != 1 || usage.NoteCount != 1 {
		t.Fatalf("usage %+v", usage)
	}
	var links inventory.LinkPage
	editor.Do(http.MethodGet, root+"/links", nil).Expect(t, 200).JSON(t, &links)
	if len(links.Items) != 1 || links.Items[0].ID != seed.visibleLink {
		t.Fatalf("editor links %+v", links)
	}
	var visibleLink inventory.Link
	editor.Do(http.MethodGet, root+"/links/"+seed.visibleLink.String(), nil).Expect(t, 200).JSON(t, &visibleLink)
	if visibleLink.SourceNodeID != seed.visibleNode || visibleLink.TargetNodeID != seed.secondNode {
		t.Fatalf("visible link %+v", visibleLink)
	}
	editor.Do(http.MethodGet, root+"/links/"+seed.hiddenLink.String(), nil).Expect(t, 404)
	var areas inventory.AreaPage
	editor.Do(http.MethodGet, root+"/areas", nil).Expect(t, 200).JSON(t, &areas)
	if len(areas.Items) != 1 || areas.Items[0].CIDR != nil || areas.Items[0].VlanID != nil {
		t.Fatalf("area redaction %+v", areas)
	}
	var area inventory.Area
	editor.Do(http.MethodGet, root+"/areas/"+seed.area.String(), nil).Expect(t, 200).JSON(t, &area)
	if area.ID != seed.area || area.NodeCount != 2 {
		t.Fatalf("visible area %+v", area)
	}
	var note inventory.Note
	editor.Do(http.MethodGet, root+"/notes/"+seed.sensitiveNote.String(), nil).Expect(t, 200).JSON(t, &note)
	if note.Title != "" || note.Body != "" || !note.Redacted {
		t.Fatalf("note leaked %+v", note)
	}
	viewer.Do(http.MethodGet, root+"/nodes/"+seed.hiddenNode.String(), nil).Expect(t, 200)
	var notes inventory.NotePage
	owner.Do(http.MethodGet, root+"/notes?nodeId="+seed.visibleNode.String(), nil).Expect(t, 200).JSON(t, &notes)
	if len(notes.Items) != 1 || notes.Items[0].ID != seed.sensitiveNote || notes.Items[0].Body != "secret body" {
		t.Fatalf("owner notes %+v", notes)
	}
	owner.Do(http.MethodGet, root+"/notes?nodeId="+seed.visibleNode.String()+"&areaId="+seed.area.String(), nil).Expect(t, 422)
	var first inventory.NodePage
	owner.Do(http.MethodGet, root+"/nodes?limit=1", nil).Expect(t, 200).JSON(t, &first)
	if len(first.Items) != 1 || first.NextCursor == nil {
		t.Fatalf("pagination %+v", first)
	}
	var second inventory.NodePage
	owner.Do(http.MethodGet, root+"/nodes?limit=1&cursor="+*first.NextCursor, nil).Expect(t, 200).JSON(t, &second)
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("next page %+v", second)
	}
}
