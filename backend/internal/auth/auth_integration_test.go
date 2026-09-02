package auth

import (
	"encoding/json"
	"net/http"
	"testing"

	httptest "net/http/httptest"
)

// US1 integration suite: setup bootstrap, invite registration, password
// login, password change, token refresh rotation, and logout through the real
// HTTP handlers against a migrated PostgreSQL (see integration_harness_test.go).

type tokenPairResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"user"`
}

func postJSON(t *testing.T, e *authEnv, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	return e.request(t, http.MethodPost, path, body, token)
}

// setupManager bootstraps the deployment's first (manager) account.
func setupManager(t *testing.T, e *authEnv, phone, password string) tokenPairResp {
	t.Helper()
	body := `{"phone":"` + phone + `","password":"` + password + `","name":"مدیر"}`
	rec := postJSON(t, e, "/api/v1/auth/setup", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("setup %s: got %d want 200: %s", phone, rec.Code, rec.Body)
	}
	var pair tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &pair); err != nil {
		t.Fatalf("setup body for %s: %v", phone, err)
	}
	return pair
}

// inviteAndRegister drives manager-invite → register and returns the new
// session.
func inviteAndRegister(t *testing.T, e *authEnv, managerToken, phone, password, name string) tokenPairResp {
	t.Helper()

	rec := postJSON(t, e, "/api/v1/auth/invites", `{"phone":"`+phone+`"}`, managerToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite %s: got %d want 201: %s", phone, rec.Code, rec.Body)
	}
	var invite struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &invite); err != nil || invite.Code == "" {
		t.Fatalf("invite body for %s: code=%q err=%v", phone, invite.Code, err)
	}

	body := `{"phone":"` + phone + `","code":"` + invite.Code + `","password":"` + password + `","name":"` + name + `"}`
	rec = postJSON(t, e, "/api/v1/auth/register", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register %s: got %d want 200: %s", phone, rec.Code, rec.Body)
	}
	var pair tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &pair); err != nil {
		t.Fatalf("register body for %s: %v", phone, err)
	}
	return pair
}

func TestIntegration_SetupBootstrapAndRegistration(t *testing.T) {
	e := newAuthEnv(t)
	const managerPhone = "09120000001"
	const managerPassword = "hamsha-1234"

	// 1. Fresh deployment: setup creates the manager with tokens.
	first := setupManager(t, e, managerPhone, managerPassword)
	if first.User.Role != RoleManager || first.User.ID == "" {
		t.Fatalf("bootstrap: role=%q id=%q want manager + non-empty id", first.User.Role, first.User.ID)
	}
	if first.AccessToken == "" || first.RefreshToken == "" || first.ExpiresIn <= 0 {
		t.Fatalf("token pair incomplete: %+v", first)
	}

	// 2. A second setup is rejected — bootstrap applies once.
	if rec := postJSON(t, e, "/api/v1/auth/setup", `{"phone":"09120000009","password":"hamsha-1234"}`, ""); rec.Code != http.StatusConflict {
		t.Fatalf("second setup: got %d want 409: %s", rec.Code, rec.Body)
	}

	// 3. Manager issues an invite; the resident registers with it.
	second := inviteAndRegister(t, e, first.AccessToken, "09120000002", "hamsha-1234", "ساکن")
	if second.User.Role != RoleResident {
		t.Fatalf("second user role=%q want resident", second.User.Role)
	}

	// 4. Password login works for both accounts.
	for _, tc := range []struct{ phone, password string }{
		{managerPhone, managerPassword},
		{"09120000002", "hamsha-1234"},
	} {
		body := `{"phone":"` + tc.phone + `","password":"` + tc.password + `"}`
		if rec := postJSON(t, e, "/api/v1/auth/login", body, ""); rec.Code != http.StatusOK {
			t.Fatalf("login %s: got %d want 200: %s", tc.phone, rec.Code, rec.Body)
		}
	}

	// 5. Wrong password → 401 with no account enumeration (same code for
	// unknown phones).
	for _, body := range []string{
		`{"phone":"09120000002","password":"wrong-pass"}`,
		`{"phone":"09129999999","password":"hamsha-1234"}`,
	} {
		if rec := postJSON(t, e, "/api/v1/auth/login", body, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("login %s: got %d want 401", body, rec.Code)
		}
	}

	// 6. GET /auth/me with the access token returns the same identity plus
	// its (empty until US2) building scope.
	rec := e.request(t, http.MethodGet, "/api/v1/auth/me", "", first.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: got %d want 200: %s", rec.Code, rec.Body)
	}
	var me struct {
		ID        string `json:"id"`
		Role      string `json:"role"`
		Buildings []any  `json:"buildings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("me body: %v", err)
	}
	if me.ID != first.User.ID || me.Role != RoleManager {
		t.Fatalf("me mismatch: id=%s role=%s", me.ID, me.Role)
	}

	// 7. Every session issue was audited (T025): setup + register + 2 logins.
	logins := 0
	for _, r := range e.auditor.records {
		if r.action == "user.login" && r.actorID != nil && r.objectType == "user" {
			logins++
		}
	}
	if logins != 4 {
		t.Fatalf("user.login audit entries=%d want 4", logins)
	}
}

