# Implementation Plan: Multi-Manager Support with Superadmin Bootstrap

**Branch**: `002-multi-manager-support` | **Date**: 2026-09-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-multi-manager-support/spec.md`

## Summary

Three-tier privilege model for Hamsa: the one-time first-run setup now creates a
single **superadmin** whose only power is issuing manager-role invites; any
**building manager** can issue manager or resident invites, create buildings,
and — through new per-building endpoints — list, add, and remove the building's
managers, with last-manager and self-removal protections. Invites gain a role
column (default `resident`); redemption assigns the invite's role to NEW users
only and never changes an existing user's role. All three sensitive actions
(`invite.issued`, `building.manager_granted`, `building.manager_revoked`) are
audited best-effort through the existing `audit.Service`. The Flutter app gains
a ساکن/مدیر role toggle on the invite screen and a new per-building managers
screen.

**Technical approach** (from [research.md](research.md)): widen the existing
`user_role` PG enum with `superadmin` and add `role user_role NOT NULL DEFAULT
'resident'` to `invite_codes` in one migration (0010); extend
`InviteService.Issue/Redeem` to carry the role; enforce who may issue
manager invites purely at the handler/route layer (`RequireRole(RoleManager,
RoleSuperAdmin)` on `/auth/invites`, value validation 400); add manager-scoped
routes under the existing `/buildings/:id` group in `building.Register`, each
gated by the existing per-building check, with an atomic single-statement
guarded DELETE for last-manager protection. No JWT changes (tokens carry
identity only; role is resolved from the DB per request).

## Technical Context

**Language/Version**: Go 1.23+ (backend), Dart 3.x / Flutter 3.24+ (client)

**Primary Dependencies**: `gin-gonic/gin`, `gorm.io/gorm` + postgres driver,
`golang-migrate/migrate`, `golang-jwt/jwt/v5`, `bcrypt`; Flutter:
`flutter_riverpod`, `go_router`, `dio`, `shamsi_date`, local single-locale
ARB (`mobile/lib/core/l10n/app_fa.arb`)

**Storage**: PostgreSQL 16 — one migration (`0010_invite_roles`): enum widening
+ `invite_codes.role`; `user_buildings` composite PK unchanged (already
many-to-many)

**Testing**: Go `testing` + `testify`; unit (`auth/invite_test.go`),
integration against real PostgreSQL via the `integration_harness_test.go`
harness (docker/testcontainers with dev-PG fallback, skip if unreachable);
`flutter_test` widget tests; run with `--no-pub` (pub get broken on this
machine)

**Target Platform**: Linux/Windows server (single Go binary) + Flutter client
(web/android)

**Project Type**: Web-service API + mobile-app client (modular monolith,
feature packages under `backend/internal/`)

**Performance Goals**: manager list/add/remove are single-building, small-cardinality
operations; p95 < 300 ms; invite issue/redeem unchanged (< 100 ms + bcrypt)

**Constraints**: no silent role promotion (invite redemption never mutates an
existing user's role); building never drops to zero managers via this feature;
superadmin is exactly one, permanent, and never a building-management
participant; Persian error envelope on every new failure; audit writes
best-effort; clean cutover — `/auth/invites` response shape may change outright

**Scale/Scope**: 1 migration, ~6 backend files + 3 test files, 1 new + 3
modified Flutter files, 4 spec-doc updates

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

`.specify/memory/constitution.md` exists but is an **unfilled template**
(placeholders only — no ratified principles or gates). Therefore:

- **Pre-research gate**: PASS (no active constraints to violate)
- **Post-design re-check (after Phase 1)**: PASS — carried the spec's own
  governance invariants into the design instead: least-privilege grants
  (explicit raise-only), append-only audit on every privilege change,
  per-building object authorization, no zero-manager buildings.

## Project Structure

### Documentation (this feature)

```text
specs/002-multi-manager-support/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/
│   └── api.md           # Delta REST contract over 001/contracts/api.md
├── checklists/
│   └── requirements.md  # from /speckit.specify
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── migrations/
│   ├── 0010_invite_roles.up.sql        # user_role += 'superadmin'; invite_codes.role
│   └── 0010_invite_roles.down.sql      # drop column (enum value stays — see research R2)
├── internal/
│   ├── auth/
│   │   ├── invite.go                   # InviteCode.Role; Issue(+role); Redeem→(role,error)
│   │   ├── user.go                     # RoleSuperAdmin constant
│   │   ├── handlers.go                 # setup→superadmin; register uses invite role;
│   │   │                               #   createInvite role validation + audit;
│   │   │                               #   RequireRole(manager, superadmin) on /auth/invites
│   │   ├── invite_test.go              # unit: role round-trip Issue/Redeem (extend)
│   │   └── auth_integration_test.go    # integration: superadmin setup, manager invite
│   │                                   #   bootstrap, no-promotion-on-redeem (extend)
│   ├── building/
│   │   ├── handlers.go                 # GET/POST/DELETE /buildings/:id/managers
│   │   │                               #   + IsManagerOf gate on each
│   │   ├── service.go                  # ListManagers / GrantManager / RevokeManager
│   │   │                               #   (resident→manager flip, last-manager +
│   │   │                               #   self-removal rules, superadmin refusal)
│   │   ├── repository.go               # raw user_buildings access: list w/ user join,
│   │   │                               #   idempotent grant, guarded single-statement delete
│   │   └── manager_integration_test.go # new: grant/revoke round-trip, 409s, 403 authz,
│   │                                   #   audit rows (harness style, real Postgres)
│   └── cmd wiring: none — building.Register already receives authMW group;
│       auth route middleware gains RoleSuperAdmin (see R4)

mobile/lib/
├── features/auth/
│   ├── screens/manager_invite_screen.dart  # SegmentedButton ساکن/مدیر, role in banner
│   └── auth_repository.dart                # invites(role:) payload
├── features/buildings/
│   ├── screens/building_managers_screen.dart   # NEW: list + add + remove(confirm)
│   ├── buildings_repository.dart               # 3 manager endpoints via apiClientProvider
│   ├── buildings_controller.dart               # managers list/add/remove providers
│   ├── models/building.dart (or new managers.dart) # BuildingManager model
│   └── screens/building_list_screen.dart       # managers action on each row
├── core/router/app_router.dart             # route /manager/buildings/:buildingId/managers
└── core/l10n/app_fa.arb                    # all new Persian strings

specs/001-building-management-mvp/          # doc touch-up during implementation:
├── contracts/api.md                        # auth + buildings tables updated
├── data-model.md                           # invite_codes.role, multi-manager note
├── spec.md                                 # FR-036/FR-037 single-manager wording
└── plan.md                                 # module notes
```

**Structure Decision**: No new packages. The change lands entirely in the two
existing domain packages that already own the affected data (`auth` for
invites/roles, `building` for `user_buildings` grants), following their
established handler→service→repository layering and raw-table GORM access for
`user_buildings`. Mobile follows the feature-first layout: the managers screen
joins the `buildings` feature next to units/people; the invite role toggle stays
in `auth`.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations — table intentionally empty.
