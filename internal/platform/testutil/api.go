package testutil

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/auth"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/config"
	"github.com/Whitedragon115/Netopology-Backend/internal/server"
)

// API is a running api server on a fresh database.
type API struct {
	t      testing.TB
	DB     *DB
	Server *httptest.Server
	Config *config.Config
}

// NewAPI starts the real HTTP stack (all middleware, all features) against Postgres(t).
func NewAPI(t testing.TB) *API {
	return newAPI(t, nil)
}

// NewAPIWithOAuth uses the real API and database with fake provider HTTP supplied by the test.
func NewAPIWithOAuth(t testing.TB, providers map[string]auth.OAuthProvider) *API {
	return newAPI(t, providers)
}

func newAPI(t testing.TB, providers map[string]auth.OAuthProvider) *API {
	t.Helper()
	d := Postgres(t)
	cfg := &config.Config{
		Env:            "development",
		PublicURL:      "http://netopology.test",
		CookieSecure:   false,
		TrustedProxies: []string{"127.0.0.1/32"},
		DatabaseURL:    d.AppURL,
		SecretKey:      []byte("0123456789abcdef0123456789abcdef"),
		SecretKeyB64:   base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		SessionTTL:     24 * time.Hour,
	}
	if providers["google"] != nil {
		cfg.GoogleClientID, cfg.GoogleClientSecret = "test-google", "test-secret"
	}
	if providers["discord"] != nil {
		cfg.DiscordClientID, cfg.DiscordClientSecret = "test-discord", "test-secret"
	}
	log := slog.New(slog.DiscardHandler)
	if testing.Verbose() {
		log = slog.New(slog.NewTextHandler(testWriter{t}, nil))
	}
	h, err := server.New(server.Deps{Config: cfg, DB: d.DB, Log: log, OAuthProviders: providers})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &API{t: t, DB: d, Server: srv, Config: cfg}
}

// Client is one browser: it keeps cookies and sends the same-origin header like a browser would.
type Client struct {
	t    testing.TB
	api  *API
	http *http.Client
}

// Client returns a new anonymous client (its own cookie jar).
func (a *API) Client() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{t: a.t, api: a, http: &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Response is a finished request with its body read.
type Response struct {
	*http.Response
	Body []byte
}

// JSON decodes the body into v (fails the test on error).
func (r *Response) JSON(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v\n%s", r.Request.URL.Path, err, r.Body)
	}
}

// Code returns the problem+json `code` ("" if the body is not a problem).
func (r *Response) Code() string {
	var p struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(r.Body, &p)
	return p.Code
}

// Do sends a request to /api/v1+path. body may be nil, a string (raw JSON) or any value (marshaled).
// headers are "Name: value" strings, e.g. `If-Match: "3"`.
func (c *Client) Do(method, path string, body any, headers ...string) *Response {
	c.t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rd = strings.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			c.t.Fatal(err)
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(c.t.Context(), method, c.api.Server.URL+"/api/v1"+path, rd)
	if err != nil {
		c.t.Fatal(err)
	}
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", c.api.Config.PublicURL) // a browser on the real site
	for _, h := range headers {
		k, v, _ := strings.Cut(h, ":")
		req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return &Response{Response: res, Body: b}
}

// Expect fails the test unless the status matches; returns r for chaining.
func (r *Response) Expect(t testing.TB, status int) *Response {
	t.Helper()
	if r.StatusCode != status {
		t.Fatalf("%s %s: status %d, want %d\n%s", r.Request.Method, r.Request.URL.Path, r.StatusCode, status, r.Body)
	}
	return r
}

// Owner completes first-run setup and returns a client signed in as the server owner, plus the
// id of the first workspace ("Home lab", slug "home-lab"). Most tests start here.
// TODO(WP-04+): add helpers as features land: CreateUser, AddMember(ws, user, role),
// SignedInAs(role) returning a client per workspace role, SeedInventory(ws) ...
func (a *API) Owner() *Client {
	c := a.Client()
	c.Do(http.MethodPost, "/setup", map[string]any{
		"username": "owner", "password": "owner-password-123", "workspaceName": "Home lab",
	}).Expect(a.t, http.StatusCreated)
	return c
}

// Login returns a separate browser signed in through the password endpoint.
func (a *API) Login(username, password string) *Client {
	c := a.Client()
	c.Do(http.MethodPost, "/auth/login", map[string]any{
		"username": username, "password": password,
	}).Expect(a.t, http.StatusOK)
	return c
}

// CreateUser adds an account through the API and returns its own signed-in browser.
// Call Owner first so the server owner exists.
func (a *API) CreateUser(username, role string) *Client {
	const password = "fixture-password-123"
	a.Login("owner", "owner-password-123").Do(http.MethodPost, "/users", map[string]any{
		"username": username, "password": password, "instanceRole": role,
	}).Expect(a.t, http.StatusCreated)
	return a.Login(username, password)
}

// AddMember adds an existing account through the owner-facing API.
func (a *API) AddMember(wsID, userID uuid.UUID, role string) {
	a.Login("owner", "owner-password-123").Do(http.MethodPost,
		"/workspaces/"+wsID.String()+"/members", map[string]any{"userId": userID, "role": role}).Expect(a.t, http.StatusCreated)
}

// ClientAs creates a signed-in member of the first workspace. Call Owner first.
func (a *API) ClientAs(role string) *Client {
	var workspaces struct{ Items []struct{ ID uuid.UUID } }
	a.Login("owner", "owner-password-123").Do(http.MethodGet, "/workspaces", nil).
		Expect(a.t, http.StatusOK).JSON(a.t, &workspaces)
	if len(workspaces.Items) == 0 {
		a.t.Fatal("ClientAs requires a workspace")
	}
	username := "fixture-" + uuid.NewString()[:8]
	client := a.CreateUser(username, "member")
	var me struct{ ID uuid.UUID }
	client.Do(http.MethodGet, "/me", nil).Expect(a.t, http.StatusOK).JSON(a.t, &me)
	a.AddMember(workspaces.Items[0].ID, me.ID, role)
	return client
}

type testWriter struct{ t testing.TB }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
