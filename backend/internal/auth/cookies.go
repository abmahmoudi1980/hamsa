package auth

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Browser session cookies (003-web-frontend, F6).
//
// The refresh token is still returned in the JSON body because the Android
// client presents it explicitly and has no cookie jar. For the browser the
// token ADDITIONALLY travels as an httpOnly cookie, so a script injected into
// the page cannot read it — the residual XSS exposure tracked as F6.
//
// Which channel a request used is decided per request by refreshTokenFrom: a
// token in the body is self-presented (a cross-site page cannot read or forge
// it) and needs no CSRF proof; an ambient cookie is attached by the browser to
// any request to its path, so a cookie-authenticated request must also present
// the double-submit token.
const (
	// RefreshCookieName is the httpOnly cookie carrying the rotating refresh
	// token. Scoped to the auth routes so it is not sent anywhere else.
	RefreshCookieName = "hamsa_refresh"
	// CSRFCookieName carries the readable double-submit token. Readable by
	// design: the client must echo it in the X-CSRF-Token header.
	CSRFCookieName = "hamsa_csrf"
	// CSRFHeaderName is the header a cookie-authenticated request must echo.
	CSRFHeaderName = "X-CSRF-Token"

	// ClientHeaderName marks the web client, which the server then treats as
	// cookie-session-only: the refresh token is NOT echoed in the response body
	// (an XSS could otherwise read the long-lived token out of the JSON and
	// defeat the httpOnly cookie). The Android client sends no such header and
	// keeps receiving the body token.
	ClientHeaderName = "X-Hamsa-Client"
	// ClientWeb is the ClientHeaderName value the SvelteKit client sends.
	ClientWeb = "web"

	// refreshCookiePath scopes the refresh cookie to the auth endpoints.
	refreshCookiePath = "/api/v1/auth"
	// csrfCookiePath must stay "/": document.cookie on any SPA route has to be
	// able to read the token, and SvelteKit routes live at the site root.
	csrfCookiePath = "/"
)

// CookieConfig configures the browser session cookies. The zero value is valid
// and correct for plain-HTTP development (Secure off); production MUST set
// Secure so the cookie never crosses an unencrypted connection.
type CookieConfig struct {
	// Secure marks the cookies Secure. True in production (HTTPS only).
	Secure bool
}

// defaultRefreshTTL bounds the cookie lifetime if the token service is
// unavailable (only in a misconfigured build).
const defaultRefreshTTL = 30 * 24 * time.Hour

// setSessionCookies issues a fresh refresh cookie and a fresh CSRF token. It is
// called wherever a session (or a rotated pair) is issued: setup, register,
// login, and refresh.
func (h *Handler) setSessionCookies(c *gin.Context, refreshToken string) {
	maxAge := h.refreshCookieMaxAge()
	// Lax blocks the cross-site POST a CSRF attack needs, while still allowing
	// the top-level navigation the payment gateway returns the browser with.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, refreshToken, maxAge, refreshCookiePath, "", h.Cookies.Secure, true)
	if csrf, err := randomHex(32); err == nil {
		c.SetCookie(CSRFCookieName, csrf, maxAge, csrfCookiePath, "", h.Cookies.Secure, false)
	}
}

// clearSessionCookies expires both cookies on logout.
func (h *Handler) clearSessionCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, "", -1, refreshCookiePath, "", h.Cookies.Secure, true)
	c.SetCookie(CSRFCookieName, "", -1, csrfCookiePath, "", h.Cookies.Secure, false)
}

// refreshCookieMaxAge matches the cookie lifetime to the refresh token's.
func (h *Handler) refreshCookieMaxAge() int {
	ttl := defaultRefreshTTL
	if h.Tokens != nil && h.Tokens.refreshTTL > 0 {
		ttl = h.Tokens.refreshTTL
	}
	return int(ttl.Seconds())
}

// refreshTokenFrom returns the presented refresh token and whether it came from
// the cookie rather than the request body.
//
// The body wins: a client that sends the token itself does not depend on
// ambient browser credentials and therefore needs no CSRF token. Only a
// browser-attached token — which any site could trigger — is CSRF-guarded.
func (h *Handler) refreshTokenFrom(c *gin.Context) (token string, fromCookie bool) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		return req.RefreshToken, false
	}
	if raw, err := c.Cookie(RefreshCookieName); err == nil && raw != "" {
		return raw, true
	}
	return "", false
}

// browserClient reports whether the request comes from the SvelteKit web
// client, which relies on the session cookie and must never receive the refresh
// token in a JavaScript-readable body.
func (h *Handler) browserClient(c *gin.Context) bool {
	return c.GetHeader(ClientHeaderName) == ClientWeb
}

// validCSRF enforces the double-submit check for a cookie-authenticated
// request: the readable cookie must equal the header. The comparison is
// constant-time so a probe cannot recover the token byte by byte.
func (h *Handler) validCSRF(c *gin.Context) bool {
	cookie, err := c.Cookie(CSRFCookieName)
	if err != nil || cookie == "" {
		return false
	}
	header := c.GetHeader(CSRFHeaderName)
	if header == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) == 1
}
