# Phase 0 Research: Multi-Manager Support with Superadmin Bootstrap

**Feature**: specs/002-multi-manager-support | **Date**: 2026-09-02

All NEEDS CLARIFICATION from the plan's Technical Context: none — every
unknown below was resolved from the codebase or reasonable defaults.

## R1. Superadmin as a value of the existing `user_role` enum

- **Decision**: `ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'superadmin'` in
  migration `0010_invite_roles`; Go gains `RoleSuperAdmin = "superadmin"`
  beside the existing constants in `auth/user.go`.
- **Rationale**: Roles are already a PG enum (`0001_core.up.sql`) resolved
  per-request from the DB (`middleware.go`: the JWT carries identity only), so
  widening the enum requires zero token/session changes and every existing
  role comparison site stays a plain string compare.
- **Alternatives considered**: separate `is_superadmin` boolean (rejected:
  forks privilege checks, invites a drift where code forgets the flag); a
  fourth table of platform admins (rejected: overkill for exactly one account,
  breaks the uniform `users.role` model the whole auth stack reads).

## R2. Enum widening inside a transaction + non-reversible down migration

- **Decision**: Migration 0010 wraps its statements as usual but never *uses*
  the new `superadmin` literal (no backfill of existing first accounts —
  spec Assumption: existing deployments keep their first account as a plain
  manager; only a *future* fresh setup produces a superadmin). The down
  migration drops `invite_codes.role` and leaves the enum value in place,
  commented.
- **Rationale**: PostgreSQL ≥ 12 permits `ALTER TYPE ... ADD VALUE` inside a
  transaction but forbids *using* the new value in that same transaction;
  deferring all superadmin writes to application code keeps one clean
  migration. An enum value cannot be dropped without rebuilding the type —
  not worth the churn on a down path.
- **Alternatives considered**: backfilling the earliest manager as superadmin
  on upgrade (rejected: violates the spec's "no retroactive superadmin"
  assumption and surprises existing deployments); two migrations (rejected:
  no dependency between the column and the value).

## R3. Invite role: schema and service shape

