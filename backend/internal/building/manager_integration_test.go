package building

// 002-multi-manager-support US3 integration suite: per-building manager
// grant/revoke through the real HTTP handlers against a migrated PostgreSQL
// (harness in building_integration_test.go). Covers the grant round-trip,
// duplicate 409 with the contractual Persian message, self-removal 400,
// last-manager 409 (checked first, so a LONE manager removing themself gets
// the precise "at least one manager must remain" answer, while a manager in
// a multi-manager building gets the handover 400), the resident→manager role
// flip, unknown/inactive/superadmin refusals, the 403 cross-building matrix,
// and the building.manager_granted / building.manager_revoked audit rows.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"hamsa/internal/auth"
)

// seedPhone inserts a user with a fixed phone and returns (id, access_token).
func (e *bEnv) seedPhone(t *testing.T, role, phone string) (uuid.UUID, string) {
	t.Helper()
	id := uuid.New()
	if err := e.exec(
		`INSERT INTO users (id, phone, role, name) VALUES (?, ?, ?, ?)`,
		id, phone, role, "مدیر تست",
	); err != nil {
		t.Fatalf("seed user %s: %v", phone, err)
	}
	token, _, err := e.tokens.IssueAccessToken(id)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return id, token
}

// newBuildingAs creates a building via POST and returns its id.
func newBuildingAs(t *testing.T, e *bEnv, token, name string) uuid.UUID {
	t.Helper()
	res := e.request(t, "POST", "/api/v1/buildings",
		fmt.Sprintf(`{"name":%q,"floor_count":3}`, name), token)
	if res.Code != http.StatusCreated {
		t.Fatalf("create building: got %d want 201: %s", res.Code, res.Body)
	}
	id, _ := uuid.Parse(bodyJSON(t, res)["id"].(string))
	return id
}

func managersPath(bid uuid.UUID) string {
	return "/api/v1/buildings/" + bid.String() + "/managers"
}

// errMessage extracts error.message from an envelope body.
func errMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	return bodyJSON(t, rec)["error"].(map[string]any)["message"].(string)
}

