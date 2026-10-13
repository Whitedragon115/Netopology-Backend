package audit_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/audit"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestAuditPaginationFiltersAndPermissions(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	var ws struct{ Items []struct{ ID uuid.UUID } }
	owner.Do(http.MethodGet, "/workspaces", nil).Expect(t, 200).JSON(t, &ws)
	root := "/workspaces/" + ws.Items[0].ID.String()
	var node struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Audit router", "category": "network"}).Expect(t, 201).JSON(t, &node)
	owner.Do(http.MethodPatch, root+"/nodes/"+node.ID.String(), map[string]any{"name": "Renamed"}).Expect(t, 200)
	var page audit.AuditPage
	owner.Do(http.MethodGet, root+"/audit-log?action=node.&limit=1", nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 || page.Items[0].Action != "node.update" || page.NextCursor == nil || page.Items[0].Actor.DisplayName == nil {
		t.Fatalf("first page %+v", page)
	}
	first := page.Items[0].ID
	owner.Do(http.MethodGet, root+"/audit-log?action=node.&limit=1&cursor="+url.QueryEscape(*page.NextCursor), nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 || page.Items[0].Action != "node.create" || page.Items[0].ID == first || page.NextCursor != nil {
		t.Fatalf("second page %+v", page)
	}
	actor := page.Items[0].Actor.ID
	owner.Do(http.MethodGet, root+"/audit-log?action=node.create&targetId="+node.ID.String()+"&actorId="+actor.String(), nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 || page.Items[0].Target == nil || page.Items[0].Metadata == nil {
		t.Fatalf("filtered %+v", page)
	}
	owner.Do(http.MethodGet, root+"/audit-log?since="+url.QueryEscape(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)), nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 0 {
		t.Fatal("since filter returned old entries")
	}
	for _, role := range []string{"admin", "editor", "viewer"} {
		member := api.ClientAs(role)
		status := 403
		if role == "admin" {
			status = 200
		}
		member.Do(http.MethodGet, root+"/audit-log", nil).Expect(t, status)
	}
	api.CreateUser("outsider", "admin").Do(http.MethodGet, root+"/audit-log", nil).Expect(t, 404)
	api.Client().Do(http.MethodGet, root+"/audit-log", nil).Expect(t, 401)
	owner.Do(http.MethodGet, root+"/audit-log?cursor=broken", nil).Expect(t, 422)
}
