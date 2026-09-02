# Phase 1 Data Model: Multi-Manager Support with Superadmin Bootstrap

**Feature**: specs/002-multi-manager-support | **Date**: 2026-09-02
Migration: `backend/migrations/0010_invite_roles.{up,down}.sql`

## Changes to existing entities

### user_role (PostgreSQL enum) — WIDENED

```
manager | resident | superadmin   -- superadmin added by 0010
```

- One added value; no existing value redefined. Not used inside migration
  0010's own transaction (research R2). Down migration cannot drop enum
  values; it leaves `'superadmin'` in place (documented in the .down.sql).

### users (unchanged schema; new legal value)

| column | type | change |
|---|---|---|
| role | user_role NOT NULL DEFAULT 'resident' | may now be `'superadmin'` |

**Role state machine (invariant: raise-only, never lower via app paths)**

```
(new) --/auth/setup-------------> superadmin        (terminal; exactly one; permanent)
(new) --register w/ resident invite--> resident
(new) --register w/ manager invite--> manager
resident --POST /buildings/:id/managers (grant)---> manager   (audited, explicit)
manager  -- any app path ----------> resident       (IMPOSSIBLE: no demotion path exists)
```

Validation rules (enforced where noted):
- Exactly one superadmin may exist → enforced by: setup runs only while
  `CountActive == 0`; invite role whitelist excludes `superadmin` (handler);
  grant targets reject superadmin phones (building service). *(FR-001/002/003)*
- Redemption never writes `users.role` for an existing phone. *(FR-007)*

### invite_codes — WIDENED (one column)

| column | type | notes |
|---|---|---|
| id | UUID PK | unchanged |
| phone | VARCHAR(11) NOT NULL | unchanged — phone binding |
| code_hash | VARCHAR(64) NOT NULL | unchanged — SHA-256, never plaintext |
| **role** | **user_role NOT NULL DEFAULT 'resident'** | **NEW** — role a NEW registrant receives; whitelist at issue-time = {resident, manager} |
| created_by | UUID → users.id | unchanged |
| expires_at | TIMESTAMPTZ NOT NULL | unchanged — 7-day validity |
| consumed_at | TIMESTAMPTZ | unchanged — single-use (conditional consume) |
| created_at | TIMESTAMPTZ | unchanged |

Lifecycle unchanged: `issued → (redeemed | expired | superseded-by-newer-latest)`.
`role` is immutable after issue. Existing rows default to `'resident'` — a
pre-feature outstanding invite registers a resident, exactly as today.

## Unchanged entities with new invariants

### user_buildings (manager scope) — same composite PK, now app-managed

`PK (user_id, building_id)`, `granted_at TIMESTAMPTZ DEFAULT now()`.
Already many-to-many; previously written only by `CreateBuilding`'s auto-grant.
New writers/reader (all via `building.Repository` raw-table access):

| operation | rule | on violation |
|---|---|---|
| grant(phone) | caller must pass `CanManagerAccess(building)`; target must be an **active** user; target role ≠ superadmin | 403 / 404 / 400 |
| grant(phone) | not already granted | 409 `"این مدیر از قبل دسترسی دارد."` |
| grant side effect | target role resident → manager (raise-only UPDATE guarded by `role='resident'`) | — |
| revoke(userId) | caller granted to building; `userId ≠ caller` | 403 / 400 |
| revoke(userId) | building must retain ≥ 1 manager — atomic guarded DELETE (research R9) | 409 |

### audit_logs (append-only trigger preserved) — NEW action vocabulary

| action | object_type | object_id | before → after | actor |
|---|---|---|---|---|
| `invite.issued` | `invite` | NULL | NULL → `{phone, role}` | manager/superadmin (best-effort via LoginAuditor-style append, like `user.login`) |
| `building.manager_granted` | `building` | building id | NULL → `{user_id, role}` (role = resulting) | granted manager (aud.Middleware) |
| `building.manager_revoked` | `building` | building id | `{user_id}` → NULL | granted manager (aud.Middleware) |

Best-effort: audit write failure never fails the operation (FR-020).

## Derived read models

**BuildingManagerView** (GET `/buildings/{id}/managers` row) — join of
`user_buildings ⋈ users`, live rows only:

| field | source |
|---|---|
| user_id | user_buildings.user_id |
| phone | users.phone |
| name | users.name |
| role | users.role (always `manager` today; carried for client display) |
| granted_at | user_buildings.granted_at |

Ordering: `granted_at ASC` (creation-grant first). Superadmin never appears
(cannot be granted).

## Go model touch-points (for tasks phase)

- `auth.User`: + `RoleSuperAdmin = "superadmin"`.
- `auth.InviteCode`: + `Role string` (`gorm:"column:role"`).
- `building` package: `BuildingManager` struct for the view (raw Scan per
  repository.go style; no new GORM association model).
- JWT claims, refresh tokens, occupancies, notifications: **untouched**.
