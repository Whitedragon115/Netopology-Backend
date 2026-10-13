package account_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func ownerDB(t *testing.T, api *testutil.API) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func accountID(t *testing.T, body []byte) uuid.UUID {
	t.Helper()
	var account struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(body, &account); err != nil {
		t.Fatal(err)
	}
	if account.ID == uuid.Nil {
		t.Fatalf("missing account id: %s", body)
	}
	return account.ID
}

func TestOwnProfilePasswordAndIdentities(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	owner.Do(http.MethodPatch, "/me", map[string]any{"displayName": "Owner", "email": "owner@example.test"}).Expect(t, 200)
	var me struct {
		DisplayName string
		Email       *string
	}
	owner.Do(http.MethodGet, "/me", nil).Expect(t, 200).JSON(t, &me)
	if me.DisplayName != "Owner" || me.Email == nil || *me.Email != "owner@example.test" {
		t.Fatalf("profile %+v", me)
	}
	owner.Do(http.MethodPatch, "/me", map[string]any{"email": nil}).Expect(t, 200).JSON(t, &me)
	if me.Email != nil {
		t.Fatalf("email was not cleared: %+v", me)
	}
	owner.Do(http.MethodPatch, "/me", map[string]any{}).Expect(t, 422)

	var identities struct {
		Items       []any
		HasPassword bool
	}
	owner.Do(http.MethodGet, "/me/identities", nil).Expect(t, 200).JSON(t, &identities)
	if !identities.HasPassword || len(identities.Items) != 0 {
		t.Fatalf("identities %+v", identities)
	}

	other := api.Login("owner", "owner-password-123")
	owner.Do(http.MethodPost, "/me/password", map[string]any{"currentPassword": "wrong", "newPassword": "new-owner-password-123"}).Expect(t, 401)
	owner.Do(http.MethodPost, "/me/password", map[string]any{"currentPassword": "owner-password-123", "newPassword": "new-owner-password-123"}).Expect(t, 204)
	other.Do(http.MethodGet, "/me", nil).Expect(t, 401)
	owner.Do(http.MethodGet, "/me", nil).Expect(t, 200)
	api.Login("owner", "new-owner-password-123")

	conn := ownerDB(t, api)
	for action, want := range map[string]int{"me.update": 2, "me.password": 1} {
		var count int
		if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE action=$1`, action).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Errorf("audit %s: %d, want %d", action, count, want)
		}
	}
}

func TestUserAdministrationAndSettings(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	created := owner.Do(http.MethodPost, "/users", map[string]any{"username": "alice", "password": "alice-password-123", "email": "alice@example.test"}).Expect(t, 201)
	id := accountID(t, created.Body)
	if strings.Contains(string(created.Body), "passwordHash") {
		t.Fatal("password hash exposed")
	}
	owner.Do(http.MethodPost, "/users", map[string]any{"username": "alice", "password": "alice-password-123"}).Expect(t, 409)
	owner.Do(http.MethodPost, "/users", map[string]any{"username": "emailonly", "email": "emailonly@example.test"}).Expect(t, 201)
	owner.Do(http.MethodPost, "/users", map[string]any{"username": "neither"}).Expect(t, 422)

	var page struct {
		Items      []struct{ Username string }
		NextCursor *string
	}
	owner.Do(http.MethodGet, "/users?limit=1", nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("page %+v", page)
	}
	owner.Do(http.MethodGet, "/users?limit=1&cursor="+*page.NextCursor, nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 {
		t.Fatalf("second page %+v", page)
	}
	owner.Do(http.MethodGet, "/users?cursor=bad", nil).Expect(t, 422)
	owner.Do(http.MethodGet, "/users?q=alice", nil).Expect(t, 200).JSON(t, &page)
	if len(page.Items) != 1 || page.Items[0].Username != "alice" {
		t.Fatalf("filter %+v", page)
	}
	owner.Do(http.MethodGet, "/users/"+id.String(), nil).Expect(t, 200)
	owner.Do(http.MethodGet, "/users/lookup?username=ALICE", nil).Expect(t, 200)
	owner.Do(http.MethodGet, "/users/lookup?username=absent", nil).Expect(t, 404)

	alice := api.Login("alice", "alice-password-123")
	owner.Do(http.MethodPatch, "/users/"+id.String(), map[string]any{"displayName": "Alice", "email": nil}).Expect(t, 200)
	owner.Do(http.MethodPost, "/users/"+id.String()+"/password", map[string]any{"newPassword": "new-alice-password-123"}).Expect(t, 204)
	alice.Do(http.MethodGet, "/me", nil).Expect(t, 401)
	api.Login("alice", "new-alice-password-123")
	owner.Do(http.MethodPatch, "/users/"+id.String(), map[string]any{"status": "disabled"}).Expect(t, 200)
	api.Client().Do(http.MethodPost, "/auth/login", map[string]any{"username": "alice", "password": "new-alice-password-123"}).Expect(t, 401)

	var settings struct {
		SessionTtlHours      int
		OauthAutoLinkByEmail bool
	}
	owner.Do(http.MethodGet, "/instance/settings", nil).Expect(t, 200).JSON(t, &settings)
	if settings.SessionTtlHours != 168 || !settings.OauthAutoLinkByEmail {
		t.Fatalf("defaults %+v", settings)
	}
	owner.Do(http.MethodPatch, "/instance/settings", map[string]any{"sessionTtlHours": 48, "oauthAutoLinkByEmail": false}).Expect(t, 200).JSON(t, &settings)
	if settings.SessionTtlHours != 48 || settings.OauthAutoLinkByEmail {
		t.Fatalf("settings %+v", settings)
	}
	owner.Do(http.MethodPatch, "/instance/settings", map[string]any{"sessionTtlHours": 0}).Expect(t, 422)
	api.Login("owner", "owner-password-123")
	conn := ownerDB(t, api)
	var duration time.Duration
	if err := conn.QueryRow(t.Context(), `SELECT expires_at-created_at FROM sessions ORDER BY created_at DESC LIMIT 1`).Scan(&duration); err != nil {
		t.Fatal(err)
	}
	if duration < 47*time.Hour || duration > 49*time.Hour {
		t.Fatalf("session lifetime %s", duration)
	}
	for _, action := range []string{"user.create", "user.update", "user.password_reset", "instance.settings_update"} {
		var count int
		if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE action=$1`, action).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == 0 {
			t.Errorf("no audit entry for %s", action)
		}
	}
}

