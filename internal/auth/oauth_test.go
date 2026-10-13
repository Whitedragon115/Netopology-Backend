package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Whitedragon115/Netopology-Backend/internal/auth"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

type fakeOAuthProvider struct {
	server    *httptest.Server
	mu        sync.Mutex
	verifiers map[string]bool
	profiles  map[string]auth.OAuthProfile
}

func fakeProvider(t *testing.T, profiles map[string]auth.OAuthProfile) *fakeOAuthProvider {
	t.Helper()
	f := &fakeOAuthProvider{verifiers: map[string]bool{}, profiles: profiles}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/profile" {
			http.NotFound(w, r)
			return
		}
		profile, ok := f.profiles[r.URL.Query().Get("code")]
		if !ok {
			http.Error(w, "missing profile", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(profile)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeOAuthProvider) Authorize(state, verifier, redirectURI string) string {
	f.mu.Lock()
	f.verifiers[verifier] = true
	f.mu.Unlock()
	values := url.Values{"state": {state}, "redirect_uri": {redirectURI}, "code_challenge_method": {"S256"}}
	return f.server.URL + "/authorize?" + values.Encode()
}

func (f *fakeOAuthProvider) Profile(ctx context.Context, code, verifier, _ string) (auth.OAuthProfile, error) {
	f.mu.Lock()
	known := f.verifiers[verifier]
	f.mu.Unlock()
	if !known {
		return auth.OAuthProfile{}, fmt.Errorf("wrong PKCE verifier")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.server.URL+"/profile?code="+url.QueryEscape(code), nil)
	if err != nil {
		return auth.OAuthProfile{}, err
	}
	res, err := f.server.Client().Do(req)
	if err != nil {
		return auth.OAuthProfile{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return auth.OAuthProfile{}, fmt.Errorf("profile status %d", res.StatusCode)
	}
	var profile auth.OAuthProfile
	if err := json.NewDecoder(res.Body).Decode(&profile); err != nil {
		return auth.OAuthProfile{}, err
	}
	return profile, nil
}

func ptr(s string) *string { return &s }

func startOAuth(t *testing.T, client *testutil.Client, path string) string {
	t.Helper()
	res := client.Do(http.MethodGet, path, nil).Expect(t, 302)
	if len(res.Cookies()) != 1 || res.Cookies()[0].Name != "netopology_oauth" {
		t.Fatalf("OAuth cookie: %v", res.Cookies())
	}
	location, err := url.Parse(res.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := location.Query().Get("state")
	if state == "" || location.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("OAuth URL %s", location)
	}
	return state
}

func finishOAuth(t *testing.T, client *testutil.Client, provider, code, state string) *testutil.Response {
	t.Helper()
	path := "/auth/oauth/" + provider + "/callback?code=" + url.QueryEscape(code) + "&state=" + url.QueryEscape(state)
	return client.Do(http.MethodGet, path, nil).Expect(t, 302)
}

func TestOAuthLinkedLoginAndReturnTo(t *testing.T) {
	provider := fakeProvider(t, map[string]auth.OAuthProfile{
		"linked": {Subject: "linked-sub", Email: ptr("owner@example.test"), EmailVerified: true},
	})
	api := testutil.NewAPIWithOAuth(t, map[string]auth.OAuthProvider{"google": provider})
	owner := api.Owner()
	owner.Do(http.MethodPatch, "/me", map[string]any{"email": "owner@example.test"}).Expect(t, 200)
	conn := ownerDB(t, api)
	defer conn.Close(context.Background())
	if _, err := conn.Exec(t.Context(), `INSERT INTO user_identities(user_id,provider,subject)
		SELECT id,'google','linked-sub' FROM users WHERE username='owner'`); err != nil {
		t.Fatal(err)
	}
	client := api.Client()
	state := startOAuth(t, client, "/auth/oauth/google/start?returnTo=%2Fcanvas")
	res := finishOAuth(t, client, "google", "linked", state)
	if res.Header.Get("Location") != "http://netopology.test/canvas" {
		t.Fatalf("redirect %q", res.Header.Get("Location"))
	}
	if len(res.Cookies()) != 2 {
		t.Fatalf("expected cleared OAuth and session cookies, got %v", res.Cookies())
	}
	client.Do(http.MethodGet, "/me", nil).Expect(t, 200)
	var count int
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE action='auth.oauth_callback'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("OAuth audit count %d", count)
	}
}

func TestOAuthAutoLinkAndUnverifiedRefusal(t *testing.T) {
	provider := fakeProvider(t, map[string]auth.OAuthProfile{
		"verified":   {Subject: "new-sub", Email: ptr("new@example.test"), EmailVerified: true},
		"unverified": {Subject: "bad-sub", Email: ptr("other@example.test"), EmailVerified: false},
		"unknown":    {Subject: "unknown-sub", Email: ptr("absent@example.test"), EmailVerified: true},
	})
	api := testutil.NewAPIWithOAuth(t, map[string]auth.OAuthProvider{"google": provider})
	owner := api.Owner()
	owner.Do(http.MethodPost, "/users", map[string]any{"username": "newuser", "email": "new@example.test"}).Expect(t, 201)
	owner.Do(http.MethodPost, "/users", map[string]any{"username": "other", "email": "other@example.test"}).Expect(t, 201)
	for code, want := range map[string]string{"unverified": "identity_not_linked", "unknown": "identity_not_linked", "verified": ""} {
		client := api.Client()
		state := startOAuth(t, client, "/auth/oauth/google/start")
		res := finishOAuth(t, client, "google", code, state)
		if !strings.Contains(res.Header.Get("Location"), want) {
			t.Fatalf("%s redirect %s", code, res.Header.Get("Location"))
		}
		if want == "" {
			client.Do(http.MethodGet, "/me", nil).Expect(t, 200)
		} else {
			client.Do(http.MethodGet, "/me", nil).Expect(t, 401)
		}
	}
	conn := ownerDB(t, api)
	defer conn.Close(context.Background())
	var users, identities int
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM user_identities WHERE provider='google'`).Scan(&identities); err != nil {
		t.Fatal(err)
	}
	if users != 3 || identities != 1 {
		t.Fatalf("auto-link users=%d identities=%d", users, identities)
	}
}

func TestOAuthLinkStateAndRedirectSafety(t *testing.T) {
	provider := fakeProvider(t, map[string]auth.OAuthProfile{
		"link": {Subject: "same-sub", Email: ptr("linked@example.test"), EmailVerified: true},
	})
	api := testutil.NewAPIWithOAuth(t, map[string]auth.OAuthProvider{"google": provider})
	owner := api.Owner()
	member := api.CreateUser("member", "member")
	api.Client().Do(http.MethodGet, "/me/identities/google/link", nil).Expect(t, 401)
	api.Client().Do(http.MethodGet, "/auth/oauth/discord/start", nil).Expect(t, 404)

	state := startOAuth(t, owner, "/me/identities/google/link")
	res := finishOAuth(t, owner, "google", "link", state)
	if res.Header.Get("Location") != "http://netopology.test/" {
		t.Fatalf("link redirect %s", res.Header.Get("Location"))
	}
	var list struct {
		Items       []struct{ Provider string }
		HasPassword bool
	}
	owner.Do(http.MethodGet, "/me/identities", nil).Expect(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Provider != "google" {
		t.Fatalf("linked identities %+v", list)
	}
	owner.Do(http.MethodGet, "/me/identities/google/link", nil).Expect(t, 409)

	state = startOAuth(t, member, "/me/identities/google/link")
	res = finishOAuth(t, member, "google", "link", state)
	if !strings.Contains(res.Header.Get("Location"), "error=identity_in_use") {
		t.Fatalf("other account link %s", res.Header.Get("Location"))
	}

	for _, returnTo := range []string{"https://evil.example/", "//evil.example/path", "/%5Cevil.example"} {
		client := api.Client()
		state = startOAuth(t, client, "/auth/oauth/google/start?returnTo="+url.QueryEscape(returnTo))
		res = finishOAuth(t, client, "google", "link", state)
		if res.Header.Get("Location") != "http://netopology.test/" {
			t.Fatalf("unsafe returnTo %q -> %q", returnTo, res.Header.Get("Location"))
		}
	}

	client := api.Client()
	state = startOAuth(t, client, "/auth/oauth/google/start")
	res = finishOAuth(t, client, "google", "link", "wrong-state")
	if !strings.Contains(res.Header.Get("Location"), "error=invalid_state") {
		t.Fatalf("wrong state %s", res.Header.Get("Location"))
	}
	res = api.Client().Do(http.MethodGet,
		"/auth/oauth/google/callback?code=link&state="+url.QueryEscape(state), nil,
		"Cookie: netopology_oauth=tampered").Expect(t, 302)
	if !strings.Contains(res.Header.Get("Location"), "error=invalid_state") {
		t.Fatalf("tampered cookie %s", res.Header.Get("Location"))
	}
	proof, _ := json.Marshal(map[string]any{"state": state, "verifier": "test", "provider": "google", "mode": "login", "returnTo": "/"})
	expired := crypto.NewSigner(api.Config.SecretKey, "oauth").Sign(proof, -time.Minute)
	res = api.Client().Do(http.MethodGet,
		"/auth/oauth/google/callback?code=link&state="+url.QueryEscape(state), nil,
		"Cookie: netopology_oauth="+expired).Expect(t, 302)
	if !strings.Contains(res.Header.Get("Location"), "error=invalid_state") {
		t.Fatalf("expired cookie %s", res.Header.Get("Location"))
	}

	client = api.Client()
	state = startOAuth(t, client, "/auth/oauth/google/start")
	res = client.Do(http.MethodGet, "/auth/oauth/google/callback?error=access_denied&state="+url.QueryEscape(state), nil).Expect(t, 302)
	if !strings.Contains(res.Header.Get("Location"), "error=access_denied") {
		t.Fatalf("cancelled OAuth %s", res.Header.Get("Location"))
	}
	conn := ownerDB(t, api)
	defer conn.Close(context.Background())
	var count int
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE action='me.identity_link'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("link audit count %d", count)
	}
}

func TestOAuthAutoLinkDisabled(t *testing.T) {
	provider := fakeProvider(t, map[string]auth.OAuthProfile{
		"verified": {Subject: "unused", Email: ptr("ready@example.test"), EmailVerified: true},
	})
	api := testutil.NewAPIWithOAuth(t, map[string]auth.OAuthProvider{"google": provider})
	owner := api.Owner()
	owner.Do(http.MethodPost, "/users", map[string]any{"username": "ready", "email": "ready@example.test"}).Expect(t, 201)
	owner.Do(http.MethodPatch, "/instance/settings", map[string]any{"oauthAutoLinkByEmail": false}).Expect(t, 200)
	client := api.Client()
	state := startOAuth(t, client, "/auth/oauth/google/start")
	res := finishOAuth(t, client, "google", "verified", state)
	if !strings.Contains(res.Header.Get("Location"), "error=identity_not_linked") {
		t.Fatalf("auto-link disabled %s", res.Header.Get("Location"))
	}
}
