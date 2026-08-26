package auth

import (
	"encoding/json"
	"net/http"
	"testing"

	httptest "net/http/httptest"
)

// US1 integration suite (T021): OTP flow, token refresh rotation, rate
// limiting, logout, and first-user bootstrap through the real HTTP handlers
// against a migrated PostgreSQL (see integration_harness_test.go).

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

// requestOTPAndVerify drives request→verify and returns the token pair.
func requestOTPAndVerify(t *testing.T, e *authEnv, phone string) tokenPairResp {
	t.Helper()

	rec := postJSON(t, e, "/api/v1/auth/otp/request", `{"phone":"`+phone+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("otp request %s: got %d want 200: %s", phone, rec.Code, rec.Body)
	}
	var issued struct {
		DevCode string `json:"dev_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil || len(issued.DevCode) != 6 {
		t.Fatalf("otp request body for %s: code=%q err=%v", phone, issued.DevCode, err)
	}

	rec = postJSON(t, e, "/api/v1/auth/otp/verify",
		`{"phone":"`+phone+`","code":"`+issued.DevCode+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify %s: got %d want 200: %s", phone, rec.Code, rec.Body)
	}
	var pair tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &pair); err != nil {
		t.Fatalf("verify body for %s: %v", phone, err)
	}
	return pair
}

func TestIntegration_OTPFlowAndFirstUserBootstrap(t *testing.T) {
	e := newAuthEnv(t)
	const phone = "09120000001"

	// 1. Request → dev mode returns the 6-digit code in the response.
	rec := postJSON(t, e, "/api/v1/auth/otp/request", `{"phone":"`+phone+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("otp request: got %d want 200: %s", rec.Code, rec.Body)
	}
	var issued struct {
		DevCode string `json:"dev_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil || len(issued.DevCode) != 6 {
		t.Fatalf("dev_code missing: body=%s err=%v", rec.Body, err)
	}

	// 2. Immediate re-request inside the 60 s throttle window → 429.
	if rec := postJSON(t, e, "/api/v1/auth/otp/request", `{"phone":"`+phone+`"}`, ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("throttled re-request: got %d want 429", rec.Code)
	}

	// 3. Wrong code → 401 UNAUTHENTICATED envelope.
	if rec := postJSON(t, e, "/api/v1/auth/otp/verify", `{"phone":"`+phone+`","code":"000000"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code: got %d want 401", rec.Code)
	}

	// 4. Correct code → tokens + user; first user of a fresh deployment is
	// manager (T024 bootstrap).
	rec = postJSON(t, e, "/api/v1/auth/otp/verify",
		`{"phone":"`+phone+`","code":"`+issued.DevCode+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: got %d want 200: %s", rec.Code, rec.Body)
	}
	var first tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatalf("verify body: %v", err)
	}
	if first.User.Role != RoleManager || first.User.ID == "" {
		t.Fatalf("bootstrap: role=%q id=%q want manager + non-empty id", first.User.Role, first.User.ID)
	}
	if first.AccessToken == "" || first.RefreshToken == "" || first.ExpiresIn <= 0 {
		t.Fatalf("token pair incomplete: %+v", first)
	}

	// 5. Second registration is resident — bootstrap applies once.
	second := requestOTPAndVerify(t, e, "09120000002")
	if second.User.Role != RoleResident {
		t.Fatalf("second user role=%q want resident", second.User.Role)
	}

	// 6. GET /auth/me with the access token returns the same identity plus
	// its (empty until US2) building scope.
	rec = e.request(t, http.MethodGet, "/api/v1/auth/me", "", first.AccessToken)
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

	// 7. Login was audited (T025): one user.login entry per verification.
	logins := 0
	for _, r := range e.auditor.records {
		if r.action == "user.login" && r.actorID != nil && r.objectType == "user" {
			logins++
		}
	}
	if logins != 2 {
		t.Fatalf("user.login audit entries=%d want 2", logins)
	}
}

func TestIntegration_RefreshRotation(t *testing.T) {
	e := newAuthEnv(t)
	pair := requestOTPAndVerify(t, e, "09130000001")

	// Rotate: old refresh → new pair.
	rec := postJSON(t, e, "/api/v1/auth/refresh", `{"refresh_token":"`+pair.RefreshToken+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh: got %d want 200: %s", rec.Code, rec.Body)
	}
	var rotated tokenPairResp
	if err := json.Unmarshal(rec.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("refresh body: %v", err)
	}
	if rotated.RefreshToken == pair.RefreshToken || rotated.AccessToken == "" {
		t.Fatalf("rotation must mint fresh tokens: %+v", rotated)
	}

	// The new access token authenticates.
	if rec := e.request(t, http.MethodGet, "/api/v1/auth/me", "", rotated.AccessToken); rec.Code != http.StatusOK {
		t.Fatalf("me after refresh: got %d want 200", rec.Code)
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
	pair := requestOTPAndVerify(t, e, "09140000001")

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

	// Unauthenticated /auth/me.
	if rec := e.request(t, http.MethodGet, "/api/v1/auth/me", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated me: got %d want 401", rec.Code)
	}
}