func TestUS3_ManagerGrantLifecycle(t *testing.T) {
	env := newBuildingEnv(t)
	const (
		phoneM1   = "09160000001"
		phoneM2   = "09160000002"
		phoneRes  = "09160000003"
		phoneSup  = "09160000004"
		phoneDead = "09160000005"
	)
	m1ID, m1Tok := env.seedPhone(t, auth.RoleManager, phoneM1)
	m2ID, _ := env.seedPhone(t, auth.RoleManager, phoneM2)
	resID, _ := env.seedPhone(t, auth.RoleResident, phoneRes)
	env.seedPhone(t, auth.RoleSuperAdmin, phoneSup)

	bid := newBuildingAs(t, env, m1Tok, "برج مدیران")

	// --- GET: creator listed, full row shape.
	rec := env.request(t, "GET", managersPath(bid), "", m1Tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list managers: got %d want 200: %s", rec.Code, rec.Body)
	}
	items := bodyJSON(t, rec)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("initial managers=%d want 1", len(items))
	}
	row0 := items[0].(map[string]any)
	if row0["user_id"] != m1ID.String() || row0["phone"] != phoneM1 ||
		row0["name"] == "" || row0["granted_at"] == "" || row0["role"] != auth.RoleManager {
		t.Fatalf("creator row incomplete: %v", row0)
	}

	// --- POST manager: 201 with resulting role.
	rec = env.request(t, "POST", managersPath(bid), `{"phone":"`+phoneM2+`"}`, m1Tok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("grant m2: got %d want 201: %s", rec.Code, rec.Body)
	}
	granted := bodyJSON(t, rec)
	if granted["user_id"] != m2ID.String() || granted["role"] != auth.RoleManager {
		t.Fatalf("grant payload: %v", granted)
	}

	// --- POST duplicate: 409 with the contractual message.
	rec = env.request(t, "POST", managersPath(bid), `{"phone":"`+phoneM2+`"}`, m1Tok)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate grant: got %d want 409: %s", rec.Code, rec.Body)
	}
	if msg := errMessage(t, rec); msg != "این مدیر از قبل دسترسی دارد." {
		t.Fatalf("duplicate message=%q", msg)
	}

	// --- POST resident phone: 201 + role flip to manager (FR-013).
	rec = env.request(t, "POST", managersPath(bid), `{"phone":"`+phoneRes+`"}`, m1Tok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("grant resident: got %d want 201: %s", rec.Code, rec.Body)
	}
	if bodyJSON(t, rec)["role"] != auth.RoleManager {
		t.Fatalf("resident grant response role=%v want manager", bodyJSON(t, rec)["role"])
	}
	var flipped string
	if err := env.gormDB.Raw(`SELECT role FROM users WHERE id = ?`, resID).Scan(&flipped).Error; err != nil || flipped != auth.RoleManager {
		t.Fatalf("role flip: role=%q err=%v want manager", flipped, err)
	}

	// --- POST invalid phone format: 400.
	if rec := env.request(t, "POST", managersPath(bid), `{"phone":"123"}`, m1Tok); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad phone format: got %d want 400", rec.Code)
	}

	// --- POST unknown / inactive / superadmin targets.
	if rec := env.request(t, "POST", managersPath(bid), `{"phone":"09199999999"}`, m1Tok); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown phone: got %d want 404: %s", rec.Code, rec.Body)
	}
	deadID := uuid.New()
	if err := env.exec(`INSERT INTO users (id, phone, role, name, is_active) VALUES (?, ?, ?, 'خاموش', FALSE)`,
		deadID, phoneDead, auth.RoleManager); err != nil {
		t.Fatalf("seed inactive: %v", err)
	}
	if rec := env.request(t, "POST", managersPath(bid), `{"phone":"`+phoneDead+`"}`, m1Tok); rec.Code != http.StatusNotFound {
		t.Fatalf("inactive phone: got %d want 404: %s", rec.Code, rec.Body)
	}
	if rec := env.request(t, "POST", managersPath(bid), `{"phone":"`+phoneSup+`"}`, m1Tok); rec.Code != http.StatusBadRequest {
		t.Fatalf("superadmin target: got %d want 400: %s", rec.Code, rec.Body)
	}

	// --- DELETE another manager: 204, list shrinks.
	if rec := env.request(t, "DELETE", managersPath(bid)+"/"+m2ID.String(), "", m1Tok); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke m2: got %d want 204: %s", rec.Code, rec.Body)
	}
	rec = env.request(t, "GET", managersPath(bid), "", m1Tok)
	if n := len(bodyJSON(t, rec)["items"].([]any)); n != 2 {
		t.Fatalf("after revoke managers=%d want 2", n)
	}

	// --- DELETE never-granted user: 404.
	if rec := env.request(t, "DELETE", managersPath(bid)+"/"+uuid.NewString(), "", m1Tok); rec.Code != http.StatusNotFound {
		t.Fatalf("revoke absent: got %d want 404: %s", rec.Code, rec.Body)
	}

	// --- Audit rows for the 2 successful grants + 1 revoke (SC-004).
	var grantedRows, revokedRows int64
	if err := env.gormDB.Raw(`SELECT count(*) FROM audit_logs WHERE action='building.manager_granted' AND object_type='building' AND object_id = ?`, bid).Scan(&grantedRows).Error; err != nil || grantedRows != 2 {
		t.Fatalf("granted audit rows=%d err=%v want 2", grantedRows, err)
	}
	if err := env.gormDB.Raw(`SELECT count(*) FROM audit_logs WHERE action='building.manager_revoked' AND object_type='building' AND object_id = ?`, bid).Scan(&revokedRows).Error; err != nil || revokedRows != 1 {
		t.Fatalf("revoked audit rows=%d err=%v want 1", revokedRows, err)
	}
	var gRaw, rRaw string
	env.gormDB.Raw(`SELECT after_value::text FROM audit_logs WHERE action='building.manager_granted' AND object_id = ? ORDER BY id DESC LIMIT 1`, bid).Scan(&gRaw)
	env.gormDB.Raw(`SELECT before_value::text FROM audit_logs WHERE action='building.manager_revoked' AND object_id = ? ORDER BY id DESC LIMIT 1`, bid).Scan(&rRaw)
	var gPayload, rPayload map[string]any
	if err := json.Unmarshal([]byte(gRaw), &gPayload); err != nil || gPayload["user_id"] != resID.String() || gPayload["role"] != auth.RoleManager {
		t.Fatalf("grant audit after=%s err=%v", gRaw, err)
	}
	if err := json.Unmarshal([]byte(rRaw), &rPayload); err != nil || rPayload["user_id"] != m2ID.String() {
		t.Fatalf("revoke audit before=%s err=%v", rRaw, err)
	}
}

