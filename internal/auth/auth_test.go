package auth_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestLoginMeLogoutAndAudit(t *testing.T) {
	api := testutil.NewAPI(t)
	api.Owner()
	client := api.Client()
	loginResponse := client.Do(http.MethodPost, "/auth/login", map[string]any{
		"username": "owner", "password": "owner-password-123",
	}).Expect(t, http.StatusOK)
	var me struct {
		ID, Username, InstanceRole, Status string
		HasPassword                        bool
		Identities                         []string
	}
	loginResponse.JSON(t, &me)
	if me.ID == "" || me.Username != "owner" || me.Identities == nil {
		t.Fatalf("unexpected login body: %+v", me)
	}
	if len(loginResponse.Cookies()) != 1 || loginResponse.Cookies()[0].MaxAge != 0 {
		t.Fatalf("default login must use a browser-session cookie: %v", loginResponse.Cookies())
	}
	client.Do(http.MethodGet, "/me", nil).Expect(t, http.StatusOK).JSON(t, &me)
	if me.ID == "" || me.Username != "owner" || me.InstanceRole != "owner" || me.Status != "active" || !me.HasPassword || me.Identities == nil {
		t.Fatalf("unexpected account: %+v", me)
	}
	res := client.Do(http.MethodPost, "/auth/logout", nil).Expect(t, http.StatusNoContent)
	if len(res.Body) != 0 || len(res.Cookies()) == 0 || res.Cookies()[len(res.Cookies())-1].MaxAge >= 0 {
		t.Fatalf("logout did not clear cookie: %q, %v", res.Body, res.Cookies())
	}
	client.Do(http.MethodGet, "/me", nil).Expect(t, http.StatusUnauthorized)

	db := ownerDB(t, api)
	defer db.Close(context.Background())
	var login, logout int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE action = 'auth.login'), count(*) FILTER (WHERE action = 'auth.logout') FROM audit_log`).Scan(&login, &logout); err != nil {
		t.Fatal(err)
	}
	if login != 1 || logout != 1 {
		t.Fatalf("audit login=%d logout=%d", login, logout)
	}
}

func TestLoginFailuresAndLimits(t *testing.T) {
	api := testutil.NewAPI(t)
	api.Owner()
	client := api.Client()
	for _, tc := range []struct{ username, password string }{
		{"missing", "owner-password-123"},
		{"owner", "wrong"},
	} {
		r := client.Do(http.MethodPost, "/auth/login", map[string]any{"username": tc.username, "password": tc.password}).Expect(t, http.StatusUnauthorized)
		if r.Code() != "auth.invalid_credentials" {
			t.Fatalf("%s: code %q", tc.username, r.Code())
		}
	}
	db := ownerDB(t, api)
	defer db.Close(context.Background())
	if _, err := db.Exec(context.Background(), `UPDATE users SET status = 'disabled' WHERE username = 'owner'`); err != nil {
		t.Fatal(err)
	}
	r := client.Do(http.MethodPost, "/auth/login", map[string]any{"username": "owner", "password": "owner-password-123"}).Expect(t, http.StatusUnauthorized)
	if r.Code() != "auth.invalid_credentials" {
		t.Fatalf("disabled account code %q", r.Code())
	}
	for range 8 { // ten attempts for this username, then 429
		client.Do(http.MethodPost, "/auth/login", map[string]any{"username": "OWNER", "password": "wrong"}).Expect(t, http.StatusUnauthorized)
	}
	r = client.Do(http.MethodPost, "/auth/login", map[string]any{"username": "owner", "password": "wrong"}).Expect(t, http.StatusTooManyRequests)
	if r.Code() != "rate_limited" || r.Header.Get("Retry-After") == "" {
		t.Fatalf("rate limit code=%q retry=%q", r.Code(), r.Header.Get("Retry-After"))
	}
	var failures int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM audit_log WHERE action = 'auth.login' AND metadata->>'reason' = 'invalid_credentials'`).Scan(&failures); err != nil {
		t.Fatal(err)
	}
	if failures != 11 {
		t.Fatalf("failure audit count %d, want 11", failures)
	}
}

