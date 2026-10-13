package server_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

// Completes permissions.md §5 rows 1, 2, 4, 7 and 9 across feature boundaries.
func TestWorkspacePermissionMatrix(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	var page struct{ Items []struct{ ID uuid.UUID } }
	owner.Do(http.MethodGet, "/workspaces", nil).JSON(t, &page)
	root := "/workspaces/" + page.Items[0].ID.String()
	var canvas, node, other, area, note, link, share struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Protected", "isProtected": true}).Expect(t, 201).JSON(t, &canvas)
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Protected", "category": "network", "isProtected": true}).Expect(t, 201).JSON(t, &node)
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Other", "category": "network"}).Expect(t, 201).JSON(t, &other)
	owner.Do(http.MethodPost, root+"/areas", map[string]any{"name": "Rack"}).Expect(t, 201).JSON(t, &area)
	owner.Do(http.MethodPost, root+"/notes", map[string]any{"target": map[string]any{"type": "workspace"}, "body": "Memo"}).Expect(t, 201).JSON(t, &note)
	owner.Do(http.MethodPost, root+"/links", map[string]any{"sourceNodeId": node.ID, "targetNodeId": other.ID, "kind": "physical"}).Expect(t, 201).JSON(t, &link)
	var created struct{ ShareLink struct{ ID uuid.UUID } }
	owner.Do(http.MethodPost, root+"/canvases/"+canvas.ID.String()+"/share-links", map[string]any{}).Expect(t, 201).JSON(t, &created)
	share.ID = created.ShareLink.ID
	var account struct{ ID uuid.UUID }
	owner.Do(http.MethodGet, "/me", nil).JSON(t, &account)
	viewer := api.ClientAs("viewer")
	nodePath := root + "/nodes/" + node.ID.String()
	areaPath := root + "/areas/" + area.ID.String()
	notePath := root + "/notes/" + note.ID.String()
	linkPath := root + "/links/" + link.ID.String()
	canvasPath := root + "/canvases/" + canvas.ID.String()
	sharePath := root + "/share-links/" + share.ID.String()
	memberPath := root + "/members/" + account.ID.String()
	for _, tc := range []struct {
		name, method, path string
		body               any
	}{
		{"workspace update", http.MethodPatch, root, map[string]any{"name": "No"}},
		{"workspace delete", http.MethodDelete, root, nil},
		{"area create", http.MethodPost, root + "/areas", map[string]any{"name": "No"}},
		{"area update", http.MethodPatch, areaPath, map[string]any{"name": "No"}},
		{"area delete", http.MethodDelete, areaPath, nil},
		{"node create", http.MethodPost, root + "/nodes", map[string]any{"name": "No", "category": "network"}},
		{"node update", http.MethodPatch, nodePath, map[string]any{"name": "No"}},
		{"node delete", http.MethodDelete, nodePath, nil},
		{"health", http.MethodPut, nodePath + "/health", map[string]any{"status": "down"}},
		{"link create", http.MethodPost, root + "/links", map[string]any{"sourceNodeId": node.ID, "targetNodeId": other.ID, "kind": "logical"}},
		{"link update", http.MethodPatch, linkPath, map[string]any{"label": "No"}},
		{"link delete", http.MethodDelete, linkPath, nil},
		{"note create", http.MethodPost, root + "/notes", map[string]any{"target": map[string]any{"type": "workspace"}, "body": "No"}},
		{"note update", http.MethodPatch, notePath, map[string]any{"body": "No"}},
		{"note delete", http.MethodDelete, notePath, nil},
		{"canvas create", http.MethodPost, root + "/canvases", map[string]any{"name": "No"}},
		{"canvas update", http.MethodPatch, canvasPath, map[string]any{"name": "No"}},
		{"canvas delete", http.MethodDelete, canvasPath, nil},
		{"canvas edit", http.MethodPost, canvasPath + "/operations", map[string]any{"operations": []map[string]any{{"op": "shape.create", "kind": "rect", "x": 0, "y": 0}}}},
		{"canvas lock", http.MethodPost, canvasPath + "/lock", nil},
		{"canvas unlock", http.MethodDelete, canvasPath + "/lock", nil},
		{"share create", http.MethodPost, canvasPath + "/share-links", map[string]any{}},
		{"share update", http.MethodPatch, sharePath, map[string]any{"label": "No"}},
		{"share revoke", http.MethodDelete, sharePath, nil},
		{"member add", http.MethodPost, root + "/members", map[string]any{"userId": uuid.New(), "role": "viewer"}},
		{"member update", http.MethodPatch, memberPath, map[string]any{"role": "viewer"}},
		{"member remove other", http.MethodDelete, memberPath, nil},
		{"member grants", http.MethodPut, memberPath + "/canvas-grants", map[string]any{"canvasIds": []uuid.UUID{}}},
	} {
		t.Run("viewer/"+tc.name, func(t *testing.T) {
			response := viewer.Do(tc.method, tc.path, tc.body).Expect(t, 403)
			if response.Code() != "role.insufficient" {
				t.Fatalf("code %s", response.Code())
			}
		})
	}
	outsider := api.CreateUser("outsider", "admin")
	for _, path := range []string{root, nodePath, canvasPath, canvasPath + "/view", canvasPath + "/share-links", root + "/events", root + "/audit-log"} {
		outsider.Do(http.MethodGet, path, nil).Expect(t, 404)
	}
	viewer.Do(http.MethodPost, "/workspaces", map[string]any{"name": "No"}).Expect(t, 403)
	admin := api.ClientAs("admin")
	for _, path := range []string{nodePath, canvasPath} {
		admin.Do(http.MethodDelete, path, nil).Expect(t, 403)
		admin.Do(http.MethodPatch, path, map[string]any{"isProtected": false}).Expect(t, 403)
	}
	editor := api.ClientAs("editor")
	var ed struct{ ID uuid.UUID }
	editor.Do(http.MethodGet, "/me", nil).JSON(t, &ed)
	owner.Do(http.MethodPut, root+"/members/"+ed.ID.String()+"/canvas-grants", map[string]any{"canvasIds": []uuid.UUID{canvas.ID}}).Expect(t, 200)
	owner.Do(http.MethodPost, canvasPath+"/lock", nil).Expect(t, 200)
	for _, client := range []*testutil.Client{owner, admin, editor} {
		client.Do(http.MethodPost, canvasPath+"/operations", map[string]any{"operations": []map[string]any{{"op": "shape.create", "kind": "rect", "x": 0, "y": 0}}}).Expect(t, 423)
	}
	editor.Do(http.MethodDelete, canvasPath+"/lock", nil).Expect(t, 200)
}