- **Decision**: `invite_codes` gains `role user_role NOT NULL DEFAULT
  'resident'` (same enum as `users.role` — the `DEFAULT 'resident'` literal is
  pre-existing, so R2's rule holds). `InviteCode` model gains `Role string`;
  `Issue(ctx, phone, role string, createdBy *uuid.UUID)` validates role ∈
  {resident, manager} *at the handler*, not the service (per task: manager
  authority is a route concern). `Redeem(ctx, phone, code) (string, error)`
  returns the redeemed invite's role.
- **Rationale**: Reusing `user_role` makes an invite carry exactly the role
  vocabulary accounts have — a future `superadmin` value is refused by the
  handler's whitelist, not the DB. Changing `Redeem`'s signature (vs adding a
  `Lookup`) is a clean cutover: `register` is its only caller and needs the
  role at the same instant, and a separate Lookup would introduce a
  TOCTOU window between lookup and consume.
- **Alternatives considered**: varchar column with CHECK (rejected: second
  vocabulary to keep in sync); Lookup-then-Consume two-step (rejected: race —
  the current single authoritative-latest + conditional-consume flow stays
  intact only if one call returns the role).

## R4. Who may issue which invite

- **Decision**: `/auth/invites` route middleware becomes
  `RequireRole(RoleManager, RoleSuperAdmin)` (the helper already takes a var
  set). Handler: empty role defaults to `resident`; `manager` accepted from
  either caller role; anything else → `httpx.BadRequest` with a Persian
  message. Service `Issue` stores the role verbatim.
- **Rationale**: The superadmin's one power (spec FR-003) and the manager's
  (FR-005) are the same endpoint; RequireRole's allow-set is the existing
  idiom. Value validation lives in the handler so the service stays
  authority-agnostic (task constraint: "enforce at handler level").
- **Alternatives considered**: separate `/auth/invites/manager` endpoint
  (rejected: same resource, two routes); role checks inside InviteService
  (rejected by explicit task constraint).

## R5. Register: role assignment and the no-promotion rule

- **Decision**: In `register`, `role, err := h.Invites.Redeem(...)`; a NEW user
  is created with `Role: role`; the existing-user branch is untouched (only
  `PasswordHash`/`Name` update via `UpdatePassword`) — existing users,
  including the superadmin, keep their role structurally, not by a special
  case.
- **Rationale**: The current code path already resets-password-only for
  existing phones; simply never writing `u.Role` there *is* the invariant
  (FR-007). Promotion of a resident happens exclusively via a building grant
  (R7), which is an explicit, audited action.
- **Alternatives considered**: promoting existing residents on manager-invite
  redemption (rejected: silent widening, forbidden by spec FR-007/FR-019);
  refusing manager invites for existing phones (rejected: invites double as
  the password-recovery channel today — refusing would break recovery).

## R6. Setup creates the superadmin

- **Decision**: `setup` creates `Role: RoleSuperAdmin` instead of
  `RoleManager`; the zero-users lock and all messages otherwise unchanged
  (message wording shifts to "سرپرست" only if it names the role — the current
  text "حساب مدیریتی" stays acceptable).
- **Rationale**: Repurposing the existing once-only bootstrap is the spec's
  chosen way to have a "default" superadmin without seeded credentials
  (Assumption #1).
- **Alternatives considered**: migration seeding a fixed phone + default
  password (rejected: shipped-credential anti-pattern); leaving setup as
  manager and hand-granting superadmin via SQL (rejected: defeats SC-001's
  zero-DB-steps criterion).

## R7. Manager-grant endpoints live in the building module, gated per building

- **Decision**: Under `building.Register`'s existing `idB :=
  buildings.Group("/:id")`, add `GET/POST /managers` and `DELETE
  /managers/:userId`. Each handler first resolves the caller via the existing
  service gate (the `CanManagerAccess` check that already yields
  `httpx.Forbidden("دسترسی غیرمجاز است.")` in `service.go`), then performs the
  grant operation. The route group already carries
  `authMW + RequireRole(RoleManager)` from `main.go` — residents and (by
  design, spec FR-002) the superadmin can't reach these routes.
- **Rationale**: `user_buildings` is the building module's table (raw GORM
  access pattern per `repository.go`); per-object authorization is the
  established two-layer idiom (role middleware + `CanManagerAccess`), so all
  three sub-routes get 403-for-non-granted-managers for free. Superadmin
  exclusion is automatic: `RequireRole(RoleManager)` does not include it, so
  a superadmin cannot appear in or mutate any building's manager set.
- **Alternatives considered**: an `admin`-flavored route in the auth module
  (rejected: crosses module ownership of `user_buildings`); letting the
  superadmin manage grants (rejected: spec Non-Requirement — its role is
  bootstrap only).

## R8. Grant semantics: role flip, idempotency, superadmin refusal

- **Decision**: `POST /buildings/:id/managers {phone}`:
  1. look up active user by phone → 404-style refusal if absent/inactive;
  2. if `user.Role == RoleSuperAdmin` → `httpx.BadRequest` (Persian);
  3. conditional insert into `user_buildings` (the repo's existing
     idempotent `GrantManager`): duplicate → 409 CONFLICT
     `"این مدیر از قبل دسترسی دارد."`;
  4. if `user.Role == RoleResident` → `UPDATE users SET role='manager' WHERE
     id=? AND role='resident'` (raise-only, matches no row if raced);
  5. respond `201 {user_id, phone, name, role, granted_at}`; audit
     `building.manager_granted` with after = user_id (+ role before/after).
- **Rationale**: Response carries the *resulting* role so the client can show
  "this resident is now a manager" (task: "say so in the response shape via
  the new role"). The `WHERE role='resident'` guard makes the flip a
  monotone raise that can never demote or touch superadmin even under a race.
- **Alternatives considered**: auto-granting the new manager into the
  inviter's buildings at invite time (rejected: invites are phone-bound
  futures; recipient may register much later and grant intent is per
  building); full user replacement update (rejected: clobber risk).

## R9. Revoke: atomic last-manager protection

- **Decision**: `DELETE /buildings/:id/managers/:userId`:
  - `userId == caller.ID` → `httpx.BadRequest("برای حذف دسترسی خود ابتدا مدیر
    دیگری معرفی کنید.")` (handover first — FR-016);
  - otherwise a single guarded statement:
    `DELETE FROM user_buildings WHERE user_id=? AND building_id=? AND
    (SELECT count(*) FROM user_buildings WHERE building_id=?) > 1`;
    0 rows affected → re-check grant existence → 409 CONFLICT last-manager
    message if it existed, else 404; success → `204` + audit
    `building.manager_revoked` (before = user_id, after = null).
- **Rationale**: A count-then-delete pair races two concurrent revokes to
  zero managers; folding the count into the DELETE's predicate makes the
  invariant single-statement-atomic in PostgreSQL without explicit row
  locks, matching the module's raw-SQL-through-GORM style. The 0-rows path
  distinguishes "was last" (grant still exists) from "never granted" with one
  cheap follow-up read.
- **Alternatives considered**: `SELECT ... FOR UPDATE` transaction (rejected:
  heavier, and the predicate version needs no isolation gymnastics); advisory
  lock per building (rejected: same guarantee, more machinery).

## R10. Audit wiring for the three actions

- **Decision**: Reuse the existing pieces verbatim: `building.Register`
  already receives `*audit.Service` and wraps mutations with
  `aud.Middleware(action, objectType)` + a `setAuditCtx`-style publisher of
  object/before/after (pattern: `unit.create`, `setUnitAuditCtx`). New
  middleware entries: `building.manager_granted` / `building.manager_revoked`
  (object_type `"building"`, object_id = building id). For `invite.issued`,
  `auth.Handler` appends best-effort via its existing `LoginAuditor`
  interface (`Append(ctx, actor, "invite.issued", "invite", nil, nil,
  {phone, role})`) exactly like `user.login`, ignoring the error.
- **Rationale**: Zero new audit infrastructure; both call sites already prove
  the pattern (auth's cycle-avoiding interface, building's decorator
  middleware). Best-effort semantics match spec FR-020 and the login
  precedent.
- **Alternatives considered**: strict audit (operation fails if audit fails) —
  rejected by spec; putting invite auditing in InviteService — rejected
  (service has no audit dependency today; handler keeps infra out of domain).

## R11. `/auth/me` and session payload for superadmin

- **Decision**: `me`/`respondSession` return `role: "superadmin"` with an
  empty `buildings: []` scope (no manager lookup attempted). No changes to
  login/token flows.
- **Rationale**: Scope resolution is role-switched already (manager →
  user_buildings, resident → occupancies); superadmin simply has neither, so
  an explicit empty arm keeps one honest shape rather than leaking a SQL
  error path.
- **Alternatives considered**: treating superadmin as manager in scope
  resolution (rejected: implies building access it must not have).

## R12. Flutter: role toggle, managers screen, non-manager hiding

- **Decision**:
  - `ManagerInviteScreen`: `SegmentedButton<String>` (ساکن / مدیر), default
    ساکن; payload gains `role`; result banner text names the issued role.
    Invite-screen reachability widens to superadmin (home/quick-action gate
    checks `role == manager || role == superadmin` where it currently checks
    manager; the `/manager` shell itself still renders an empty building list
    for superadmin).
  - New `BuildingManagersScreen` at
    `context.push('/manager/buildings/$buildingId/managers')` — row action
    beside units/people on `building_list_screen.dart` (that list only ever
    contains buildings the caller manages, satisfying FR-023's hiding rule;
    residents never see the manager section at all).
  - Repository methods on `BuildingsRepository` via `apiClientProvider`
    (Dio); errors rendered through existing `describeError` + Persian
    envelope mapping; grant date formatted Jalali via `core/datetime/jalali`.
- **Rationale**: Every listed building is caller-managed by server scope
  (FR-037), so a row action *is* the manager-only gate client-side; server
  re-enforces anyway. SegmentedButton is the Material 3 binary-choice idiom
  and exists in the pinned Flutter version.
- **Alternatives considered**: dropdown for role (rejected: two options,
  segmented is faster); overflow menu vs persistent icon (project uses
  leading icon buttons per row — follow the row convention); gating by
  client-side role beyond section routing (rejected: server is the arbiter;
  redundant logic drifts).

## R13. Testing strategy

- **Decision**:
  - Unit (`auth/invite_test.go` style): `Issue` persists the role;
    `Issue/Redeem` round-trips resident & manager; invalid role rejected at
    handler level via the existing handler-test env.
  - Integration (`integration_harness_test.go` style, real Postgres via the
    docker→dev-PG→skip ladder): full bootstrap chain (setup → superadmin →
    manager invite → register → `/auth/me` shows granted-empty then
    granted-building); POST/GET/DELETE managers round-trip; duplicate 409;
    last-manager 409; self-remove 400; resident-flip; superadmin-phone grant
    refused; audit rows asserted for grant/revoke/invite.issued — plus a
    manager-without-grant 403 on all three sub-routes of building X.
  - Flutter: `flutter test --no-pub` widget tests for the role toggle payload
    and managers screen list/add/remove against a mocked Dio adapter.
- **Rationale**: Mirrors both existing suites' harnesses exactly (fakeClock,
  fakeAuditor, `e.request`), so new tests cost no new scaffolding.
- **Alternatives considered**: HTTP-contract tests only (rejected: miss the
  atomicity assertions on last-manager delete); testcontainers-only (the
  harness already degrades to dev-PG/skip on machines without Docker).

## R14. Documentation touch-ups

- **Decision**: The implementation updates `specs/001-.../contracts/api.md`
  (auth + buildings tables) and `data-model.md` (invite role, multi-manager),
  the FR-036/FR-037 single-manager wording in `spec.md`, and 001 `plan.md`
  module notes — this feature's own `contracts/api.md` stays a delta doc to
  avoid maintaining two copies.
- **Rationale**: 001's contract files are the living API reference (the task
  text treats them as such); 002 documents only what changed.
- **Alternatives considered**: full 002 contract copy (rejected: drift);
  leaving 001 stale (rejected: it's the source of truth for client work).
