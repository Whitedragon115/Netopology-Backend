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
	log := slog.New(slog.DiscardHandler)
	if testing.Verbose() {
		log = slog.New(slog.NewTextHandler(testWriter{t}, nil))
	}
	h, err := server.New(server.Deps{Config: cfg, DB: d.DB, Log: log})
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

type testWriter struct{ t testing.TB }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
