// Package server wires the HTTP stack: chi router, middleware, the Huma API and every feature.
// It is the only package that knows all features; features never import each other.
package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Whitedragon115/Netopology-Backend/internal/account"
	"github.com/Whitedragon115/Netopology-Backend/internal/audit"
	"github.com/Whitedragon115/Netopology-Backend/internal/auth"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/canvas"
	"github.com/Whitedragon115/Netopology-Backend/internal/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/inventory"
	"github.com/Whitedragon115/Netopology-Backend/internal/member"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/config"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/setup"
	"github.com/Whitedragon115/Netopology-Backend/internal/share"
	"github.com/Whitedragon115/Netopology-Backend/internal/workspace"
)

// Version is set at build time: -ldflags "-X .../internal/server.Version=1.2.3".
var Version = "0.1.0-dev"

// Deps are the process-wide dependencies handed to features.
type Deps struct {
	Config *config.Config
	DB     *pg.DB
	Log    *slog.Logger
}

// featureHandler is what every feature exposes to the server.
type featureHandler interface{ Register(api huma.API) }

// New returns the full HTTP handler for the api process.
func New(d Deps) (http.Handler, error) {
	httpx.InstallErrors(d.Log)
	trusted, err := httpx.ParsePrefixes(d.Config.TrustedProxies)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(httpx.RequestContext(trusted))
	r.Use(httpx.AccessLog(d.Log))

	// probes for Docker / the gateway, outside the API (no auth, no logging noise in the spec)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := d.DB.Pool.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(httpx.SameOrigin(d.Config.PublicURL))
		r.Use(auth.Middleware(d.DB, d.Config.SessionTTL, httpx.Cookies{Secure: d.Config.CookieSecure}, d.Log))
		r.NotFound(problem(http.StatusNotFound, "not_found", "No such API endpoint."))
		r.MethodNotAllowed(problem(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed on this endpoint."))
		api := humachi.New(r, apiConfig(d.Config.Dev()))
		for _, f := range features(d) {
			f.Register(api)
		}
	})
	return r, nil
}

func problem(status int, code, detail string) http.HandlerFunc {
	body, _ := json.Marshal(httpx.Problem{Title: http.StatusText(status), Status: status, Code: code, Detail: detail})
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

// features builds every feature with its dependencies. Add new features here.
func features(d Deps) []featureHandler {
	az := authz.NewResolver(d.DB)
	sessions := &auth.Sessions{TTL: d.Config.SessionTTL, Cookies: httpx.Cookies{Secure: d.Config.CookieSecure}}

	return []featureHandler{
		setup.NewHandler(setup.NewService(d.DB, sessions)),
		auth.NewHandler(auth.NewService(d.DB, d.Config, sessions)),
		account.NewHandler(account.NewService(d.DB)),
		workspace.NewHandler(workspace.NewService(d.DB, az)),
		member.NewHandler(member.NewService(d.DB, az)),
		inventory.NewHandler(inventory.NewService(d.DB, az)),
		canvas.NewHandler(canvas.NewService(d.DB, az)),
		share.NewHandler(share.NewService(d.DB, az)),
		events.NewHandler(events.NewService(d.DB, az)),
		audit.NewHandler(audit.NewService(d.DB, az)),
	}
}

// apiConfig mirrors the contract's info, servers and security schemes.
func apiConfig(dev bool) huma.Config {
	huma.DefaultArrayNullable = false // the contract never uses null for arrays (send [])
	cfg := huma.DefaultConfig("Netopology API", Version)
	cfg.CreateHooks = nil // no `$schema` fields / Link headers: the contract has none
	cfg.Info.Description = "Draw, understand and operate your whole infrastructure. Contract: api/contract.yaml."
	cfg.Info.License = &huma.License{Name: "Apache-2.0", Identifier: "Apache-2.0"}
	cfg.Servers = []*huma.Server{{URL: "/api/v1"}}
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {
			Type: "apiKey", In: "cookie", Name: httpx.SessionCookieName,
			Description: "Set by login/setup. HttpOnly, Secure, SameSite=Lax.",
		},
		"guest": {
			Type: "apiKey", In: "cookie", Name: httpx.GuestCookieName,
			Description: "Set by `POST /share/{token}/unlock`. Valid for one share link only.",
		},
	}
	cfg.Security = []map[string][]string{{"session": {}}}
	cfg.OpenAPIPath = "/openapi" // -> /api/v1/openapi.json and .yaml
	cfg.SchemasPath = ""
	cfg.DocsPath = ""
	if dev {
		cfg.DocsPath = "/docs" // Scalar at /api/v1/docs
		cfg.DocsRenderer = huma.DocsRendererScalar
	}
	return cfg
}

// OpenAPI builds the API without a database and returns the generated spec (for `netopology
// openapi` and the contract test). Handlers are registered but never called.
func OpenAPI() *huma.OpenAPI {
	httpx.InstallErrors(slog.New(slog.DiscardHandler))
	r := chi.NewRouter()
	var api huma.API
	r.Route("/api/v1", func(r chi.Router) {
		api = humachi.New(r, apiConfig(false))
		for _, f := range features(Deps{Config: &config.Config{SessionTTL: time.Hour}}) {
			f.Register(api)
		}
	})
	return api.OpenAPI()
}
