package inventory_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestAuditNamesSurviveDeletion(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := firstWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	var source, target struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Source", "category": "generic"}).Expect(t, 201).JSON(t, &source)
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Target", "category": "generic"}).Expect(t, 201).JSON(t, &target)
	for _, item := range []struct {
		path, kind, field string
		body              map[string]any
	}{
		{"areas", "area", "name", map[string]any{"name": "Original", "kind": "site"}},
		{"nodes", "node", "name", map[string]any{"name": "Original", "category": "generic"}},
		{"links", "link", "label", map[string]any{"label": "Original", "sourceNodeId": source.ID, "targetNodeId": target.ID, "kind": "physical"}},
		{"notes", "note", "title", map[string]any{"title": "Original", "body": "DO-NOT-AUDIT", "isSensitive": true, "target": map[string]any{"type": "workspace"}}},
		{"canvases", "canvas", "name", map[string]any{"name": "Original"}},
	} {
		t.Run(item.kind, func(t *testing.T) {
			var created struct{ ID uuid.UUID }
			owner.Do(http.MethodPost, root+"/"+item.path, item.body).Expect(t, 201).JSON(t, &created)
			path := root + "/" + item.path + "/" + created.ID.String()
			owner.Do(http.MethodPatch, path, map[string]any{item.field: "Renamed"}).Expect(t, 200)
			owner.Do(http.MethodDelete, path, nil).Expect(t, 204)
			for _, action := range []string{"create", "update", "delete"} {
				var name, metadata string
				if err := conn.QueryRow(t.Context(), `SELECT metadata->>'name',metadata::text FROM audit_log WHERE workspace_id=$1 AND target_id=$2 AND action=$3`, wsID, created.ID, item.kind+"."+action).Scan(&name, &metadata); err != nil {
					t.Fatal(err)
				}
				want := "Renamed"
				if action == "create" {
					want = "Original"
				}
				if name != want {
					t.Fatalf("%s: name %q, want %q; metadata %s", action, name, want, metadata)
				}
				if item.kind == "note" && metadata != `{"name": "`+want+`"}` {
					t.Fatalf("unexpected note metadata: %s", metadata)
				}
			}
		})
	}
}
