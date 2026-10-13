package auth_test

import (
	"net/http"
	"testing"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

// permissions.md §3: providers and login are public; logout and me.read need an account.
func TestSessionPermissions(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	anon := api.Client()
	for _, tc := range []struct {
		name   string
		client *testutil.Client
		method string
		path   string
		want   int
		code   string
	}{
		{"anonymous providers", anon, http.MethodGet, "/auth/providers", http.StatusOK, ""},
		{"anonymous logout", anon, http.MethodPost, "/auth/logout", http.StatusUnauthorized, "auth.required"},
		{"anonymous me", anon, http.MethodGet, "/me", http.StatusUnauthorized, "auth.required"},
		{"account me", owner, http.MethodGet, "/me", http.StatusOK, ""},
		{"account logout", owner, http.MethodPost, "/auth/logout", http.StatusNoContent, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.client.Do(tc.method, tc.path, nil).Expect(t, tc.want)
			if tc.code != "" && r.Code() != tc.code {
				t.Fatalf("code = %q, want %q", r.Code(), tc.code)
			}
		})
	}
}
