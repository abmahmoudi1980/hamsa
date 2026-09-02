package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"

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

// setupManager bootstraps the deployment's first account: the superadmin
// (002-multi-manager-support; was the manager pre-0010).
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

	// 1. Fresh deployment: setup creates the superadmin with tokens.
	first := setupManager(t, e, managerPhone, managerPassword)
	if first.User.Role != RoleSuperAdmin || first.User.ID == "" {
		t.Fatalf("bootstrap: role=%q id=%q want superadmin + non-empty id", first.User.Role, first.User.ID)
	}
	if first.AccessToken == "" || first.RefreshToken == "" || first.ExpiresIn <= 0 {
		t.Fatalf("token pair incomplete: %+v", first)
	}

	// 2. A second setup is rejected — bootstrap applies once.
	if rec := postJSON(t, e, "/api/v1/auth/setup", `{"phone":"09120000009","password":"hamsha-1234"}`, ""); rec.Code != http.StatusConflict {
		t.Fatalf("second setup: got %d want 409: %s", rec.Code, rec.Body)
	}

	// 3. The superadmin issues an invite; the resident registers with it
	// (default role).
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
	// its building scope (empty until a building exists; superadmin lists none).
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
	if me.ID != first.User.ID || me.Role != RoleSuperAdmin {
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

// --- 002-multi-manager-support US1: superadmin bootstrap + manager chain -----

type inviteResp struct {
	Code          string `json:"code"`
	Role          string `json:"role"`
	ExpiresInDays int    `json:"expires_in_days"`
}

// inviteRaw posts an arbitrary invite body with token and returns the parsed
// 201 payload alongside the recorder.
func inviteRaw(t *testing.T, e *authEnv, token, body string) (inviteResp, int, string) {
	t.Helper()
	rec := postJSON(t, e, "/api/v1/auth/invites", body, token)
	var out inviteResp
	if rec.Code == http.StatusCreated {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("invite body %s: %v", rec.Body, err)
		}
	}
	return out, rec.Code, rec.Body.String()
}

