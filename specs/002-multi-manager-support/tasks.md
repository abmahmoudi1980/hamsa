# Tasks: Multi-Manager Support with Superadmin Bootstrap

**Input**: Design documents from `specs/002-multi-manager-support/`

**Prerequisites**: plan.md, spec.md, research.md (R1–R14), data-model.md, contracts/api.md, quickstart.md

**Tests**: INCLUDED — the feature acceptance criteria explicitly require new
backend unit + integration tests and Flutter widget tests. Test tasks are
written to fail before their story's implementation tasks.

**Organization**: Tasks grouped by user story (US1–US5 from spec.md). The
schema change and the invite-role plumbing serve US1 and US2 identically, so
they sit in Setup/Foundational; each story phase below is an independently
testable increment.

**Machine constraints**: run Flutter with `--no-pub` always (pub get broken on
this machine). Do NOT run formatters or golangci-lint; only the acceptance
gates (`go build/vet/test`, `flutter analyze/test --no-pub`).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallelizable (different files, no incomplete dependency)
- **[US#]**: user story from spec.md
- Paths are repo-relative from `D:\apps\hamsa`

---

## Phase 1: Setup (Schema)

**Purpose**: The single migration 0010 that both stories and the building
module depend on. Follows research R2/R3 and data-model.md.

- [ ] T001 Create `backend/migrations/0010_invite_roles.up.sql`: inside `BEGIN;…COMMIT;` run `ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'superadmin';` and `ALTER TABLE invite_codes ADD COLUMN role user_role NOT NULL DEFAULT 'resident';` — the new enum value MUST NOT be referenced by any statement in this transaction (no backfill); comment the deliberate no-superadmin-backfill (research R2)
- [ ] T002 [P] Create `backend/migrations/0010_invite_roles.down.sql`: `ALTER TABLE invite_codes DROP COLUMN role;` only, with a comment that enum values cannot be dropped in place and `'superadmin'` intentionally remains (research R2)
- [ ] T003 Verify on a throwaway DB: `docker compose up -d`, then `cd backend && go run ./cmd/migrate up` and confirm via psql that `invite_codes.role` exists with default `'resident'` and `SELECT enum_range(NULL::user_role)` lists `manager,resident,superadmin`; then `go run ./cmd/migrate down 1 && go run ./cmd/migrate up` round-trips cleanly

**Checkpoint**: Schema ready; server boots with 0010 applied.

---

## Phase 2: Foundational (Invite-role plumbing — blocks US1/US2)

**Purpose**: Role-carrying invites and the compile-consistent handler
adaptation. Both stories consume exactly this; keep the whole Go module
compiling and the existing suites green at every task boundary.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T004 Add `RoleSuperAdmin = "superadmin"` constant (with doc comment: exactly one, created only by `/auth/setup`, never a building-management participant) to `backend/internal/auth/user.go` beside `RoleManager`/`RoleResident`
- [ ] T005 [P] Write FAILING unit tests in `backend/internal/auth/invite_test.go` (existing fakeInviteStore + fakeClock style): `Issue` persists the requested role on the `InviteCode`; `Redeem` returns the invite's role; an invite row without a role (legacy fixture) behaves as `resident`; round-trip resident and manager; existing phone-binding/single-use/expiry tests keep passing. Run `cd backend && go test ./internal/auth/ -run Invite` and confirm the new tests FAIL (compile error counts as red)
- [ ] T006 Implement role plumbing in `backend/internal/auth/invite.go`: add `Role string \`gorm:"column:role"\`` to `InviteCode`; change `Issue(ctx, phone string, role string, createdBy *uuid.UUID)` storing the role verbatim (validation is a handler concern — research R4); change `Redeem(ctx, phone, code) (string, error)` returning the redeemed invite's role (research R3); keep hashCode/constant-time/optimistic-consume logic byte-identical
- [ ] T007 Update `register` in `backend/internal/auth/handlers.go`: capture `role, err := h.Invites.Redeem(...)`; new-user branch creates `&User{... Role: role ...}`; EXISTING-user branch untouched (password/name only — structural no-promotion, research R5); update the `register` doc comment to state invite-role semantics; make T005 unit tests green (`go test ./internal/auth/`)
- [ ] T008 In `backend/internal/auth/handlers.go`: widen the invite route to `authed.POST("/invites", RequireRole(RoleManager, RoleSuperAdmin), h.createInvite)`; extend `createInviteReq` with `Role string \`json:"role"\``; default empty → `RoleResident`; whitelist {`resident`,`manager`} else `httpx.BadRequest("نقش دعوت نامعتبر است.")` (rejects `superadmin` and garbage alike); pass role to `Issue`; respond `201 {code, role, expires_in_days: 7}` (clean cutover — old shape removed); append best-effort audit via the existing `LoginAuditor` interface: `Append(ctx, &manager.ID, "invite.issued", "invite", nil, nil, gin.H{"phone": req.Phone, "role": role})` with the error ignored like `user.login` (research R10)
- [ ] T009 Fix every remaining `Issue`/`Redeem` callers so the module compiles (`cd backend && go build ./... && go vet ./...`), and run `go test ./internal/...` — all existing suites green

**Checkpoint**: Invites carry roles end-to-end; nothing user-visible changed for residents; US1/US2 unblocked.

---

## Phase 3: User Story 1 — Default superadmin bootstraps the deployment (Priority: P1) 🎯 MVP

**Goal**: First-run setup produces the single permanent superadmin; it can create building managers; no second superadmin, demotion, or deletion path; no invite ever promotes an existing user.

**Independent Test**: Fresh DB → setup → `/auth/me` shows `role: superadmin`, empty buildings → superadmin issues manager invite → phone registers as manager → second setup attempt 409. (spec US1 scenarios 1–4; SC-001 steps 1–2, SC-002, SC-005 half 2)

### Tests for User Story 1 ⚠️

- [ ] T010 [P] [US1] Add FAILING integration tests to `backend/internal/auth/auth_integration_test.go` (existing `newAuthEnv` harness, real Postgres): (a) setup on empty DB → session role `superadmin`, second setup → 409; (b) `GET /auth/me` as superadmin → `role: superadmin`, `buildings: []`; (c) superadmin `POST /auth/invites {phone, role:"manager"}` → 201 with `role` echoed → `/auth/register` that phone → login → `role: manager`; (d) no-promotion: manager invite redeemed for an EXISTING resident phone → password reset works, `/auth/me` role stays `resident`; same against the superadmin's phone → role stays `superadmin`; (e) `POST /auth/invites {role:"superadmin"}` → 400, `{role:"wizard"}` → 400, missing role → defaults `resident`; (f) audit assertion via the env's auditor records: one `invite.issued` per success with `{phone, role}`. Verify red: `go test ./internal/auth/ -run Superadmin`

### Implementation for User Story 1

- [ ] T011 [US1] Change `setup` in `backend/internal/auth/handlers.go`: create `&User{... Role: RoleSuperAdmin ...}` (was `RoleManager`) — zero-users lock, phone/password validation, and conflict messages otherwise unchanged (research R6); update the doc comment
- [ ] T012 [US1] Add the superadmin arm to `me` in `backend/internal/auth/handlers.go`: role `superadmin` → return `buildings: []` with no `user_buildings` query (research R11); manager/resident arms untouched
- [ ] T013 [US1] Run `cd backend && go test ./internal/auth/` until T010 passes fully (no skips tolerated when Postgres is up)

**Checkpoint**: SC-001 bootstrap half proven; a fresh deployment can create managers without SQL.

---

## Phase 4: User Story 2 — Managers create buildings and other users (Priority: P1)

**Goal**: Managers are self-sufficient: building creation auto-grants (existing), and managers issue manager/resident invites through the same now-role-aware endpoint; residents keep zero invite ability.

**Independent Test**: With a manager account (created via US1 chain): create building → visible in own `/auth/me`; issue manager + resident invites → registrants get those roles; resident caller of `/auth/invites` → 403. (spec US2 scenarios 1–8; SC-001 steps 3–4)

### Tests for User Story 2 ⚠️

- [ ] T014 [P] [US2] Add integration tests to `backend/internal/auth/auth_integration_test.go` (and extend `authz_test.go` for the resident case): manager (not superadmin) issues `role:"manager"` and default-role invites → registrants become manager/resident respectively; invalid role value from a manager → 400 Persian; resident token → 403 on `/auth/invites`; manager `POST /buildings` then `GET /auth/me` shows the granted building id (auto-grant regression). Verify red for any behavior not already satisfied by Phase 2/3, then green without code change OR with the T015 fix

### Implementation for User Story 2

- [ ] T015 [US2] Only if T014 exposes gaps: fix `backend/internal/auth/handlers.go` / `backend/internal/auth/invite.go` (no new surface — the manager path reuses the US1 endpoint code); re-run `go test ./internal/auth/ ./internal/building/`

**Checkpoint**: Both P1 stories work: superadmin bootstraps, managers self-serve. MVP deployable.

---

## Phase 5: User Story 3 — Grant and revoke building-manager access (Priority: P2)

**Goal**: Three per-building manager routes, each gated by the existing
`CanManagerAccess` object check, with idempotent grant (409 + exact Persian
message), resident→manager raise-only flip, self-removal 400, atomic
last-manager 409, superadmin refusal, and grant/revoke audit rows.

**Independent Test**: Two managers + one building: list → add → duplicate-409 → remove → last-manager-409 → self-400; non-granted manager gets 403 on all three. (spec US3; SC-003, SC-005)

### Tests for User Story 3 ⚠️

- [ ] T016 [P] [US3] Create FAILING `backend/internal/building/manager_integration_test.go` in the existing harness style (`building_integration_test.go` setup, real Postgres, `e.request`): (a) GET managers of owned building → creator row with `user_id, phone, name, role, granted_at`; (b) POST {phone} for an existing manager → 201 + appears in GET; (c) POST again → 409 `"این مدیر از قبل دسترسی دارد."`; (d) POST resident's phone → 201 and that user's `/auth/me` role is now `manager` (flip); (e) POST unknown/inactive phone → 404; (f) POST superadmin's phone → 400; (g) DELETE another manager → 204, count drops; (h) DELETE caller-self → 400; (i) DELETE last manager → 409; (j) manager of building C only → 403 on GET/POST/DELETE of building B's managers (3 assertions, SC-005); (k) audit: `building.manager_granted` (object_id = building, after contains user_id) and `building.manager_revoked` (before contains user_id) rows exist in `audit_logs`. Verify red: `go test ./internal/building/ -run Manager`

### Implementation for User Story 3

- [ ] T017 [US3] Add raw-table accessors to `backend/internal/building/repository.go` (existing style, no new GORM models): `ListBuildingManagers(ctx, buildingID)` → []BuildingManager via `user_buildings ⋈ users` (live rows, `ORDER BY granted_at ASC`); extend/verify `GrantManager` idempotency detection (report already-granted distinctly, reuse `pgUniqueViolation` mapping pattern); `CountBuildingManagers(ctx, buildingID)`; `RevokeManagerGuarded(ctx, userID, buildingID) (deleted bool, stillGranted bool)` implementing the single atomic statement `DELETE FROM user_buildings WHERE user_id=? AND building_id=? AND (SELECT count(*) FROM user_buildings WHERE building_id=?) > 1` plus the 0-rows existence re-check (research R9); `FindActiveUserByPhone(ctx, phone)` → id, phone, name, role via the `users` table
- [ ] T018 [US3] Add a `BuildingManager` view struct to `backend/internal/building/models.go` (`UserID, Phone, Name, Role, GrantedAt` — json tags per contracts/api.md)
- [ ] T019 [US3] Implement `ListManagers`, `GrantManagerByPhone`, `RevokeManager` on `*Service` in `backend/internal/building/service.go`: every entry first runs the existing per-building gate (`repo.CanManagerAccess` → `httpx.Forbidden(msgForbidden)` — same idiom as lines ~172); Grant: target lookup (404 `کاربری با این شماره یافت نشد.` when absent/inactive), superadmin target → `httpx.BadRequest("نمی‌توان سرپرست ارشد را مدیر ساختمان کرد.")`, duplicate → `httpx.Conflict("این مدیر از قبل دسترسی دارد.")`, then raise-only flip `UPDATE users SET role='manager' WHERE id=? AND role='resident'`; Revoke: `userId == caller.ID` → `httpx.BadRequest("برای حذف دسترسی خود ابتدا مدیر دیگری معرفی کنید.")`, guarded delete → false+granted → `httpx.Conflict("حداقل یک مدیر باید باقی بماند.")`, false+absent → `httpx.BadRequest("این مدیر دسترسی‌ای برای این ساختمان ندارد.")`; return resulting role in the grant result
- [ ] T020 [US3] Wire routes in `backend/internal/building/handlers.go` under the existing `idB := buildings.Group("/:id")`: `GET /managers`, `POST /managers` wrapped with `aud.Middleware("building.manager_granted", "building")`, `DELETE /managers/:userId` wrapped with `aud.Middleware("building.manager_revoked", "building")`; handlers use `requireManager` + `parseID`, publish object/before/after through a `setManagerAuditCtx` helper mirroring `setUnitAuditCtx`, map errors via `writeServiceErr`, respond per contracts/api.md (201 + row / 204)
- [ ] T021 [US3] Run `cd backend && go build ./... && go vet ./... && go test ./internal/...` until T016 is green with no skips

**Checkpoint**: Multi-manager per building is fully self-service.

---

## Phase 6: User Story 4 — Audit every privilege change (Priority: P3)

**Goal**: All three sensitive actions produce exactly one audit row with
actor/role/target semantics; audit failure never fails the operation.

**Independent Test**: Perform issue + grant + revoke; assert 1 row each
(quickstart §2 step 8 SQL probe). (spec US4; SC-004)

### Tests for User Story 4 ⚠️

- [ ] T022 [P] [US4] In `backend/internal/auth/auth_integration_test.go` add the best-effort guarantee: with the env's auditor configured to return an error, `POST /auth/invites` still returns 201 and the invite is stored; add a matching assertion in `backend/internal/building/manager_integration_test.go` that an audit-insert failure path (or its logged-ignore branch) does not 5xx the grant/revoke
- [ ] T023 [US4] [P] Write `backend/internal/building/manager_audit_test.go`-style payload-shape assertions (extend T016 file if simpler): `invite.issued.after_value = {phone, role}`, `building.manager_granted.after_value` contains `user_id`, `building.manager_revoked.before_value` contains `user_id`; object_type/id per data-model.md table

### Implementation for User Story 4

- [ ] T024 [US4] Only if T022/T023 expose drift: adjust the audit payloads in `backend/internal/auth/handlers.go` (createInvite) or `backend/internal/building/handlers.go` (setManagerAuditCtx) to match data-model.md exactly; no changes to `backend/internal/audit/audit.go`

**Checkpoint**: Every grant, revoke, and role invite is traceable.

---

## Phase 7: User Story 5 — Manager surfaces in the mobile app (Priority: P2)

**Goal**: Invite screen gains the ساکن/مدیر toggle (superadmin-reachable); a
new per-building managers screen lists managers, adds by phone, removes with
confirm — all Persian RTL, via the shared Dio client.

**Independent Test**: quickstart §2 steps 1–7 through the app (Chrome).
(spec US5; SC-001, SC-006)

- [ ] T025 [P] [US5] Add all new Persian strings to `mobile/lib/core/l10n/app_fa.arb` (role toggle labels ساکن/مدیر, issued-role banner lines, managers screen: title, list headers, `افزودن مدیر`, add-success/removed snackbars, remove-confirm dialog title/body, last-manager/self/conflict fallbacks) and matching getters in `mobile/lib/core/l10n/app_localizations.dart` (project's hand-maintained l10n style)
- [ ] T026 [P] [US5] Update `mobile/lib/features/auth/auth_repository.dart`: invite call sends `{phone, role}` and parses `{code, role, expires_in_days}`; keep it on `apiClientProvider`
- [ ] T027 [P] [US5] Extend `mobile/lib/features/buildings/buildings_repository.dart` with `listManagers(buildingId)`, `addManager(buildingId, phone)`, `removeManager(buildingId, userId)` (shared `apiClientProvider`; errors propagate as today for `describeError`) and a `BuildingManager` model in `mobile/lib/features/buildings/models/building.dart`
- [ ] T028 [US5] Update `mobile/lib/features/auth/screens/manager_invite_screen.dart`: `SegmentedButton<String>` (ساکن / مدیر, default ساکن), send selected role via T026, result banner names the issued role, error path unchanged (describeError)
- [ ] T029 [US5] Create `mobile/lib/features/buildings/screens/building_managers_screen.dart` (`BuildingManagersScreen({required String buildingId})`): managers list (phone, name, Jalali `granted_at` via `core/datetime/jalali`), "افزودن مدیر" phone field with existing validator, remove action per row with confirm dialog, 409/400/403 Persian server messages rendered through `describeError`; empty/loading states per project widget conventions (see `person_list_screen.dart`)
- [ ] T030 [US5] Wire navigation: add `GoRoute(path: 'buildings/:buildingId/managers', …)` under the `/manager` shell in `mobile/lib/core/router/app_router.dart`; add a managers `IconButton` (people/units row-action idiom) in `mobile/lib/features/buildings/screens/building_list_screen.dart`; extend the `_redirect` role-shell check in `mobile/lib/core/router/app_router.dart` so role `superadmin` reaches the invite screen (manager shell allowed, empty building lists; no managers action — `GET /buildings` returns nothing to grant)
- [ ] T031 [P] [US5] Add widget tests in `mobile/test/features/` (mirror existing suite layout): invite screen sends chosen `role` in the request body and banner shows issued role; managers screen renders list, add posts phone, remove requires confirm before DELETE (mock Dio adapter via the existing test harness pattern)
- [ ] T032 [US5] Run `cd mobile && flutter analyze --no-pub && flutter test --no-pub` until clean and green

**Checkpoint**: SC-001 and SC-006 demonstrable end-to-end in the app.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T033 [P] Update `specs/001-building-management-mvp/contracts/api.md`: replace the `/auth/setup`, `/auth/register`, `/auth/invites`, `/auth/me` rows and the Buildings & Units table with the delta rows from `specs/002-multi-manager-support/contracts/api.md` (living reference, research R14)
- [ ] T034 [P] Update `specs/001-building-management-mvp/data-model.md` (`invite_codes.role`, `user_role` superadmin, multi-manager note replacing one-manager-per-deployment), `spec.md` FR-036/FR-037 wording if it reads single-manager, and `plan.md` module notes
- [ ] T035 [P] Final acceptance gates: `cd backend && go build ./... && go vet ./... && go test ./internal/...` green WITH Postgres up (zero skips on new files); `cd mobile && flutter analyze --no-pub && flutter test --no-pub` clean
- [ ] T036 Run `specs/002-multi-manager-support/quickstart.md` §2 walkthrough manually (setup → superadmin → manager chain → grant/revoke protections → audit SQL probe → Flutter §3 regression probes) and record outcomes
- [ ] T037 On a dev DB only: exercise `go run ./cmd/migrate down 1` (drops `invite_codes.role`, enum value intentionally survives) and `up` again; confirm server restarts clean

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none — schema only.
- **Foundational (Phase 2)**: after T001–T003 (migration applied). BLOCKS US1/US2 (both consume role-carrying invites). Keep `go build ./...` green at each task end (T009 gate).
- **US1 (Phase 3)**: after Phase 2. Tests T010 red → T011/T012 → T013.
- **US2 (Phase 4)**: after Phase 2 (its endpoint code exists already — this phase is verification-first).
- **US3 (Phase 5)**: after Phase 1 only for `user_buildings` (unchanged) — technically parallel with US1/US2 backend work, but its grant tests exercise manager accounts best produced by the US1 chain; recommended order: US1 → US3.
- **US4 (Phase 6)**: after US1 T008 (invite.issued exists) and US3 T020 (grant/revoke exist).
- **US5 (Phase 7)**: UI tasks need the backend contracts live for manual verification; T025–T029 are file-independent and parallelizable before that.
- **Polish (Phase 8)**: all desired stories complete.

### User Story Dependencies

- **US1 ⟂ US2**: same endpoint, distinct tests; US2 adds no surface beyond US1's.
- **US3** needs ≥1 manager account (via US1/US2 or SQL fixture in tests — the T016 harness seeds users directly, so US3 is testable without US1 shipping).
- **US4** verifies side effects produced inside US1/US3 code.
- **US5** consumes US1/US2 (invite role) and US3 (managers routes).

### Within Each Story

Tests written first and verified RED → model/repo → service → handlers/routes → story gate green. No story phase is "done" while its integration tests skip on a machine with Postgres running.

### Parallel Opportunities

- Phase 1: T001 ∥ T002.
- Phase 2: T005 ∥ T004.
- Phase 5: T016 red-tests while Phase 3/4 finish (different packages: `building` vs `auth`).
- Phase 7: T025 ∥ T026 ∥ T027 (different files), then T028 ∥ T029.
- Phase 8: T033 ∥ T034 (different docs), T035 after both.
- After Foundational: US1, US2, and the US3 repository/service layers (T017/T018) touch different files and can run concurrently; handlers T020 and service T019 share `building` package compile state — serialize those two.

## Parallel Example: User Story 3

```bash
# After Foundational + Phase 1:
Task: "T016 [P] [US3] failing integration tests in backend/internal/building/manager_integration_test.go"
Task: "T018 [US3] BuildingManager view struct in backend/internal/building/models.go"
# then serially (same package):
Task: "T017 [US3] repository accessors in backend/internal/building/repository.go"
Task: "T019 [US3] service operations in backend/internal/building/service.go"
Task: "T020 [US3] routes + audit ctx in backend/internal/building/handlers.go"
```

## Parallel Example: User Story 5

```bash
Task: "T025 [US5] Persian strings in mobile/lib/core/l10n/app_fa.arb"
Task: "T026 [US5] invite role payload in mobile/lib/features/auth/auth_repository.dart"
Task: "T027 [US5] manager endpoints + model in mobile/lib/features/buildings/"
```

---

## Implementation Strategy

### MVP First (Stories US1 + US2 — the P1 pair)

1. Phase 1 (T001–T003) → Phase 2 (T004–T009)
2. Phase 3 (T010–T013) → **STOP & VALIDATE**: fresh-DB bootstrap chain per quickstart §2.1–2.3
3. Phase 4 (T014–T015) → **VALIDATE**: manager self-sufficiency; deploy/demo — a deployment can now get unlimited managers, just not yet per-building grants for pre-existing ones

### Incremental Delivery

1. + Phase 5 (US3) → per-building manager CRUD live → demo grant/revoke protections (SC-003/SC-005)
2. + Phase 6 (US4) → audit story provable (SC-004)
3. + Phase 7 (US5) → full app journey (SC-001 end-to-end, SC-006 under-a-minute)
4. + Phase 8 → docs synced, final gates, quickstart walkthrough signed off

### Parallel Team Strategy

- Dev A: Foundational → US1 → US2 (auth package owner)
- Dev B: US3 (building package) once Foundational lands; Dev C: US5 UI against contracts/api.md with mocked Dio while B implements
- Integration owner: Phase 6 + 8 after merges

---

## Notes

- Exact Persian strings are contractual (contracts/api.md) — copy them verbatim; do not paraphrase.
- Never add a demotion, superadmin-mutation, or self-revoke path — spec Non-Requirements.
- `[P]` = different files, no incomplete dependency; same-package Go edits (T017/T019/T020) serialize to keep `go build` honest.
- Commit after each task or logical group; stop at any checkpoint for independent story validation.
- Skip formatters/golangci-lint per the task constraints; the acceptance gates above are the lint budget.
