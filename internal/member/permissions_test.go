package member_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestMemberPermissionMatrix(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := workspace(t, owner)
	base := "/workspaces/" + wsID.String() + "/members"
	admin := api.ClientAs("admin")
	editor := api.ClientAs("editor")
	viewer := api.ClientAs("viewer")
	outsider := api.CreateUser("outsider", "member")
	ownerID := user(t, owner)
	for _, tc := range []struct {
		name         string
		client       *testutil.Client
		method, path string
		body         any
		status       int
	}{
		{"owner list", owner, http.MethodGet, base, nil, 200},
		{"admin list", admin, http.MethodGet, base, nil, 200},
		{"editor list", editor, http.MethodGet, base, nil, 200},
		{"viewer list", viewer, http.MethodGet, base, nil, 200},
		{"outsider list", outsider, http.MethodGet, base, nil, 404},
		{"editor add", editor, http.MethodPost, base, map[string]any{"userId": uuid.New(), "role": "viewer"}, 403},
		{"viewer add", viewer, http.MethodPost, base, map[string]any{"userId": uuid.New(), "role": "viewer"}, 403},
		{"editor update", editor, http.MethodPatch, base + "/" + ownerID.String(), map[string]any{"role": "viewer"}, 403},
		{"viewer grants", viewer, http.MethodPut, base + "/" + ownerID.String() + "/canvas-grants", map[string]any{"canvasIds": []any{}}, 403},
		{"editor remove other", editor, http.MethodDelete, base + "/" + ownerID.String(), nil, 403},
		{"outsider remove", outsider, http.MethodDelete, base + "/" + ownerID.String(), nil, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.client.Do(tc.method, tc.path, tc.body).Expect(t, tc.status)
		})
	}
}
