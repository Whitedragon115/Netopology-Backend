package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/config"
)

// OAuthProfile contains only the provider attributes needed to identify or link an account.
type OAuthProfile struct {
	Subject       string
	Email         *string
	DisplayName   *string
	EmailVerified bool
}

// OAuthProvider keeps provider HTTP out of the service. Tests supply an httptest-backed provider.
type OAuthProvider interface {
	Authorize(state, verifier, redirectURI string) string
	Profile(ctx context.Context, code, verifier, redirectURI string) (OAuthProfile, error)
}

type remoteProvider struct {
	config     oauth2.Config
	profileURL string
	provider   string
	client     *http.Client
}

func defaultOAuthProviders(cfg *config.Config) map[string]OAuthProvider {
	providers := map[string]OAuthProvider{}
	client := &http.Client{Timeout: 10 * time.Second}
	if cfg.GoogleEnabled() {
		providers["google"] = &remoteProvider{
			provider: "google", client: client,
			config: oauth2.Config{
				ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret,
				Scopes: []string{"openid", "email", "profile"}, Endpoint: oauth2.Endpoint{ //nolint:gosec // Endpoint URLs are public, not credentials.
					AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token",
				},
			},
			profileURL: "https://openidconnect.googleapis.com/v1/userinfo",
		}
	}
	if cfg.DiscordEnabled() {
		providers["discord"] = &remoteProvider{
			provider: "discord", client: client,
			config: oauth2.Config{
				ClientID: cfg.DiscordClientID, ClientSecret: cfg.DiscordClientSecret,
				Scopes: []string{"identify", "email"}, Endpoint: oauth2.Endpoint{ //nolint:gosec // Endpoint URLs are public, not credentials.
					AuthURL: "https://discord.com/oauth2/authorize", TokenURL: "https://discord.com/api/oauth2/token",
				},
			},
			profileURL: "https://discord.com/api/v10/users/@me",
		}
	}
	return providers
}

func (p *remoteProvider) Authorize(state, verifier, redirectURI string) string {
	conf := p.config
	conf.RedirectURL = redirectURI
	return conf.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (p *remoteProvider) Profile(ctx context.Context, code, verifier, redirectURI string) (OAuthProfile, error) {
	conf := p.config
	conf.RedirectURL = redirectURI
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return OAuthProfile{}, fmt.Errorf("exchange OAuth code: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.profileURL, nil)
	if err != nil {
		return OAuthProfile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := p.client.Do(req)
	if err != nil {
		return OAuthProfile{}, fmt.Errorf("fetch OAuth profile: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return OAuthProfile{}, fmt.Errorf("OAuth profile returned status %d", response.StatusCode)
	}
	var raw struct {
		Sub           string  `json:"sub"`
		ID            string  `json:"id"`
		Email         *string `json:"email"`
		EmailVerified bool    `json:"email_verified"`
		Verified      bool    `json:"verified"`
		Name          *string `json:"name"`
		GlobalName    *string `json:"global_name"`
		Username      *string `json:"username"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&raw); err != nil {
		return OAuthProfile{}, fmt.Errorf("decode OAuth profile: %w", err)
	}
	profile := OAuthProfile{Email: raw.Email}
	if p.provider == "google" {
		profile.Subject, profile.EmailVerified, profile.DisplayName = raw.Sub, raw.EmailVerified, raw.Name
	} else {
		profile.Subject, profile.EmailVerified, profile.DisplayName = raw.ID, raw.Verified, raw.GlobalName
		if profile.DisplayName == nil {
			profile.DisplayName = raw.Username
		}
	}
	if profile.Subject == "" {
		return OAuthProfile{}, fmt.Errorf("OAuth profile has no subject")
	}
	return profile, nil
}