func TestProviderOnlyPasswordAndLastIdentity(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	id := accountID(t, owner.Do(http.MethodPost, "/users", map[string]any{
		"username": "provider", "email": "provider@example.test",
	}).Expect(t, 201).Body)
	conn := ownerDB(t, api)
	var identityID uuid.UUID
	if err := conn.QueryRow(t.Context(), `INSERT INTO user_identities (user_id, provider, subject, last_used_at)
		VALUES ($1, 'google', 'provider-subject', now()) RETURNING id`, id).Scan(&identityID); err != nil {
		t.Fatal(err)
	}
	token := "provider-test-session-token"
	if _, err := conn.Exec(t.Context(), `INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, now()+interval '1 day')`, id, crypto.HashToken(token)); err != nil {
		t.Fatal(err)
	}
	client := api.Client()
	cookie := "Cookie: netopology_session=" + token
	client.Do(http.MethodDelete, "/me/identities/"+identityID.String(), nil, cookie).Expect(t, 409)
	client.Do(http.MethodPost, "/me/password", map[string]any{
		"currentPassword": "", "newPassword": "provider-password-123",
	}, cookie).Expect(t, 403)
	if _, err := conn.Exec(t.Context(), `UPDATE user_identities SET last_used_at=now() WHERE id=$1`, identityID); err != nil {
		t.Fatal(err)
	}
	client.Do(http.MethodPost, "/me/password", map[string]any{
		"currentPassword": "", "newPassword": "provider-password-123",
	}, cookie).Expect(t, 204)
	client.Do(http.MethodDelete, "/me/identities/"+identityID.String(), nil, cookie).Expect(t, 204)
	api.Login("provider", "provider-password-123")
}
