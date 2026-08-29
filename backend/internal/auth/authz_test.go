package auth

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hamsa/internal/platform/httpx"
)

// US1 authorization suite (T022): role gates and object-level isolation
// (quickstart Scenario 7). Resident A must never reach resident B's objects
// or any manager route; 403 regardless of resource existence
// (contracts/api.md).
//
// The unit tables ship with the US2/US3 migrations (persons/occupancies in
// migration 0003), which newAuthEnv applies; the fixture below only seeds
// rows for the ScopeResolver.

func newAuthzEnv(t *testing.T) (*authEnv, string, string) {
	t.Helper()
	e := newAuthEnv(t)
	return e, "", ""
}

func seedUser(t *testing.T, e *authEnv, phone, role string) (uuid.UUID, string) {
	t.Helper()
	u := &User{ID: uuid.New(), Phone: phone, Role: role, IsActive: true}
	if err := e.users.Create(context.Background(), u); err != nil {
		t.Fatalf("seed user %s: %v", phone, err)
	}
	access, _, err := e.tokens.IssueAccessToken(u.ID)
	if err != nil {
		t.Fatalf("issue access for %s: %v", phone, err)
	}
	return u.ID, access
}

func TestIntegration_ManagerRouteRejectsResident(t *testing.T) {
	e, _, _ := newAuthzEnv(t)
	authMW := Authenticate(e.tokens, e.users)

	managerRouteCalled := false
	e.engine.GET("/api/v1/manager-only", authMW, RequireRole(RoleManager), func(c *gin.Context) {
		managerRouteCalled = true
		c.Status(http.StatusOK)
	})

	_, managerAccess := seedUser(t, e, "09150000001", RoleManager)
	_, residentAccess := seedUser(t, e, "09150000002", RoleResident)

	// Positive control: manager passes.
	if rec := e.request(t, http.MethodGet, "/api/v1/manager-only", "", managerAccess); rec.Code != http.StatusOK {
		t.Fatalf("manager on manager route: got %d want 200", rec.Code)
	}

	managerRouteCalled = false
	// Resident → manager route → 403 FORBIDDEN (contracts/api.md).
	if rec := e.request(t, http.MethodGet, "/api/v1/manager-only", "", residentAccess); rec.Code != http.StatusForbidden {
		t.Fatalf("resident on manager route: got %d want 403", rec.Code)
	}
	if managerRouteCalled {
		t.Fatalf("resident request reached the handler body")
	}

	// No token → 401.
	if rec := e.request(t, http.MethodGet, "/api/v1/manager-only", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: got %d want 401", rec.Code)
	}
}

func TestIntegration_ObjectLevelResidentIsolation(t *testing.T) {
	e, _, _ := newAuthzEnv(t)
	ctx := context.Background()
	scopes := NewScopeResolver(e.gormDB)

	_, residentA := seedUser(t, e, "09160000001", RoleResident)
	_, residentB := seedUser(t, e, "09160000002", RoleResident)

	personA, personB := uuid.New(), uuid.New()
	unitA, unitB := uuid.New(), uuid.New()
	fixtureBuilding := uuid.New()
	if err := e.exec(
		`INSERT INTO buildings (id, name) VALUES ($1, $2)`,
		fixtureBuilding, "ساختمان تست مجوز",
	); err != nil {
		t.Fatalf("seed building: %v", err)
	}
	for _, unit := range []uuid.UUID{unitA, unitB} {
		if err := e.exec(
			`INSERT INTO units (id, building_id, number, area_m2) VALUES ($1, $2, $3, 100)`,
			unit, fixtureBuilding, fmt.Sprintf("%d", unit.ID()%1000),
		); err != nil {
			t.Fatalf("seed unit: %v", err)
		}
	}

	for _, row := range []struct {
		id    uuid.UUID
		phone string
	}{{personA, "09160000001"}, {personB, "09160000002"}} {
		if err := e.exec(`INSERT INTO persons (id, building_id, full_name, phone) VALUES ($1, $2, $3, $4)`,
			row.id, fixtureBuilding, "ساکن تست "+row.phone, row.phone); err != nil {
			t.Fatalf("seed person: %v", err)
		}
	}
	for _, occ := range []struct {
		person, unit uuid.UUID
	}{
		{personA, unitA}, {personB, unitB},
	} {
		if err := e.exec(
			`INSERT INTO occupancies (person_id, unit_id, relationship, start_date, end_date)
			 VALUES ($1, $2, 'tenant', CURRENT_DATE, NULL)`,
			occ.person, occ.unit,
		); err != nil {
			t.Fatalf("seed occupancy: %v", err)
		}
	}

	// The object-level guard every resident invoice/unit route applies from
	// US4 onward: resolve scope server-side, 403 without touching storage.
	guard := func(c *gin.Context) {
		u := CurrentUser(c)
		unitID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			httpx.WriteError(c, httpx.BadRequest("شناسه نامعتبر است."))
			return
		}
		ok, err := scopes.CanResidentAccessUnit(ctx, u.Phone, unitID)
		if err != nil {
			httpx.WriteError(c, httpx.Internal("خطای داخلی."))
			return
		}
		if !ok {
			httpx.WriteError(c, httpx.Forbidden("دسترسی غیرمجاز است.")) // 403 even when the object exists for someone else
			return
		}
		c.Status(http.StatusOK)
	}
	e.engine.GET("/api/v1/invoices/:id", Authenticate(e.tokens, e.users), guard)

	type step struct {
		name   string
		token  string
		target string
		want   int
	}
	for _, s := range []step{
		{"A reads own unit's invoice", residentA, unitA.String(), http.StatusOK},
		{"A reads B's invoice", residentA, unitB.String(), http.StatusForbidden},
		{"B reads A's invoice", residentB, unitA.String(), http.StatusForbidden},
		{"A reads nonexistent invoice", residentA, uuid.New().String(), http.StatusForbidden},
	} {
		t.Run(s.name, func(t *testing.T) {
			rec := e.request(t, http.MethodGet, "/api/v1/invoices/"+s.target, "", s.token)
			if rec.Code != s.want {
				t.Fatalf("got %d want %d", rec.Code, s.want)
			}
		})
	}
}