func TestRememberCookieAndSlidingExpiry(t *testing.T) {
	api := testutil.NewAPI(t)
	api.Owner()
	client := api.Client()
	r := client.Do(http.MethodPost, "/auth/login", map[string]any{
		"username": "owner", "password": "owner-password-123", "remember": true,
	}).Expect(t, http.StatusOK)
	if len(r.Cookies()) != 1 || r.Cookies()[0].MaxAge != 30*24*60*60 {
		t.Fatalf("remembered cookie: %v", r.Cookies())
	}
	db := ownerDB(t, api)
	defer db.Close(context.Background())
	var remember bool
	var expires time.Time
	if err := db.QueryRow(context.Background(), `SELECT remember, expires_at FROM sessions WHERE user_id = (SELECT id FROM users WHERE username = 'owner') AND remember`).Scan(&remember, &expires); err != nil {
		t.Fatal(err)
	}
	if !remember || time.Until(expires) < 29*24*time.Hour {
		t.Fatalf("remember=%v expiry=%v", remember, expires)
	}
	if _, err := db.Exec(context.Background(), `UPDATE sessions SET last_seen_at = now() - interval '2 minutes', expires_at = now() + interval '1 hour' WHERE remember`); err != nil {
		t.Fatal(err)
	}
	r = client.Do(http.MethodGet, "/me", nil).Expect(t, http.StatusOK)
	if len(r.Cookies()) == 0 || r.Cookies()[0].MaxAge != 30*24*60*60 {
		t.Fatalf("sliding cookie: %v", r.Cookies())
	}
	if err := db.QueryRow(context.Background(), `SELECT expires_at FROM sessions WHERE remember`).Scan(&expires); err != nil {
		t.Fatal(err)
	}
	if time.Until(expires) < 29*24*time.Hour {
		t.Fatalf("remembered expiry did not slide: %v", expires)
	}

	plain := api.Client()
	r = plain.Do(http.MethodPost, "/auth/login", map[string]any{
		"username": "owner", "password": "owner-password-123",
	}).Expect(t, http.StatusOK)
	if len(r.Cookies()) != 1 || r.Cookies()[0].MaxAge != 0 || !r.Cookies()[0].Expires.IsZero() {
		t.Fatalf("plain login must use a browser-session cookie: %v", r.Cookies())
	}
	if err := db.QueryRow(context.Background(), `SELECT expires_at FROM sessions WHERE NOT remember ORDER BY created_at DESC LIMIT 1`).Scan(&expires); err != nil {
		t.Fatal(err)
	}
	if remaining := time.Until(expires); remaining < 23*time.Hour || remaining > 25*time.Hour {
		t.Fatalf("plain session expiry %v, want about 24h", remaining)
	}
}

func TestProviders(t *testing.T) {
	api := testutil.NewAPI(t)
	var body struct {
		Password  bool
		Providers []struct{ ID, Name string }
	}
	api.Client().Do(http.MethodGet, "/auth/providers", nil).Expect(t, http.StatusOK).JSON(t, &body)
	if !body.Password || len(body.Providers) != 0 {
		t.Fatalf("default providers: %+v", body)
	}
	api.Config.GoogleClientID, api.Config.GoogleClientSecret = "id", "secret"
	api.Config.DiscordClientID, api.Config.DiscordClientSecret = "id", "secret"
	api.Client().Do(http.MethodGet, "/auth/providers", nil).Expect(t, http.StatusOK).JSON(t, &body)
	if len(body.Providers) != 2 || body.Providers[0].ID != "google" || body.Providers[1].ID != "discord" {
		t.Fatalf("configured providers: %+v", body)
	}
}

func TestRehashOnLogin(t *testing.T) {
	api := testutil.NewAPI(t)
	api.Owner()
	db := ownerDB(t, api)
	defer db.Close(context.Background())
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte("owner-password-123"), salt, 2, 8*1024, 1, 32)
	old := fmt.Sprintf("$argon2id$v=19$m=8192,t=2,p=1$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
	if _, err := db.Exec(context.Background(), `UPDATE users SET password_hash = $1 WHERE username = 'owner'`, old); err != nil {
		t.Fatal(err)
	}
	api.Login("owner", "owner-password-123")
	var updated string
	if err := db.QueryRow(context.Background(), `SELECT password_hash FROM users WHERE username = 'owner'`).Scan(&updated); err != nil {
		t.Fatal(err)
	}
	if updated == old || crypto.NeedsRehash(updated) {
		t.Fatal("password hash was not upgraded")
	}
}

func ownerDB(t *testing.T, api *testutil.API) *pgx.Conn {
	t.Helper()
	db, err := pgx.Connect(context.Background(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
