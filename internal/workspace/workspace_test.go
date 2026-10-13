package workspace_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
	"github.com/Whitedragon115/Netopology-Backend/internal/workspace"
)

func TestWorkspaceLifecycleAndPermissions(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	admin := api.CreateUser("admin", "admin")
	member := api.CreateUser("member", "member")
	ownerList := owner.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200)
	var page workspace.Page
	ownerList.JSON(t, &page)
	if len(page.Items) != 1 || page.Items[0].Membership.Role != "owner" || !page.Items[0].Membership.CanViewSensitive {
		t.Fatalf("owner list: %+v", page)
	}
	firstID := page.Items[0].ID
	if len(page.Items[0].Settings) != 0 {
		t.Fatalf("settings: %+v", page.Items[0].Settings)
	}
	member.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 0 {
		t.Fatalf("member list: %+v", page)
	}
	member.Do(http.MethodGet, "/workspaces/"+firstID.String(), nil).Expect(t, 404)
	member.Do(http.MethodPost, "/workspaces", map[string]any{"name": "No"}).Expect(t, 403)
	created := admin.Do(http.MethodPost, "/workspaces", map[string]any{"name": "Second Lab"}).Expect(t, 201)
	if created.Header.Get("ETag") != `"1"` {
		t.Fatalf("etag %s", created.Header.Get("ETag"))
	}
	var second workspace.Workspace
	created.JSON(t, &second)
	if second.Slug != "second-lab" || second.Membership.Role != "owner" || !second.Membership.CanViewSensitive {
		t.Fatalf("created: %+v", second)
	}
	owner.Do(http.MethodGet, "/workspaces/"+second.ID.String(), nil).Expect(t, 404)
	owner.Do(http.MethodPost, "/workspaces", map[string]any{"name": "Duplicate", "slug": "second-lab"}).Expect(t, 409)
	owner.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 {
		t.Fatalf("owner list leaked: %+v", page)
	}

	path := "/workspaces/" + second.ID.String()
	admin.Do(http.MethodPatch, path, map[string]any{"name": "Updated", "settings": map[string]any{"theme": "dark"}}, `If-Match: "2"`).Expect(t, 412)
	changed := admin.Do(http.MethodPatch, path, map[string]any{"name": "Updated", "settings": map[string]any{"theme": "dark"}}, `If-Match: "1"`).Expect(t, 200)
	if changed.Header.Get("ETag") != `"2"` {
		t.Fatalf("updated etag %s", changed.Header.Get("ETag"))
	}
	changed.JSON(t, &second)
	if second.Settings["theme"] != "dark" || second.Version != 2 {
		t.Fatalf("updated: %+v", second)
	}
	admin.Do(http.MethodPatch, path, map[string]any{}).Expect(t, 422)
	admin.Do(http.MethodDelete, path, nil, `If-Match: "1"`).Expect(t, 412)
	admin.Do(http.MethodPatch, path, map[string]any{"isProtected": true}).Expect(t, 200)
	admin.Do(http.MethodDelete, path, nil).Expect(t, 403)
	admin.Do(http.MethodPatch, path, map[string]any{"isProtected": false}).Expect(t, 200)
	for _, action := range []string{"workspace.create", "workspace.update"} {
		var n int
		if err := api.DB.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE action = $1 AND target_id = $2`, action, second.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			t.Fatalf("missing %s audit", action)
		}
	}
	admin.Do(http.MethodDelete, path, nil).Expect(t, 204)
	admin.Do(http.MethodGet, path, nil).Expect(t, 404)

	var n int
	if err := api.DB.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE action = 'workspace.delete' AND target_id = $1 AND workspace_id IS NULL`, second.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("missing deletion audit")
	}
}

func TestWorkspaceAdminCannotUnprotect(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	admin := api.CreateUser("admin", "admin")
	viewer := api.CreateUser("viewer", "member")
	var page workspace.Page
	owner.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200).JSON(t, &page)
	id := page.Items[0].ID
	adminUser, err := api.DB.Q().GetUserByUsername(t.Context(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	viewerUser, err := api.DB.Q().GetUserByUsername(t.Context(), "viewer")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []struct {
		id   uuid.UUID
		role string
	}{{adminUser.ID, "admin"}, {viewerUser.ID, "viewer"}} {
		if err := api.DB.Q().InsertMember(t.Context(), dbgen.InsertMemberParams{WorkspaceID: id, UserID: m.id, Role: m.role}); err != nil {
			t.Fatal(err)
		}
	}
	path := "/workspaces/" + id.String()
	viewer.Do(http.MethodGet, path, nil).Expect(t, 200)
	viewer.Do(http.MethodPatch, path, map[string]any{"name": "Denied"}).Expect(t, 403)
	viewer.Do(http.MethodDelete, path, nil).Expect(t, 403)
	admin.Do(http.MethodPatch, path, map[string]any{"isProtected": true}).Expect(t, 200)
	admin.Do(http.MethodPatch, path, map[string]any{"isProtected": false}).Expect(t, 403)
	owner.Do(http.MethodPatch, path, map[string]any{"isProtected": false}).Expect(t, 200)
	admin.Do(http.MethodDelete, path, nil).Expect(t, 403)
	owner.Do(http.MethodDelete, path, nil).Expect(t, 204)
}
