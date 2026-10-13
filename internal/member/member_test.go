package member_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Whitedragon115/Netopology-Backend/internal/member"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func workspace(t *testing.T, owner *testutil.Client) uuid.UUID {
	t.Helper()
	var page struct{ Items []struct{ ID uuid.UUID } }
	owner.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200).JSON(t, &page)
	return page.Items[0].ID
}

func user(t *testing.T, client *testutil.Client) uuid.UUID {
	t.Helper()
	var me struct{ ID uuid.UUID }
	client.Do(http.MethodGet, "/me", nil).Expect(t, 200).JSON(t, &me)
	return me.ID
}

func TestMemberLifecycleAndRules(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := workspace(t, owner)
	base := "/workspaces/" + wsID.String() + "/members"
	admin := api.CreateUser("admin", "admin")
	editor := api.CreateUser("editor", "member")
	viewer := api.CreateUser("viewer", "member")
	adminID, editorID, viewerID := user(t, admin), user(t, editor), user(t, viewer)
	for _, tc := range []struct {
		id   uuid.UUID
		role string
	}{{adminID, "admin"}, {editorID, "editor"}, {viewerID, "viewer"}} {
		res := owner.Do(http.MethodPost, base, member.MemberAdd{UserID: tc.id, Role: tc.role}).Expect(t, 201)
		var result member.Member
		res.JSON(t, &result)
		if result.Role != tc.role || result.User.ID != tc.id || result.CanViewSensitive == nil {
			t.Fatalf("added %+v", result)
		}
	}
	owner.Do(http.MethodPost, base, member.MemberAdd{UserID: editorID, Role: "editor"}).Expect(t, 409)
	viewer.Do(http.MethodPost, base, member.MemberAdd{UserID: uuid.New(), Role: "viewer"}).Expect(t, 403)
	admin.Do(http.MethodPost, base, member.MemberAdd{UserID: uuid.New(), Role: "owner"}).Expect(t, 403)
	owner.Do(http.MethodPost, base, member.MemberAdd{UserID: uuid.New(), Role: "viewer"}).Expect(t, 404)
	owner.Do(http.MethodDelete, base+"/"+user(t, owner).String(), nil).Expect(t, 409)
	admin.Do(http.MethodPatch, base+"/"+user(t, owner).String(), map[string]any{"role": "viewer"}).Expect(t, 403)
	admin.Do(http.MethodDelete, base+"/"+user(t, owner).String(), nil).Expect(t, 403)
	viewer.Do(http.MethodPatch, base+"/"+editorID.String(), map[string]any{"role": "viewer"}).Expect(t, 403)
	viewer.Do(http.MethodDelete, base+"/"+editorID.String(), nil).Expect(t, 403)
	owner.Do(http.MethodPatch, base+"/"+editorID.String(), map[string]any{"canViewSensitive": true}).Expect(t, 200)
	admin.Do(http.MethodPatch, base+"/"+editorID.String(), map[string]any{"role": "viewer"}).Expect(t, 200)
	owner.Do(http.MethodPatch, base+"/"+editorID.String(), map[string]any{}).Expect(t, 422)

	var page member.MemberPage
	owner.Do(http.MethodGet, base, nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 4 {
		t.Fatalf("owner list %+v", page)
	}
	res := editor.Do(http.MethodGet, base, nil).Expect(t, 200)
	var redacted struct{ Items []map[string]json.RawMessage }
	res.JSON(t, &redacted)
	if len(redacted.Items) != 4 {
		t.Fatalf("editor list %+v", redacted)
	}
	for _, item := range redacted.Items {
		for _, hidden := range []string{"canViewSensitive", "canvasIds", "createdAt"} {
			if _, ok := item[hidden]; ok {
				t.Fatalf("leaked %s: %s", hidden, item[hidden])
			}
		}
	}
	viewer.Do(http.MethodDelete, base+"/"+viewerID.String(), nil).Expect(t, 204)
	viewer.Do(http.MethodGet, base, nil).Expect(t, 404)
	var count int
	if err := api.DB.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE workspace_id=$1 AND action IN ('member.add','member.update','member.remove')`, wsID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 6 {
		t.Fatalf("audit count %d", count)
	}
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM workspace_events WHERE workspace_id=$1 AND type='member.changed'`, wsID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 6 {
		t.Fatalf("event count %d", count)
	}
}

func TestMemberCanvasGrants(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := workspace(t, owner)
	base := "/workspaces/" + wsID.String() + "/members"
	editor := api.CreateUser("editor", "member")
	id := user(t, editor)
	owner.Do(http.MethodPost, base, member.MemberAdd{UserID: id, Role: "editor"}).Expect(t, 201)
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	var tab, inner uuid.UUID
	if err := conn.QueryRow(t.Context(), `INSERT INTO canvases (workspace_id,name) VALUES ($1,'Tab') RETURNING id`, wsID).Scan(&tab); err != nil {
		t.Fatal(err)
	}
	var area uuid.UUID
	if err := conn.QueryRow(t.Context(), `INSERT INTO areas (workspace_id,name,kind) VALUES ($1,'Area','custom') RETURNING id`, wsID).Scan(&area); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(t.Context(), `INSERT INTO canvases (workspace_id,name,area_id) VALUES ($1,'Inner',$2) RETURNING id`, wsID, area).Scan(&inner); err != nil {
		t.Fatal(err)
	}
	path := base + "/" + id.String() + "/canvas-grants"
	owner.Do(http.MethodPut, path, member.CanvasGrants{CanvasIDs: []uuid.UUID{inner}}).Expect(t, 422)
	owner.Do(http.MethodPut, path, member.CanvasGrants{CanvasIDs: []uuid.UUID{tab, tab}}).Expect(t, 422)
	var result member.Member
	owner.Do(http.MethodPut, path, member.CanvasGrants{CanvasIDs: []uuid.UUID{tab}}).Expect(t, 200).JSON(t, &result)
	if result.CanvasIDs == nil || len(*result.CanvasIDs) != 1 || (*result.CanvasIDs)[0] != tab {
		t.Fatalf("grants %+v", result)
	}
	owner.Do(http.MethodPut, path, member.CanvasGrants{CanvasIDs: []uuid.UUID{tab}}).Expect(t, 200)
	owner.Do(http.MethodPut, path, member.CanvasGrants{CanvasIDs: []uuid.UUID{}}).Expect(t, 200).JSON(t, &result)
	if result.CanvasIDs == nil || len(*result.CanvasIDs) != 0 {
		t.Fatalf("grants not cleared %+v", result)
	}
	owner.Do(http.MethodPut, base+"/"+user(t, owner).String()+"/canvas-grants", member.CanvasGrants{CanvasIDs: []uuid.UUID{tab}}).Expect(t, 422)
}
