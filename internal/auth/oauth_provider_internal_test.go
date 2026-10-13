package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestRemoteOAuthProviderPKCEAndProfiles(t *testing.T) {
	for _, tc := range []struct {
		name, body, subject string
		verified            bool
	}{
		{"google", `{"sub":"google-123","email":"g@example.test","email_verified":true,"name":"Google User"}`, "google-123", true},
		{"discord", `{"id":"discord-456","email":"d@example.test","verified":true,"global_name":"Discord User"}`, "discord-456", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifier := oauth2.GenerateVerifier()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/token":
					if err := r.ParseForm(); err != nil {
						t.Error(err)
					}
					if r.Form.Get("code_verifier") != verifier || r.Form.Get("code") != "code" {
						t.Errorf("token exchange missing verifier or code: %v", r.Form)
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "token_type": "Bearer"})
				case "/profile":
					if r.Header.Get("Authorization") != "Bearer test-token" {
						t.Errorf("profile missing bearer auth")
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tc.body))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			provider := &remoteProvider{
				provider: tc.name, client: server.Client(), profileURL: server.URL + "/profile",
				config: oauth2.Config{ClientID: "client", ClientSecret: "secret", Endpoint: oauth2.Endpoint{
					AuthURL: server.URL + "/authorize", TokenURL: server.URL + "/token",
				}},
			}
			authURL, err := url.Parse(provider.Authorize("state", verifier, "http://app.test/callback"))
			if err != nil {
				t.Fatal(err)
			}
			if authURL.Query().Get("code_challenge") != oauth2.S256ChallengeFromVerifier(verifier) ||
				authURL.Query().Get("code_challenge_method") != "S256" {
				t.Fatalf("bad PKCE URL: %s", authURL)
			}
			profile, err := provider.Profile(context.Background(), "code", verifier, "http://app.test/callback")
			if err != nil {
				t.Fatal(err)
			}
			if profile.Subject != tc.subject || profile.EmailVerified != tc.verified {
				t.Fatalf("profile %+v", profile)
			}
		})
	}
}
