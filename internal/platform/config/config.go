// Package config reads all settings from environment variables, once, at start-up.
//
// Rules (CONVENTIONS.md §3.2): environment only, parsed into this typed struct, passed down
// explicitly. No package reads os.Getenv anywhere else. Every variable is documented in
// .env.example and docs/ENVIRONMENT.md; keep the three in sync.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// Env is "development" or "production". Development relaxes nothing security-related;
	// it only switches log format to text and enables /api/v1/docs.
	Env      string `env:"NETOPOLOGY_ENV" envDefault:"production"`
	LogLevel string `env:"NETOPOLOGY_LOG_LEVEL" envDefault:"info"`

	// HTTP (api process)
	Addr string `env:"NETOPOLOGY_ADDR" envDefault:":8080"`
	// PublicURL is the URL people open in the browser, e.g. https://netopology.home.arpa.
	// Used for the OAuth redirect URIs, the Origin check and share-link URLs.
	PublicURL string `env:"NETOPOLOGY_PUBLIC_URL,required"`
	// CookieSecure must stay true outside plain-http local testing (browsers accept Secure
	// cookies on http://localhost anyway).
	CookieSecure bool `env:"NETOPOLOGY_COOKIE_SECURE" envDefault:"true"`
	// TrustedProxies lists CIDRs whose X-Forwarded-For is believed (the nginx gateway).
	TrustedProxies []string `env:"NETOPOLOGY_TRUSTED_PROXIES" envSeparator:"," envDefault:"172.16.0.0/12,10.0.0.0/8,192.168.0.0/16,127.0.0.1/32"`

	// Database. DatabaseURL is the NON-superuser app role (RLS applies to it).
	// MigrateDatabaseURL is the schema-owner role, used only by `netopology migrate`.
	DatabaseURL        string `env:"NETOPOLOGY_DATABASE_URL,required"`
	MigrateDatabaseURL string `env:"NETOPOLOGY_MIGRATE_DATABASE_URL"`
	DBMaxConns         int32  `env:"NETOPOLOGY_DB_MAX_CONNS" envDefault:"20"`
	// MigrateOnStart makes `api` run migrations (behind an advisory lock) before serving.
	MigrateOnStart bool `env:"NETOPOLOGY_MIGRATE_ON_START" envDefault:"true"`

	// SecretKey: 32 random bytes, base64. Signs cookies (OAuth state, share unlock) and, from C3,
	// encrypts plugin credentials (AES-256-GCM). Losing it invalidates those; rotating it is an ADR.
	SecretKeyB64 string `env:"NETOPOLOGY_SECRET_KEY,required"`
	SecretKey    []byte `env:"-"`

	// Sessions
	SessionTTL time.Duration `env:"NETOPOLOGY_SESSION_TTL" envDefault:"168h"`

	// OAuth providers: a provider is enabled when its client id and secret are both set.
	GoogleClientID      string `env:"NETOPOLOGY_GOOGLE_CLIENT_ID"`
	GoogleClientSecret  string `env:"NETOPOLOGY_GOOGLE_CLIENT_SECRET"`
	DiscordClientID     string `env:"NETOPOLOGY_DISCORD_CLIENT_ID"`
	DiscordClientSecret string `env:"NETOPOLOGY_DISCORD_CLIENT_SECRET"`

	// Worker
	WorkerConcurrency int `env:"NETOPOLOGY_WORKER_CONCURRENCY" envDefault:"10"`
}

// Load parses the environment and validates it. It returns every problem at once.
func Load() (*Config, error) {
	c, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}
	var errs []error
	if c.Env != "development" && c.Env != "production" {
		errs = append(errs, fmt.Errorf("NETOPOLOGY_ENV must be development or production, got %q", c.Env))
	}
	u, err := url.Parse(c.PublicURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, fmt.Errorf("NETOPOLOGY_PUBLIC_URL must be an absolute http(s) URL, got %q", c.PublicURL))
	}
	c.PublicURL = strings.TrimRight(c.PublicURL, "/")
	key, err := base64.StdEncoding.DecodeString(c.SecretKeyB64)
	if err != nil || len(key) != 32 {
		errs = append(errs, errors.New("NETOPOLOGY_SECRET_KEY must be 32 random bytes, base64 (openssl rand -base64 32)"))
	}
	c.SecretKey = key
	if c.MigrateOnStart && c.MigrateDatabaseURL == "" {
		errs = append(errs, errors.New("NETOPOLOGY_MIGRATE_ON_START needs NETOPOLOGY_MIGRATE_DATABASE_URL (the schema-owner role)"))
	}
	if c.SessionTTL < time.Hour {
		errs = append(errs, errors.New("NETOPOLOGY_SESSION_TTL must be at least 1h"))
	}
	if (c.GoogleClientID == "") != (c.GoogleClientSecret == "") {
		errs = append(errs, errors.New("set both NETOPOLOGY_GOOGLE_CLIENT_ID and NETOPOLOGY_GOOGLE_CLIENT_SECRET, or neither"))
	}
	if (c.DiscordClientID == "") != (c.DiscordClientSecret == "") {
		errs = append(errs, errors.New("set both NETOPOLOGY_DISCORD_CLIENT_ID and NETOPOLOGY_DISCORD_CLIENT_SECRET, or neither"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) Dev() bool { return c.Env == "development" }

// GoogleEnabled and DiscordEnabled tell auth.providers which buttons to show.
func (c *Config) GoogleEnabled() bool  { return c.GoogleClientID != "" }
func (c *Config) DiscordEnabled() bool { return c.DiscordClientID != "" }