func TestIntegration_InviteLifecycle(t *testing.T) {
	e := newAuthEnv(t)
	manager := setupManager(t, e, "09120000001", "hamsha-1234")
	const phone = "09120000003"

	// 1. Register consumes the code — reuse fails.
	rec := postJSON(t, e, "/api/v1/auth/invites", `{"phone":"`+phone+`"}`, manager.AccessToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite: got %d want 201: %s", rec.Code, rec.Body)
	}
	var invite struct{ Code string }
	if err := json.Unmarshal(rec.Body.Bytes(), &invite); err != nil || invite.Code == "" {
		t.Fatalf("invite body: %v", err)
	}
	body := `{"phone":"` + phone + `","code":"` + invite.Code + `","password":"hamsha-1234"}`
	if rec := postJSON(t, e, "/api/v1/auth/register", body, ""); rec.Code != http.StatusOK {
		t.Fatalf("register: got %d want 200: %s", rec.Code, rec.Body)
	}
	if rec := postJSON(t, e, "/api/v1/auth/register", body, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("invite reuse: got %d want 401", rec.Code)
	}

	// 2. A wrong code never registers a new phone.
	if rec := postJSON(t, e, "/api/v1/auth/register", `{"phone":"09120000004","code":"AAAAAAAA","password":"hamsha-1234"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code: got %d want 401", rec.Code)
	}

	// 3. A new invite for an existing phone resets the password (recovery
	// path) — the old password stops working, the new one logs in.
	reset := inviteAndRegister(t, e, manager.AccessToken, phone, "new-password-9", "")
	rec = postJSON(t, e, "/api/v1/auth/login", `{"phone":"`+phone+`","password":"hamsha-1234"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("old password after reset: got %d want 401", rec.Code)
	}
	rec = postJSON(t, e, "/api/v1/auth/login", `{"phone":"`+phone+`","password":"new-password-9"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("new password login: got %d want 200: %s", rec.Code, rec.Body)
	}
	if reset.User.ID == "" {
		t.Fatalf("reset session missing user id")
	}
}

func TestIntegration_PasswordChange(t *testing.T) {
	e := newAuthEnv(t)
	pair := setupManager(t, e, "09120000001", "hamsha-1234")

	// 1. Wrong current password → 401.
	if rec := postJSON(t, e, "/api/v1/auth/password",
		`{"current_password":"nope","new_password":"changed-99"}`, pair.AccessToken); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current: got %d want 401", rec.Code)
	}

	// 2. Correct change → 204, old login dies, new login works.
	if rec := postJSON(t, e, "/api/v1/auth/password",
		`{"current_password":"hamsha-1234","new_password":"changed-99"}`, pair.AccessToken); rec.Code != http.StatusNoContent {
		t.Fatalf("change: got %d want 204: %s", rec.Code, rec.Body)
	}
	if rec := postJSON(t, e, "/api/v1/auth/login", `{"phone":"09120000001","password":"hamsha-1234"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old password: got %d want 401", rec.Code)
	}
	if rec := postJSON(t, e, "/api/v1/auth/login", `{"phone":"09120000001","password":"changed-99"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("new password: got %d want 200: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_RefreshRotation(t *testing.T) {
	e := newAuthEnv(t)
	pair := setupManager(t, e, "09120000001", "hamsha-1234")

	// Rotate: old refresh → new pair.
	rec := postJSON(t, e, "/api/v1/auth/refresh", `{"refresh_token":"`+pair.RefreshToken+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh: got %d want 200: %s", rec.Code, rec.Body)
	}
	var rotated tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("refresh body: %v", err)
	}
	if rotated.AccessToken == "" || rotated.RefreshToken == "" || rotated.RefreshToken == pair.RefreshToken {
		t.Fatalf("rotation did not produce a fresh pair: %+v", rotated)
	}

	// Reuse of the OLD token revokes the whole family → 401…
	if rec := postJSON(t, e, "/api/v1/auth/refresh", `{"refresh_token":"`+pair.RefreshToken+`"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("reuse: got %d want 401", rec.Code)
	}
	// …and the successor from that family is dead too.
	if rec := postJSON(t, e, "/api/v1/auth/refresh", `{"refresh_token":"`+rotated.RefreshToken+`"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("family revoked successor: got %d want 401", rec.Code)
	}
}

func TestIntegration_LogoutRevokesFamily(t *testing.T) {
	e := newAuthEnv(t)
	pair := setupManager(t, e, "09120000001", "hamsha-1234")

	if rec := postJSON(t, e, "/api/v1/auth/logout", `{"refresh_token":"`+pair.RefreshToken+`"}`, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("logout: got %d want 204", rec.Code)
	}

	// Refresh after logout → 401; logout again stays 204 (idempotent).
	if rec := postJSON(t, e, "/api/v1/auth/refresh", `{"refresh_token":"`+pair.RefreshToken+`"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: got %d want 401", rec.Code)
	}
	if rec := postJSON(t, e, "/api/v1/auth/logout", `{"refresh_token":"`+pair.RefreshToken+`"}`, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("idempotent logout: got %d want 204", rec.Code)
	}
}
