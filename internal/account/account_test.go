package account_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestGetMeLinkedProvidersAndNoSecret(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	db, err := pgx.Connect(context.Background(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	if _, err := db.Exec(context.Background(), `INSERT INTO user_identities (user_id, provider, subject) SELECT id, 'google', 'test-subject' FROM users WHERE username = 'owner'`); err != nil {
		t.Fatal(err)
	}
	r := owner.Do(http.MethodGet, "/me", nil).Expect(t, http.StatusOK)
	var me struct {
		Username   string
		Identities []string
	}
	r.JSON(t, &me)
	if me.Username != "owner" || len(me.Identities) != 1 || me.Identities[0] != "google" {
		t.Fatalf("unexpected account: %+v", me)
	}
	if strings.Contains(string(r.Body), "passwordHash") || strings.Contains(string(r.Body), "owner-password-123") {
		t.Fatal("getMe exposed a password or hash")
	}
}
