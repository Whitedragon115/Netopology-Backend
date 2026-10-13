package share_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/audit"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
	"github.com/Whitedragon115/Netopology-Backend/internal/share"
)

func fixture(t *testing.T) (*testutil.API, *testutil.Client, string, uuid.UUID) {
	t.Helper()
	api := testutil.NewAPI(t)
	owner := api.Owner()
	var ws struct{ Items []struct{ ID uuid.UUID } }
	owner.Do(http.MethodGet, "/workspaces", nil).JSON(t, &ws)
	root := "/workspaces/" + ws.Items[0].ID.String()
	var tab struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Shared"}).Expect(t, 201).JSON(t, &tab)
	return api, owner, root, tab.ID
}

func TestGuestViewAndLinkLifecycle(t *testing.T) {
	api, owner, root, canvasID := fixture(t)
	var node struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Visible", "category": "network", "properties": []map[string]any{{"key": "public", "label": "Public", "value": "hello"}, {"key": "secret", "label": "Secret", "value": "DO-NOT-SEND", "sensitive": true}}, "placeOn": map[string]any{"canvasId": canvasID, "x": 0, "y": 0}}).Expect(t, 201).JSON(t, &node)
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "UNPLACED", "category": "network"}).Expect(t, 201)
	owner.Do(http.MethodPost, root+"/areas", map[string]any{"name": "Shared VLAN", "kind": "vlan", "cidr": "10.10.0.0/24", "vlanId": 77, "placeOn": map[string]any{"canvasId": canvasID, "x": 100, "y": 100}}).Expect(t, 201)
	owner.Do(http.MethodPut, root+"/nodes/"+node.ID.String()+"/health", map[string]any{"status": "healthy", "message": "PRIVATE-HEALTH"}).Expect(t, 200)
	for _, sensitive := range []bool{false, true} {
		owner.Do(http.MethodPost, root+"/notes", map[string]any{"target": map[string]any{"type": "workspace"}, "title": "Placed note", "body": "Note body", "isSensitive": sensitive, "placeOn": map[string]any{"canvasId": canvasID, "x": 10, "y": 20}}).Expect(t, 201)
	}
	owner.Do(http.MethodPost, root+"/notes", map[string]any{"target": map[string]any{"type": "workspace"}, "title": "UNPLACED-NOTE", "body": "Not on canvas"}).Expect(t, 201)
	var created share.ShareLinkCreated
	owner.Do(http.MethodPost, root+"/canvases/"+canvasID.String()+"/share-links", map[string]any{"label": "Guests", "password": "share-password"}).Expect(t, 201).JSON(t, &created)
	guest := api.Client()
	path := "/share/" + created.Token
	guest.Do(http.MethodGet, path, nil).Expect(t, 200)
	guest.Do(http.MethodGet, path+"/view", nil).Expect(t, 401)
	guest.Do(http.MethodPost, path+"/unlock", map[string]any{"password": "wrong"}).Expect(t, 401)
	unlocked := guest.Do(http.MethodPost, path+"/unlock", map[string]any{"password": "share-password"}).Expect(t, 204)
	cookie := unlocked.Cookies()[0]
	if !cookie.HttpOnly || cookie.Path != "/api/v1/share" || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie %+v", cookie)
	}
	response := guest.Do(http.MethodGet, path+"/view", nil).Expect(t, 200)
	var view apimodel.CanvasView
	response.JSON(t, &view)
	if view.Viewer.Role != "guest" || view.Viewer.CanEdit || view.Viewer.CanViewSensitive || len(view.Nodes) != 1 || len(view.Notes) != 1 || len(view.Placements) != 3 {
		t.Fatalf("guest view %+v", view)
	}
	for _, secret := range []string{"DO-NOT-SEND", "PRIVATE-HEALTH", "UNPLACED", "createdBy", "updatedBy", "\"key\":\"secret\"", "\"cidr\"", "\"vlanId\""} {
		if strings.Contains(string(response.Body), secret) {
			t.Fatalf("guest exposed %s: %s", secret, response.Body)
		}
	}
	guest.Do(http.MethodGet, "/me", nil, "Cookie: "+cookie.Name+"="+cookie.Value).Expect(t, 401)
	guest.Do(http.MethodGet, root+"/nodes", nil, "Cookie: "+cookie.Name+"="+cookie.Value).Expect(t, 401)
	var links share.ShareLinkPage
	listed := owner.Do(http.MethodGet, root+"/canvases/"+canvasID.String()+"/share-links", nil).Expect(t, 200)
	listed.JSON(t, &links)
	if len(links.Items) != 1 || links.Items[0].UseCount != 1 || links.Items[0].LastUsedAt == nil || strings.Contains(string(listed.Body), created.Token) || strings.Contains(string(listed.Body), "password_hash") {
		t.Fatalf("list %s", listed.Body)
	}
	linkPath := root + "/share-links/" + created.ShareLink.ID.String()
	owner.Do(http.MethodPatch, linkPath, map[string]any{"password": "new-password", "label": "Changed"}).Expect(t, 200)
	guest.Do(http.MethodGet, path+"/view", nil).Expect(t, 401)
	guest.Do(http.MethodPost, path+"/unlock", map[string]any{"password": "new-password"}).Expect(t, 204)
	owner.Do(http.MethodPatch, linkPath, map[string]any{"password": nil, "expiresAt": nil}).Expect(t, 200)
	guest.Do(http.MethodGet, path+"/view", nil).Expect(t, 401)
	guest.Do(http.MethodPost, path+"/unlock", nil).Expect(t, 204)
	owner.Do(http.MethodDelete, linkPath, nil).Expect(t, 204)
	guest.Do(http.MethodGet, path, nil).Expect(t, 404)
	guest.Do(http.MethodGet, path+"/view", nil).Expect(t, 404)
	guest.Do(http.MethodPost, path+"/unlock", nil).Expect(t, 404)
	var page audit.AuditPage
	owner.Do(http.MethodGet, root+"/audit-log?action=share.", nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 3 {
		t.Fatalf("unlock audits %+v", page)
	}
	for _, entry := range page.Items {
		if entry.Actor.Type != "share_link" {
			t.Fatalf("wrong unlock actor %+v", entry)
		}
	}
}