// meResp fetches /auth/me and returns its role + building scope.
func meResp(t *testing.T, e *authEnv, token string) (string, []any) {
	t.Helper()
	rec := e.request(t, http.MethodGet, "/api/v1/auth/me", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: got %d want 200: %s", rec.Code, rec.Body)
	}
	var me struct {
		Role      string `json:"role"`
		Buildings []any  `json:"buildings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("me body: %v", err)
	}
	return me.Role, me.Buildings
}

func TestIntegration_SuperadminCreatesManagers(t *testing.T) {
	e := newAuthEnv(t)
	super := setupManager(t, e, "09120000001", "hamsha-1234")

	// Superadmin /auth/me: role superadmin, empty (never absent) buildings.
	role, buildings := meResp(t, e, super.AccessToken)
	if role != RoleSuperAdmin {
		t.Fatalf("me role=%q want superadmin", role)
	}
	if buildings == nil || len(buildings) != 0 {
		t.Fatalf("me buildings=%v want []", buildings)
	}

	// Manager invite → registrant is a manager (granted nothing yet).
	inv, code, bodyText := inviteRaw(t, e, super.AccessToken, `{"phone":"09120000011","role":"manager"}`)
	if code != http.StatusCreated || inv.Role != RoleManager || inv.ExpiresInDays != 7 || inv.Code == "" {
		t.Fatalf("manager invite: code=%d payload=%+v body=%s", code, inv, bodyText)
	}
	reg := postJSON(t, e, "/api/v1/auth/register",
		`{"phone":"09120000011","code":"`+inv.Code+`","password":"hamsha-1234","name":"مدیر"}`, "")
	if reg.Code != http.StatusOK {
		t.Fatalf("register: got %d want 200: %s", reg.Code, reg.Body)
	}
	var pair tokenPairResp
	if err := json.Unmarshal(reg.Body.Bytes(), &pair); err != nil {
		t.Fatalf("register body: %v", err)
	}
	if pair.User.Role != RoleManager {
		t.Fatalf("registrant role=%q want manager", pair.User.Role)
	}
	role, buildings = meResp(t, e, pair.AccessToken)
	if role != RoleManager || len(buildings) != 0 {
		t.Fatalf("new manager me: role=%q buildings=%v want manager + []", role, buildings)
	}

	// invite.issued audit: actor = superadmin, payload = phone + role.
	found := 0
	for _, r := range e.auditor.records {
		if r.action != "invite.issued" {
			continue
		}
		found++
		if r.actorID == nil || r.actorID.String() != super.User.ID {
			t.Fatalf("invite.issued actor=%v want %s", r.actorID, super.User.ID)
		}
		after, ok := r.after.(map[string]any)
		if !ok || after["phone"] != "09120000011" || after["role"] != RoleManager {
			t.Fatalf("invite.issued after=%#v want phone+manager", r.after)
		}
	}
	if found != 1 {
		t.Fatalf("invite.issued entries=%d want 1", found)
	}
}

func TestIntegration_InviteNeverPromotesExistingUsers(t *testing.T) {
	e := newAuthEnv(t)
	super := setupManager(t, e, "09120000001", "hamsha-1234")
	res := inviteAndRegister(t, e, super.AccessToken, "09120000012", "hamsha-1234", "ساکن")

	// Manager invite for an existing resident: password reset only.
	inv, code, _ := inviteRaw(t, e, super.AccessToken, `{"phone":"09120000012","role":"manager"}`)
	if code != http.StatusCreated {
		t.Fatalf("manager invite for resident: %d", code)
	}
	reg := postJSON(t, e, "/api/v1/auth/register",
		`{"phone":"09120000012","code":"`+inv.Code+`","password":"brand-new-9"}`, "")
	if reg.Code != http.StatusOK {
		t.Fatalf("reset register: got %d want 200: %s", reg.Code, reg.Body)
	}
	var pair tokenPairResp
	if err := json.Unmarshal(reg.Body.Bytes(), &pair); err != nil {
		t.Fatalf("reset body: %v", err)
	}
	if pair.User.Role != RoleResident || pair.User.ID != res.User.ID {
		t.Fatalf("promotion leak: role=%q id=%s want resident + same id %s", pair.User.Role, pair.User.ID, res.User.ID)
	}
	role, _ := meResp(t, e, pair.AccessToken)
	if role != RoleResident {
		t.Fatalf("me after reset role=%q want resident", role)
	}

	// Manager invite for the SUPERADMIN's phone: role stays superadmin.
	sup, code, _ := inviteRaw(t, e, super.AccessToken, `{"phone":"09120000001","role":"manager"}`)
	if code != http.StatusCreated {
		t.Fatalf("invite for superadmin phone: %d", code)
	}
	reg = postJSON(t, e, "/api/v1/auth/register",
		`{"phone":"09120000001","code":"`+sup.Code+`","password":"recovered-9"}`, "")
	if reg.Code != http.StatusOK {
		t.Fatalf("superadmin reset: got %d want 200: %s", reg.Code, reg.Body)
	}
	if err := json.Unmarshal(reg.Body.Bytes(), &pair); err != nil {
		t.Fatalf("superadmin reset body: %v", err)
	}
	if pair.User.Role != RoleSuperAdmin {
		t.Fatalf("superadmin demoted/promoted: role=%q", pair.User.Role)
	}
	if rec := postJSON(t, e, "/api/v1/auth/login", `{"phone":"09120000001","password":"hamsha-1234"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old superadmin password still alive: %d", rec.Code)
	}
}

func TestIntegration_InviteRoleValidationAndGates(t *testing.T) {
	e := newAuthEnv(t)
	super := setupManager(t, e, "09120000001", "hamsha-1234")

	for _, bad := range []string{"superadmin", "wizard", "MANAGER"} {
		_, code, bodyText := inviteRaw(t, e, super.AccessToken, `{"phone":"09120000013","role":"`+bad+`"}`)
		if code != http.StatusBadRequest {
			t.Fatalf("role %q: got %d want 400: %s", bad, code, bodyText)
		}
	}

	// Missing role defaults to resident.
	inv, code, _ := inviteRaw(t, e, super.AccessToken, `{"phone":"09120000014"}`)
	if code != http.StatusCreated || inv.Role != RoleResident {
		t.Fatalf("default role: code=%d role=%q want 201+resident", code, inv.Role)
	}

	// Residents cannot issue invites at all.
	res := inviteAndRegister(t, e, super.AccessToken, "09120000015", "hamsha-1234", "ساکن")
	if _, code, _ := inviteRaw(t, e, res.AccessToken, `{"phone":"09120000016"}`); code != http.StatusForbidden {
		t.Fatalf("resident invite: got %d want 403", code)
	}
}

// --- 002-multi-manager-support US2: manager self-sufficiency -----------------

func TestIntegration_ManagerIssuesRoleInvitesAndMeScope(t *testing.T) {
	e := newAuthEnv(t)
	super := setupManager(t, e, "09120000001", "hamsha-1234")

	// Superadmin bootstraps one manager with the standard chain.
	inv, code, _ := inviteRaw(t, e, super.AccessToken, `{"phone":"09120000021","role":"manager"}`)
	if code != http.StatusCreated {
		t.Fatalf("bootstrap manager invite: %d", code)
	}
	reg := postJSON(t, e, "/api/v1/auth/register",
		`{"phone":"09120000021","code":"`+inv.Code+`","password":"hamsha-1234","name":"مدیر"}`, "")
	if reg.Code != http.StatusOK {
		t.Fatalf("manager register: %d %s", reg.Code, reg.Body)
	}
	var mgr tokenPairResp
	if err := json.Unmarshal(reg.Body.Bytes(), &mgr); err != nil {
		t.Fatalf("manager register body: %v", err)
	}

	// The MANAGER now issues invites for both roles (no superadmin needed).
	m, code, _ := inviteRaw(t, e, mgr.AccessToken, `{"phone":"09120000022","role":"manager"}`)
	if code != http.StatusCreated || m.Role != RoleManager {
		t.Fatalf("manager-issued manager invite: code=%d role=%q", code, m.Role)
	}
	r, code, _ := inviteRaw(t, e, mgr.AccessToken, `{"phone":"09120000023"}`)
	if code != http.StatusCreated || r.Role != RoleResident {
		t.Fatalf("manager-issued default invite: code=%d role=%q", code, r.Role)
	}
	if _, code, _ := inviteRaw(t, e, mgr.AccessToken, `{"phone":"09120000024","role":"nope"}`); code != http.StatusBadRequest {
		t.Fatalf("manager-issued invalid role: got %d want 400", code)
	}

	// Registrant of the manager invite is a manager.
	reg = postJSON(t, e, "/api/v1/auth/register",
		`{"phone":"09120000022","code":"`+m.Code+`","password":"hamsha-1234"}`, "")
	if reg.Code != http.StatusOK {
		t.Fatalf("second manager register: %d %s", reg.Code, reg.Body)
	}
	if err := json.Unmarshal(reg.Body.Bytes(), &mgr); err != nil {
		t.Fatalf("second manager body: %v", err)
	}
	if mgr.User.Role != RoleManager {
		t.Fatalf("second registrant role=%q want manager", mgr.User.Role)
	}

	// GET /auth/me reflects a granted building (the user_buildings scope the
	// manager auto-owns when creating a building — see building suite).
	buildingID := uuid.NewString()
	if err := e.exec(
		`INSERT INTO buildings (id, name) VALUES (?, 'برج آزمایش')`, buildingID); err != nil {
		t.Fatalf("seed building: %v", err)
	}
	if err := e.exec(
		`INSERT INTO user_buildings (user_id, building_id) VALUES (?, ?)`,
		mgr.User.ID, buildingID); err != nil {
		t.Fatalf("seed grant: %v", err)
	}
	rec := e.request(t, http.MethodGet, "/api/v1/auth/me", "", mgr.AccessToken)
	var me struct {
		Role      string   `json:"role"`
		Buildings []string `json:"buildings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("me body: %v", err)
	}
	if me.Role != RoleManager || len(me.Buildings) != 1 || me.Buildings[0] != buildingID {
		t.Fatalf("me scope: role=%q buildings=%v want manager + [%s]", me.Role, me.Buildings, buildingID)
	}
}

// FR-020/US4: a failing audit store never fails the audited operation —
// the invite is issued and redeemable even when Append errors.
func TestIntegration_InviteSucceedsWhenAuditFails(t *testing.T) {
	e := newAuthEnv(t)
	super := setupManager(t, e, "09120000001", "hamsha-1234")
	e.auditor.err = errors.New("audit store down")

	inv, code, bodyText := inviteRaw(t, e, super.AccessToken, `{"phone":"09120000031","role":"manager"}`)
	if code != http.StatusCreated || inv.Code == "" {
		t.Fatalf("invite with failing audit: code=%d body=%s", code, bodyText)
	}
	reg := postJSON(t, e, "/api/v1/auth/register",
		`{"phone":"09120000031","code":"`+inv.Code+`","password":"hamsha-1234"}`, "")
	if reg.Code != http.StatusOK {
		t.Fatalf("redemption of best-effort invite: got %d want 200: %s", reg.Code, reg.Body)
	}
	var pair tokenPairResp
	if err := json.Unmarshal(reg.Body.Bytes(), &pair); err != nil {
		t.Fatalf("register body: %v", err)
	}
	if pair.User.Role != RoleManager {
		t.Fatalf("role=%q want manager", pair.User.Role)
	}
}
