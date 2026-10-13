package setup_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

// Reference integration test: real HTTP stack, real Postgres, black-box through the API.
func TestFirstRunSetup(t *testing.T) {
	api := testutil.NewAPI(t)
	anon := api.Client()

	var st struct{ NeedsSetup bool }
	anon.Do(http.MethodGet, "/setup", nil).Expect(t, http.StatusOK).JSON(t, &st)
	if !st.NeedsSetup {
		t.Fatal("fresh database should need setup")
	}

	// validation comes from the contract (password minLength 12)
	if r := anon.Do(http.MethodPost, "/setup", map[string]any{"username": "alice", "password": "short"}); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("short password: status %d, want 422", r.StatusCode)
	}

	var me struct {
		Username, InstanceRole string
		HasPassword            bool
		Identities             []string
	}
	res := anon.Do(http.MethodPost, "/setup", map[string]any{
		"username": "alice", "password": "a-long-enough-password", "workspaceName": "Home lab",
	}).Expect(t, http.StatusCreated)
	res.JSON(t, &me)
	if me.Username != "alice" || me.InstanceRole != "owner" || !me.HasPassword || me.Identities == nil {
		t.Fatalf("unexpected account: %+v", me)
	}
	if len(res.Cookies()) == 0 || res.Cookies()[0].Name != "netopology_session" || !res.Cookies()[0].HttpOnly {
		t.Fatalf("expected an HttpOnly session cookie, got %v", res.Cookies())
	}

	anon.Do(http.MethodGet, "/setup", nil).Expect(t, http.StatusOK).JSON(t, &st)
	if st.NeedsSetup {
		t.Fatal("setup should be done")
	}

	// permissions.md §5 row 8: setup.complete after the first account exists -> 409
	r := api.Client().Do(http.MethodPost, "/setup", map[string]any{"username": "mallory", "password": "another-long-password"})
	r.Expect(t, http.StatusConflict)
	if r.Code() != "setup.already_completed" {
		t.Fatalf("code %q", r.Code())
	}
}

// Two browsers racing the setup form: exactly one wins.
func TestSetupRace(t *testing.T) {
	api := testutil.NewAPI(t)
	var wg sync.WaitGroup
	codes := make([]int, 5)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = api.Client().Do(http.MethodPost, "/setup", map[string]any{
				"username": "user" + string(rune('a'+i)), "password": "a-long-enough-password",
			}).StatusCode
		}()
	}
	wg.Wait()
	created := 0
	for _, c := range codes {
		if c == http.StatusCreated {
			created++
		} else if c != http.StatusConflict {
			t.Errorf("unexpected status %d", c)
		}
	}
	if created != 1 {
		t.Fatalf("%d setups succeeded, want exactly 1 (%v)", created, codes)
	}
}

// A cross-site POST carrying our cookie is blocked before it reaches any handler.
func TestCrossSiteWriteBlocked(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	r := owner.Do(http.MethodPost, "/setup", map[string]any{"username": "x", "password": "a-long-enough-password"},
		"Origin: https://evil.example")
	r.Expect(t, http.StatusForbidden)
	if r.Code() != "origin.mismatch" {
		t.Fatalf("code %q", r.Code())
	}
}
