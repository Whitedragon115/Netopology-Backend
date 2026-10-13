package account_test

import (
	"net/http"
	"testing"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

// permissions.md §5 row 7 and §3 Instance: server roles and workspace roles stay separate.
func TestAccountPermissionMatrix(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	admin := api.CreateUser("admin", "admin")
	otherAdmin := api.CreateUser("otheradmin", "admin")
	member := api.CreateUser("member", "member")
	ownerID := accountID(t, owner.Do(http.MethodGet, "/me", nil).Expect(t, 200).Body)
	adminID := accountID(t, admin.Do(http.MethodGet, "/me", nil).Expect(t, 200).Body)
	otherAdminID := accountID(t, otherAdmin.Do(http.MethodGet, "/me", nil).Expect(t, 200).Body)
	memberID := accountID(t, member.Do(http.MethodGet, "/me", nil).Expect(t, 200).Body)

	for _, tc := range []struct {
		name         string
		client       *testutil.Client
		method, path string
		body         any
		status       int
	}{
		{"anonymous list", api.Client(), http.MethodGet, "/users", nil, 401},
		{"member list", member, http.MethodGet, "/users", nil, 403},
		{"admin list", admin, http.MethodGet, "/users", nil, 200},
		{"owner list", owner, http.MethodGet, "/users", nil, 200},
		{"member create", member, http.MethodPost, "/users", map[string]any{"username": "blocked", "password": "blocked-password-123"}, 403},
		{"admin grant admin", admin, http.MethodPost, "/users", map[string]any{"username": "blockedadmin", "password": "blocked-password-123", "instanceRole": "admin"}, 403},
		{"admin edit owner", admin, http.MethodPatch, "/users/" + ownerID.String(), map[string]any{"displayName": "No"}, 403},
		{"admin edit other admin", admin, http.MethodPatch, "/users/" + otherAdminID.String(), map[string]any{"displayName": "No"}, 403},
		{"admin grant owner", admin, http.MethodPatch, "/users/" + memberID.String(), map[string]any{"instanceRole": "owner"}, 403},
		{"admin grant admin by update", admin, http.MethodPatch, "/users/" + memberID.String(), map[string]any{"instanceRole": "admin"}, 403},
		{"admin reset owner", admin, http.MethodPost, "/users/" + ownerID.String() + "/password", map[string]any{"newPassword": "changed-password-123"}, 403},
		{"admin reset other admin", admin, http.MethodPost, "/users/" + otherAdminID.String() + "/password", map[string]any{"newPassword": "changed-password-123"}, 403},
		{"member settings", member, http.MethodGet, "/instance/settings", nil, 403},
		{"admin settings", admin, http.MethodGet, "/instance/settings", nil, 200},
		{"member lookup", member, http.MethodGet, "/users/lookup?username=owner", nil, 403},
		{"admin without membership lookup", admin, http.MethodGet, "/users/lookup?username=owner", nil, 403},
		{"owner lookup", owner, http.MethodGet, "/users/lookup?username=member", nil, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.client.Do(tc.method, tc.path, tc.body).Expect(t, tc.status)
		})
	}

	owner.Do(http.MethodPatch, "/users/"+ownerID.String(), map[string]any{"instanceRole": "member"}).Expect(t, 409)
	owner.Do(http.MethodPatch, "/users/"+ownerID.String(), map[string]any{"status": "disabled"}).Expect(t, 409)
	owner.Do(http.MethodPatch, "/users/"+memberID.String(), map[string]any{"instanceRole": "owner"}).Expect(t, 200)
	var previous struct{ InstanceRole string }
	owner.Do(http.MethodGet, "/me", nil).Expect(t, 200).JSON(t, &previous)
	if previous.InstanceRole != "admin" {
		t.Fatalf("old owner role %q", previous.InstanceRole)
	}
	member.Do(http.MethodGet, "/users/"+adminID.String(), nil).Expect(t, 200)
}
