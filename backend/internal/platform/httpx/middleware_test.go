package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// serve runs the CORS middleware over a single request and returns the
// recorder, so each test reads as "one request, one expectation".
func serve(h gin.HandlerFunc, method, path, origin string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, nil)
	if origin != "" {
		c.Request.Header.Set("Origin", origin)
	}
	h(c)
	return rec
}

// TestCORSAllowedOriginReflected verifies a configured origin is echoed back so
// the browser accepts the response (required by the fetch spec).
func TestCORSAllowedOriginReflected(t *testing.T) {
	rec := serve(CORS([]string{"http://localhost:5173"}),
		http.MethodGet, "/api/v1/me/home", "http://localhost:5173")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Allow-Origin = %q, want the configured origin", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin (per-origin responses must not be cached across origins)", got)
	}
	// The web client sends its refresh cookie, so an allowlisted origin must
	// permit credentials (003-web-frontend, F6).
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials = %q, want true for an allowlisted origin", got)
	}
}

// TestCORSDeniesUnlistedOrigin is the host-theft guard: an unlisted origin must
// receive NO Allow-Origin header at all, so the browser blocks the response.
func TestCORSDeniesUnlistedOrigin(t *testing.T) {
	rec := serve(CORS([]string{"http://localhost:5173"}),
		http.MethodGet, "/api/v1/me/home", "https://evil.example")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty for an unlisted origin", got)
	}
}

// TestCORSEmptyAllowlistSendsNoHeaders pins the production posture: the web
// client is same-origin with the API, so no CORS headers are emitted.
func TestCORSEmptyAllowlistSendsNoHeaders(t *testing.T) {
	rec := serve(CORS(nil), http.MethodGet, "/api/v1/me/home", "http://localhost:5173")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty with no configured origins", got)
	}
}

// TestCORSWildcardKeepsLegacyBehaviour keeps the escape hatch working for a
// single "*" entry.
func TestCORSWildcardKeepsLegacyBehaviour(t *testing.T) {
	rec := serve(CORS([]string{"*"}), http.MethodGet, "/api/v1/me/home", "https://any.example")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin = %q, want *", got)
	}
	// "*" cannot be combined with credentials; the legacy escape hatch must
	// stay credential-less or a browser would reject it.
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Allow-Credentials = %q, want empty on the wildcard path", got)
	}
}

// TestCORSNormalizesTrailingSlashAndCase prevents a config typo (a trailing
// slash, or an uppercase scheme/host) from silently rejecting the real origin.
func TestCORSNormalizesTrailingSlashAndCase(t *testing.T) {
	rec := serve(CORS([]string{"http://localhost:5173/"}),
		http.MethodGet, "/api/v1/me/home", "HTTP://Localhost:5173")

	// The header must echo the request's Origin verbatim (browsers compare it
	// byte-for-byte); only the allowlist lookup is normalized.
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "HTTP://Localhost:5173" {
		t.Fatalf("Allow-Origin = %q, want the request origin echoed verbatim", got)
	}
}

// TestCORSNoOriginRequestIsUntouched keeps non-browser clients (the Android
// app, curl, server-to-server) free of CORS headers.
func TestCORSNoOriginRequestIsUntouched(t *testing.T) {
	rec := serve(CORS([]string{"http://localhost:5173"}), http.MethodGet, "/api/v1/me/home", "")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty when the request carries no Origin", got)
	}
}

// TestCORSPreflightShortCircuits ensures OPTIONS is answered directly with 204
// and the auth header is advertised, so a JSON POST is not blocked.
func TestCORSPreflightShortCircuits(t *testing.T) {
	rec := serve(CORS([]string{"http://localhost:5173"}),
		http.MethodOptions, "/api/v1/auth/login", "http://localhost:5173")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("preflight must advertise the Authorization header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-CSRF-Token") {
		t.Fatalf("Allow-Headers = %q, want it to include X-CSRF-Token", got)
	}
}