func TestUS3_SelfRemovalAndLastManager(t *testing.T) {
	env := newBuildingEnv(t)
	m1ID, m1Tok := env.seedPhone(t, auth.RoleManager, "09170000001")
	m9ID, m9Tok := env.seedPhone(t, auth.RoleManager, "09170000009")
	bid := newBuildingAs(t, env, m1Tok, "برج تحویل")

	// Two managers: creator removing their own grant → 400 (handover first).
	if rec := env.request(t, "POST", managersPath(bid), `{"phone":"09170000009"}`, m1Tok); rec.Code != http.StatusCreated {
		t.Fatalf("grant m9: %d %s", rec.Code, rec.Body)
	}
	if rec := env.request(t, "DELETE", managersPath(bid)+"/"+m1ID.String(), "", m1Tok); rec.Code != http.StatusBadRequest {
		t.Fatalf("self-removal (non-last): got %d want 400: %s", rec.Code, rec.Body)
	}

	// Handover: m9 revokes the creator; now m9 is the last manager.
	if rec := env.request(t, "DELETE", managersPath(bid)+"/"+m1ID.String(), "", m9Tok); rec.Code != http.StatusNoContent {
		t.Fatalf("m9 revokes creator: got %d want 204: %s", rec.Code, rec.Body)
	}

	// Lone manager attempting to remove the only grant (their own) → last-
	// manager 409 (SC-003: the building never reaches zero managers).
	if rec := env.request(t, "DELETE", managersPath(bid)+"/"+m9ID.String(), "", m9Tok); rec.Code != http.StatusConflict {
		t.Fatalf("last-manager removal: got %d want 409: %s", rec.Code, rec.Body)
	}
	var n int64
	env.gormDB.Raw(`SELECT count(*) FROM user_buildings WHERE building_id = ?`, bid).Scan(&n)
	if n != 1 {
		t.Fatalf("building lost its last manager: grants=%d want 1", n)
	}
}

func TestUS3_CrossBuildingAuthorization(t *testing.T) {
	env := newBuildingEnv(t)
	_, mA := env.seedPhone(t, auth.RoleManager, "09180000001")
	newBuildingAs(t, env, mA, "برج ای")

	_, mB := env.seedPhone(t, auth.RoleManager, "09180000002")
	bid := newBuildingAs(t, env, mB, "برج بی")

	// Manager of "ای" only: 403 on all three sub-routes of "بی" (SC-005).
	target := uuid.NewString()
	for _, tc := range []struct {
		name, method, path, body string
	}{
		{"list", "GET", managersPath(bid), ""},
		{"add", "POST", managersPath(bid), `{"phone":"09180000001"}`},
		{"remove", "DELETE", managersPath(bid) + "/" + target, ""},
	} {
		if rec := env.request(t, tc.method, tc.path, tc.body, mA); rec.Code != http.StatusForbidden {
			t.Fatalf("%s cross-building: got %d want 403: %s", tc.name, rec.Code, rec.Body)
		}
	}
}
