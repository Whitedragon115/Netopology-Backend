package httpx

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/logging"
)

// Client is who sent the request, for audit entries and rate limits.
type Client struct {
	IP        netip.Addr
	UserAgent string
	RequestID string
}

type clientKey struct{}

// ClientFrom returns the request's client info (zero value outside a request).
func ClientFrom(ctx context.Context) Client {
	c, _ := ctx.Value(clientKey{}).(Client)
	return c
}

// RequestContext assigns a request id, resolves the real client IP (X-Forwarded-For is believed
// only from trusted proxies, i.e. the gateway) and adds both to the logging context.
func RequestContext(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" || len(id) > 64 {
				id = uuid.Must(uuid.NewV7()).String()
			}
			w.Header().Set("X-Request-ID", id)
			c := Client{IP: clientIP(r, trusted), UserAgent: truncate(r.UserAgent(), 500), RequestID: id}
			ctx := context.WithValue(r.Context(), clientKey{}, c)
			ctx = logging.With(ctx, slog.String("request_id", id))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AccessLog writes one line per request. It never logs bodies, cookies or query strings
// (share tokens and OAuth codes travel in paths/queries).
func AccessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			path := r.URL.Path
			if strings.HasPrefix(path, "/api/v1/share/") {
				path = "/api/v1/share/{token}/…" // never log share tokens
			}
			log.InfoContext(r.Context(), "http",
				"method", r.Method, "path", path, "status", sw.status, "duration_ms", time.Since(start).Milliseconds())
		})
	}
}

// SameOrigin blocks cross-site writes (CSRF). A mutating request that carries one of our cookies
// must have an Origin (or Referer) equal to NETOPOLOGY_PUBLIC_URL. Requests without our cookies
// (curl, scripts) are not affected: they have no ambient authority to abuse.
func SameOrigin(publicURL string) func(http.Handler) http.Handler {
	want := origin(publicURL)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			if !hasOurCookie(r) {
				next.ServeHTTP(w, r)
				return
			}
			got := r.Header.Get("Origin")
			if got == "" {
				got = origin(r.Referer())
			}
			if got != want {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"title":"Forbidden","status":403,"code":"origin.mismatch","detail":"Cross-site request blocked."}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ParsePrefixes parses NETOPOLOGY_TRUSTED_PROXIES.
func ParsePrefixes(cidrs []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(strings.TrimSpace(c))
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func clientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remote, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	remote = remote.Unmap()
	if !inAny(remote, trusted) {
		return remote
	}
	// walk X-Forwarded-For from the right, skipping our own proxies
	parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if err != nil {
			break
		}
		ip = ip.Unmap()
		if !inAny(ip, trusted) {
			return ip
		}
	}
	return remote
}

func inAny(ip netip.Addr, prefixes []netip.Prefix) bool {
	for _, p := range prefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func origin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func hasOurCookie(r *http.Request) bool {
	for _, c := range r.Cookies() {
		if c.Name == SessionCookieName || c.Name == GuestCookieName {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

// Flush keeps SSE working through this wrapper.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