func TestSharePermissionsExpiryAndRateLimit(t *testing.T) {
	api, owner, root, canvasID := fixture(t)
	collection := root + "/canvases/" + canvasID.String() + "/share-links"
	var link share.ShareLinkCreated
	owner.Do(http.MethodPost, collection, map[string]any{"password": "password-for-guests"}).Expect(t, 201).JSON(t, &link)
	for _, role := range []string{"admin", "editor", "viewer"} {
		member := api.ClientAs(role)
		status := 403
		if role == "admin" {
			status = 200
		}
		member.Do(http.MethodGet, collection, nil).Expect(t, status)
		write := 403
		if role == "admin" {
			write = 201
		}
		member.Do(http.MethodPost, collection, map[string]any{}).Expect(t, write)
		member.Do(http.MethodPatch, root+"/share-links/"+link.ShareLink.ID.String(), map[string]any{"label": role}).Expect(t, status)
	}
	api.CreateUser("outsider", "admin").Do(http.MethodGet, collection, nil).Expect(t, 404)
	guest := api.Client()
	path := "/share/" + link.Token + "/unlock"
	for range 10 {
		guest.Do(http.MethodPost, path, map[string]any{"password": "wrong"}).Expect(t, 401)
	}
	limited := guest.Do(http.MethodPost, path, map[string]any{"password": "wrong"}).Expect(t, 429)
	if limited.Header.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	owner.Do(http.MethodPost, collection, map[string]any{"expiresAt": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}).Expect(t, 201).JSON(t, &link)
	guest.Do(http.MethodGet, "/share/"+link.Token, nil).Expect(t, 404)
	guest.Do(http.MethodGet, "/share/"+strings.Repeat("x", 43), nil).Expect(t, 404)
}
