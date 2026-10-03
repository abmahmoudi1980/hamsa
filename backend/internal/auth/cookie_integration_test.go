package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cookieJar is the slice of browser cookie handling these tests need: retain
// Set-Cookie values, replay them, and read the readable CSRF token.
type cookieJar struct {
	refresh string
	csrf    string
}

func (j *cookieJar) absorb(rec *httptest.ResponseRecorder) {
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case RefreshCookieName:
			j.refresh = c.Value
		case CSRFCookieName:
			j.csrf = c.Value
		}
	}
}

func (j *cookieJar) header() string {
	var parts []string
	if j.refresh != "" {
		parts = append(parts, RefreshCookieName+"="+j.refresh)
	}
	if j.csrf != "" {
		parts = append(parts, CSRFCookieName+"="+j.csrf)
	}
	return strings.Join(parts, "; ")
}

// webPost is a session request from the SvelteKit client: it declares itself
// with X-Hamsa-Client: web so the server keeps the refresh token out of the
// JavaScript-readable body.
func webPost(t *testing.T, e *authEnv, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ClientHeaderName, ClientWeb)
	rec := httptest.NewRecorder()
	e.engine.ServeHTTP(rec, req)
	return rec
}

// cookieRequest drives a browser-shaped call: cookies on the wire, optional
// CSRF header, body only when given.
func cookieRequest(t *testing.T, e *authEnv, method, path, body, cookies, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}
	if csrf != "" {
		req.Header.Set(CSRFHeaderName, csrf)
	}
	rec := httptest.NewRecorder()
	e.engine.ServeHTTP(rec, req)
	return rec
}

// TestIntegration_BrowserCookieSession covers the F6 hardening: the refresh
// token is additionally issued as an httpOnly cookie, browser sessions never
// receive it in a JS-readable body, cookie-authenticated state changes require
// the double-submit CSRF token, and the original body-token contract keeps
// working unchanged for the Android client.
func TestIntegration_BrowserCookieSession(t *testing.T) {
	e := newAuthEnv(t)

	rec := postJSON(t, e, "/api/v1/auth/setup",
		`{"phone":"09120000001","password":"hamsha-1234","name":"مدیر"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("setup: got %d want 200: %s", rec.Code, rec.Body)
	}
	var pair tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &pair); err != nil {
		t.Fatalf("setup body: %v", err)
	}
	// The Android contract is untouched: no client header → body token present.
	if pair.RefreshToken == "" {
		t.Fatal("a non-browser session body must still carry refresh_token")
	}

	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case RefreshCookieName:
			if !c.HttpOnly {
				t.Error("refresh cookie must be HttpOnly (the F6 guarantee)")
			}
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("refresh cookie SameSite = %v, want Lax", c.SameSite)
			}
			if c.Path != "/api/v1/auth" {
				t.Errorf("refresh cookie Path = %q, want /api/v1/auth", c.Path)
			}
		case CSRFCookieName:
			if c.HttpOnly {
				t.Error("CSRF cookie must be readable by the client, not HttpOnly")
			}
		}
	}

	// A web login is cookie-only: the body must NOT hand the long-lived token
	// to page JavaScript.
	rec = webPost(t, e, "/api/v1/auth/login", `{"phone":"09120000001","password":"hamsha-1234"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("web login: got %d want 200: %s", rec.Code, rec.Body)
	}
	var webLogin tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &webLogin); err != nil {
		t.Fatalf("web login body: %v", err)
	}
	if webLogin.AccessToken == "" {
		t.Fatal("web login body must still carry access_token")
	}
	if webLogin.RefreshToken != "" {
		t.Fatal("web login must not expose refresh_token in the body")
	}
	var jar cookieJar
	jar.absorb(rec)
	if jar.refresh == "" || jar.csrf == "" {
		t.Fatalf("web login must set both cookies: refresh=%q csrf=%q", jar.refresh, jar.csrf)
	}

	// A cookie-authenticated state change without the double-submit token is a
	// CSRF request → 403.
	if got := cookieRequest(t, e, http.MethodPost, "/api/v1/auth/refresh", "", jar.header(), ""); got.Code != http.StatusForbidden {
		t.Fatalf("cookie refresh without CSRF: got %d want 403: %s", got.Code, got.Body)
	}
	if got := cookieRequest(t, e, http.MethodPost, "/api/v1/auth/refresh", "", jar.header(), "wrong-token"); got.Code != http.StatusForbidden {
		t.Fatalf("cookie refresh with wrong CSRF: got %d want 403", got.Code)
	}

	// The matching token rotates the pair; still no body token.
	rec = cookieRequest(t, e, http.MethodPost, "/api/v1/auth/refresh", "", jar.header(), jar.csrf)
	if rec.Code != http.StatusOK {
		t.Fatalf("cookie refresh: got %d want 200: %s", rec.Code, rec.Body)
	}
	var rotated tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("refresh body: %v", err)
	}
	if rotated.AccessToken == "" {
		t.Fatal("cookie refresh must return a fresh access token")
	}
	if rotated.RefreshToken != "" {
		t.Fatal("cookie refresh must not expose refresh_token in the body")
	}
	oldRefresh := jar.refresh
	jar.absorb(rec)
	if jar.refresh == oldRefresh {
		t.Fatal("refresh cookie was not rotated with the token")
	}

	// The Android path (body token, no cookie, no CSRF) still yields a body
	// token back.
	rec = postJSON(t, e, "/api/v1/auth/refresh", `{"refresh_token":"`+pair.RefreshToken+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("body refresh: got %d want 200: %s", rec.Code, rec.Body)
	}
	var mobile tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &mobile); err != nil {
		t.Fatalf("body refresh body: %v", err)
	}
	if mobile.RefreshToken == "" {
		t.Fatal("a body-authenticated refresh must return the rotated token in the body")
	}

	// Logout with the cookie + CSRF revokes the family and expires both cookies.
	rec = cookieRequest(t, e, http.MethodPost, "/api/v1/auth/logout", "", jar.header(), jar.csrf)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("cookie logout: got %d want 204: %s", rec.Code, rec.Body)
	}
	cleared := map[string]bool{}
	for _, c := range rec.Result().Cookies() {
		cleared[c.Name] = c.Value == ""
	}
	if !cleared[RefreshCookieName] || !cleared[CSRFCookieName] {
		t.Fatalf("logout must expire both cookies, got %v", cleared)
	}

	// The revoked cookie can no longer mint a session.
	if got := cookieRequest(t, e, http.MethodPost, "/api/v1/auth/refresh", "", jar.header(), jar.csrf); got.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: got %d want 401", got.Code)
	}
}

// TestIntegration_RefreshWithoutTokenIsUnauthenticated pins the 401 so the web
// client routes an empty cookie jar to sign-in rather than showing a form error.
func TestIntegration_RefreshWithoutTokenIsUnauthenticated(t *testing.T) {
	e := newAuthEnv(t)
	if got := cookieRequest(t, e, http.MethodPost, "/api/v1/auth/refresh", "", "", ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("tokenless refresh: got %d want 401", got.Code)
	}
}
