package building

// US3 integration suite (T035): drives the real Gin handlers against a
// throwaway PostgreSQL schema with the real migrations applied — same harness
// as building_integration_test.go (shared bEnv). Skips when neither Docker nor
// a reachable dev PostgreSQL exists (set HAMSA_TEST_DSN to force).

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// validNationalIDFromBase derives the check digit for the first 9 digits using
// the Iranian national-ID checksum (data-model.md "persons.national_id").
func validNationalIDFromBase(base string) string {
	sum := 0
	for i, r := range base {
		sum += int(r-'0') * (10 - i)
	}
	rem := sum % 11
	check := rem
	if rem >= 2 {
		check = 11 - rem
	}
	return fmt.Sprintf("%s%d", base, check)
}

func createBuildingUnit(t *testing.T, env *bEnv, token, unitNumber string) (string, string) {
	t.Helper()
	rec := env.request(t, http.MethodPost, "/api/v1/buildings", `{"name":"ساختمان تست"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create building: %d %s", rec.Code, rec.Body.String())
	}
	buildingID, _ := bodyJSON(t, rec)["id"].(string)
	rec = env.request(t, http.MethodPost, "/api/v1/buildings/"+buildingID+"/units",
		fmt.Sprintf(unitPayload, unitNumber, 1), token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create unit: %d %s", rec.Code, rec.Body.String())
	}
	unitID, _ := bodyJSON(t, rec)["id"].(string)
	return buildingID, unitID
}

func createPersonRow(t *testing.T, env *bEnv, token, buildingID, name, nationalID string) string {
	t.Helper()
	payload := fmt.Sprintf(`{"full_name":%q,"phone":"09121112222","national_id":%q}`, name, nationalID)
	rec := env.request(t, http.MethodPost, "/api/v1/buildings/"+buildingID+"/persons", payload, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create person %s: %d %s", name, rec.Code, rec.Body.String())
	}
	id, _ := bodyJSON(t, rec)["id"].(string)
	return id
}

func TestUS3_OccupancyLifecycle(t *testing.T) {
	env := newBuildingEnv(t)
	_, mgr := env.seedUser(t, "manager")
	buildingID, unitID := createBuildingUnit(t, env, mgr, "1")

	personA := createPersonRow(t, env, mgr, buildingID, "علی رضایی", validNationalIDFromBase("012345678"))
	personB := createPersonRow(t, env, mgr, buildingID, "مریم کریمی", validNationalIDFromBase("123456789"))

	// Add tenant A on unit 1 starting 2026-01-01.
	rec := env.request(t, http.MethodPost, "/api/v1/units/"+unitID+"/occupancies",
		fmt.Sprintf(`{"person_id":%q,"relationship":"tenant","start_date":"2026-01-01"}`, personA), mgr)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add occupancy A: %d %s", rec.Code, rec.Body.String())
	}
	occA, _ := bodyJSON(t, rec)["id"].(string)

	// Occupant-count history: 3 as of 2026-01-01, 5 as of 2026-03-01.
	for _, tc := range []struct{ count, from string }{
		{"3", "2026-01-01"}, {"5", "2026-03-01"},
	} {
		rec = env.request(t, http.MethodPost, "/api/v1/units/"+unitID+"/occupant-count",
			fmt.Sprintf(`{"occupant_count":%s,"effective_from":%q}`, tc.count, tc.from), mgr)
		if rec.Code != http.StatusCreated {
			t.Fatalf("record occupant count %s: %d %s", tc.from, rec.Code, rec.Body.String())
		}
	}

	// Full history is preserved, newest first.
	rec = env.request(t, http.MethodGet, "/api/v1/units/"+unitID+"/occupant-count", "", mgr)
	if rec.Code != http.StatusOK {
		t.Fatalf("occupant-count history: %d %s", rec.Code, rec.Body.String())
	}
	counts := bodyJSON(t, rec)["items"].([]any)
	if len(counts) != 2 {
		t.Fatalf("want 2 occupant-count rows, got %d", len(counts))
	}
	first := counts[0].(map[string]any)
	if first["effective_from"] != "2026-03-01" || first["occupant_count"].(float64) != 5 {
		t.Fatalf("newest count row wrong: %v", first)
	}

	// As-of reads (engine input): 2026-02-15 → 3; 2026-04-01 → 5.
	repo := NewRepository(env.gormDB)
	unitUUID, err := uuid.Parse(unitID)
	if err != nil {
		t.Fatalf("parse unit id: %v", err)
	}
	asOfFeb, err := repo.OccupantCountAsOf(context.Background(), unitUUID, time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC))
	if err != nil || asOfFeb != 3 {
		t.Fatalf("as-of 2026-02-15: got %d, err %v", asOfFeb, err)
	}
	asOfApr, err := repo.OccupantCountAsOf(context.Background(), unitUUID, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || asOfApr != 5 {
		t.Fatalf("as-of 2026-04-01: got %d, err %v", asOfApr, err)
	}

	// Replacing the tenant: new active tenant closes the previous one
	// (FR-007) — A end-dated 2026-03-31, B active, both rows preserved.
	rec = env.request(t, http.MethodPost, "/api/v1/units/"+unitID+"/occupancies",
		fmt.Sprintf(`{"person_id":%q,"relationship":"tenant","start_date":"2026-04-01"}`, personB), mgr)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add occupancy B: %d %s", rec.Code, rec.Body.String())
	}
	occB, _ := bodyJSON(t, rec)["id"].(string)

	rec = env.request(t, http.MethodGet, "/api/v1/units/"+unitID+"/occupancies", "", mgr)
	if rec.Code != http.StatusOK {
		t.Fatalf("occupancy history: %d %s", rec.Code, rec.Body.String())
	}
	occs := bodyJSON(t, rec)["items"].([]any)
	if len(occs) != 2 {
		t.Fatalf("want 2 occupancy rows (history preserved), got %d", len(occs))
	}
	byID := map[string]map[string]any{}
	for _, o := range occs {
		m := o.(map[string]any)
		byID[m["id"].(string)] = m
	}
	a := byID[occA]
	if a["end_date"] != "2026-03-31" || a["is_active"] != false {
		t.Fatalf("occupancy A not closed correctly: %v", a)
	}
	b := byID[occB]
	if b["end_date"] != nil || b["is_active"] != true {
		t.Fatalf("occupancy B should be active: %v", b)
	}

	// Explicit end-date via PATCH preserves the row (never deleted).
	rec = env.request(t, http.MethodPatch, "/api/v1/occupancies/"+occB,
		`{"end_date":"2026-05-01"}`, mgr)
	if rec.Code != http.StatusOK {
		t.Fatalf("end occupancy B: %d %s", rec.Code, rec.Body.String())
	}
	if bodyJSON(t, rec)["end_date"] != "2026-05-01" {
		t.Fatalf("end_date not applied: %s", rec.Body.String())
	}

	// Resident change is audited (resident.changed, FR-038).
	var audited int64
	if err := env.gormDB.Raw(
		`SELECT count(*) FROM audit_logs WHERE action = 'resident.changed' AND object_id = ?::uuid`,
		unitID).Scan(&audited).Error; err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if audited < 2 {
		t.Fatalf("want ≥ 2 resident.changed audit rows, got %d", audited)
	}
}

func TestUS3_ValidationErrors(t *testing.T) {
	env := newBuildingEnv(t)
	_, mgr := env.seedUser(t, "manager")
	buildingID, unitID := createBuildingUnit(t, env, mgr, "1")
	personID := createPersonRow(t, env, mgr, buildingID, "سعید احمدی", validNationalIDFromBase("456789123"))

	cases := []struct {
		name, method, path, body string
		wantCode                 int
		wantMsgPart              string
	}{
		{
			name: "invalid national id checksum", method: http.MethodPost,
			path:     "/api/v1/buildings/" + buildingID + "/persons",
			body:     `{"full_name":"بد کنترل","national_id":"0123456780"}`,
			wantCode: http.StatusBadRequest, wantMsgPart: "کد ملی",
		},
		{
			name: "missing person name", method: http.MethodPost,
			path:     "/api/v1/buildings/" + buildingID + "/persons",
			body:     `{"full_name":" "}`,
			wantCode: http.StatusBadRequest, wantMsgPart: "نام",
		},
		{
			name: "bad relationship", method: http.MethodPost,
			path:     "/api/v1/units/" + unitID + "/occupancies",
			body:     fmt.Sprintf(`{"person_id":%q,"relationship":"friend","start_date":"2026-01-01"}`, personID),
			wantCode: http.StatusBadRequest, wantMsgPart: "نسبت",
		},
		{
			name: "end before start", method: http.MethodPost,
			path:     "/api/v1/units/" + unitID + "/occupancies",
			body:     fmt.Sprintf(`{"person_id":%q,"relationship":"owner","start_date":"2026-01-10","end_date":"2026-01-01"}`, personID),
			wantCode: http.StatusBadRequest, wantMsgPart: "تاریخ",
		},
		{
			name: "negative occupant count", method: http.MethodPost,
			path:     "/api/v1/units/" + unitID + "/occupant-count",
			body:     `{"occupant_count":-1,"effective_from":"2026-01-01"}`,
			wantCode: http.StatusBadRequest, wantMsgPart: "تعداد ساکن",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.request(t, tc.method, tc.path, tc.body, mgr)
			if rec.Code != tc.wantCode {
				t.Fatalf("code = %d, want %d (%s)", rec.Code, tc.wantCode, rec.Body.String())
			}
			msg, _ := bodyJSON(t, rec)["error"].(map[string]any)["message"].(string)
			if !strings.Contains(msg, tc.wantMsgPart) {
				t.Fatalf("message %q does not contain %q", msg, tc.wantMsgPart)
			}
		})
	}
}

func TestUS3_ManagerScopeAndAuth(t *testing.T) {
	env := newBuildingEnv(t)
	_, mgr := env.seedUser(t, "manager")
	_, otherTok := env.seedUser(t, "manager") // no grant on this building
	buildingID, unitID := createBuildingUnit(t, env, mgr, "2")
	personID := createPersonRow(t, env, mgr, buildingID, "ساکن دیگر", validNationalIDFromBase("234567890"))

	// Second manager gets 403 regardless of resource existence.
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/units/" + unitID + "/occupancies"},
		{http.MethodPost, "/api/v1/units/" + unitID + "/occupancies"},
		{http.MethodGet, "/api/v1/units/" + unitID + "/occupant-count"},
		{http.MethodGet, "/api/v1/buildings/" + buildingID + "/persons"},
	} {
		rec := env.request(t, tc.method, tc.path, `{}`, otherTok)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s as ungranted manager: %d, want 403", tc.method, tc.path, rec.Code)
		}
	}

	// Non-manager token is rejected on manager-only routes.
	_, res := env.seedUser(t, "resident")
	rec := env.request(t, http.MethodGet, "/api/v1/buildings/"+buildingID+"/persons", "", res)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("resident on manager route: %d, want 403", rec.Code)
	}

	// Unauthenticated requests are rejected.
	rec = env.request(t, http.MethodGet, "/api/v1/units/"+unitID+"/occupancies", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: %d, want 401", rec.Code)
	}

	// Sanity: the granted manager can read (person exists for later steps).
	rec = env.request(t, http.MethodGet, "/api/v1/buildings/"+buildingID+"/persons", "", mgr)
	if rec.Code != http.StatusOK {
		t.Fatalf("manager person list: %d %s", rec.Code, rec.Body.String())
	}
	items := bodyJSON(t, rec)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != personID {
		t.Fatalf("person list mismatch: %s", rec.Body.String())
	}
}
