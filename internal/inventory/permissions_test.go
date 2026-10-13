package inventory_test

import (
	"net/http"
	"testing"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestInventoryReadPermissionMatrix(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := firstWorkspace(t, owner)
	root := "/workspaces/" + wsID.String()
	seed := seedInventory(t, api, wsID)
	editor := api.ClientAs("editor")
	viewer := api.ClientAs("viewer")
	outsider := api.CreateUser("outsider", "admin")
	for _, tc := range []struct {
		name   string
		client *testutil.Client
		path   string
		status int
	}{
		{"owner area", owner, root + "/areas/" + seed.area.String(), 200},
		{"viewer node", viewer, root + "/nodes/" + seed.hiddenNode.String(), 200},
		{"editor hidden node", editor, root + "/nodes/" + seed.hiddenNode.String(), 404},
		{"editor hidden link", editor, root + "/links/" + seed.hiddenLink.String(), 404},
		{"editor hidden note", editor, root + "/notes/" + seed.sensitiveNote.String(), 404},
		{"outsider areas", outsider, root + "/areas", 404},
		{"outsider nodes", outsider, root + "/nodes", 404},
		{"outsider links", outsider, root + "/links", 404},
		{"outsider notes", outsider, root + "/notes", 404},
	} {
		t.Run(tc.name, func(t *testing.T) { tc.client.Do(http.MethodGet, tc.path, nil).Expect(t, tc.status) })
	}
}
