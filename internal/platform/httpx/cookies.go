package httpx

import (
	"net/http"
	"time"
)

// Cookie names are part of the contract (components.securitySchemes).
const (
	SessionCookieName = "netopology_session"
	GuestCookieName   = "netopology_share"
	OAuthCookieName   = "netopology_oauth" // signed state + PKCE verifier, 10 minutes
)

// Cookies builds the app's cookies with consistent attributes. Handlers return them in an
// output field tagged `header:"Set-Cookie"`. The Secure flag comes from config
// (NETOPOLOGY_COOKIE_SECURE, default true), which gosec cannot see: hence the nolint comments.
type Cookies struct{ Secure bool }

// Session: HttpOnly, SameSite=Lax (so the Google/Discord redirect back can complete), Path=/.
func (c Cookies) Session(token string, ttl time.Duration, remember bool) http.Cookie {
	cookie := http.Cookie{ //nolint:gosec // see Secure note above
		Name: SessionCookieName, Value: token, Path: "/",
		HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteLaxMode,
	}
	if remember {
		cookie.MaxAge = int(ttl.Seconds())
	}
	return cookie
}

// Guest: valid for share endpoints only, SameSite=Strict.
func (c Cookies) Guest(value string, ttl time.Duration) http.Cookie {
	return http.Cookie{ //nolint:gosec // see Secure note above
		Name: GuestCookieName, Value: value, Path: "/api/v1/share", MaxAge: int(ttl.Seconds()),
		HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteStrictMode,
	}
}

// OAuth: survives the round trip to the provider, only sent back to the callback path.
func (c Cookies) OAuth(value string) http.Cookie {
	return http.Cookie{ //nolint:gosec // see Secure note above
		Name: OAuthCookieName, Value: value, Path: "/api/v1", MaxAge: 600,
		HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteLaxMode,
	}
}

// Clear expires a cookie (logout, failed OAuth).
func (c Cookies) Clear(name, path string) http.Cookie {
	return http.Cookie{ //nolint:gosec // see Secure note above
		Name: name, Value: "", Path: path, MaxAge: -1, HttpOnly: true, Secure: c.Secure,
	}
}
