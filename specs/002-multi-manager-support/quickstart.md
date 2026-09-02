# Quickstart & Validation Guide: Multi-Manager Support with Superadmin Bootstrap

**Feature**: specs/002-multi-manager-support
Detail lives in [contracts/api.md](contracts/api.md) and
[data-model.md](data-model.md); this file is the runnable proof.

## Prerequisites

- Go 1.23+, Flutter (with pub deps already vendored — **always pass
  `--no-pub`**, pub get is broken on this machine), Docker **or** a local
  PostgreSQL 16 reachable with the dev credentials below.
- From repo root:

```bash
docker compose up -d                 # PostgreSQL 16 on :5432 (hamsa/hamsa)
cp backend/config.example.yaml backend/config.yaml   # first time only
```

Migrations apply automatically at server start (and inside the integration
harness). Manual: `cd backend && go run ./cmd/migrate up`.

## 1. Automated gates

```bash
cd backend && go build ./... && go vet ./... && go test ./internal/...
cd mobile  && flutter analyze --no-pub && flutter test --no-pub
```

Expected: build/vet clean; Go suite green — integration tests run against
real Postgres (testcontainers → dev-PG fallback → skip without Docker), so
with Docker or the compose DB up, no integration test may be SKIPPED;
flutter analyze reports no issues and widget tests pass.

Key new tests to point at:

| Test | Proves |
|---|---|
| `auth` unit: invite role round-trip | FR-004/006/008 (Issue/Redeem carry role; legacy invite defaults resident) |
| `auth` integration: bootstrap chain | SC-001 (setup→superadmin→manager invite→register→`/auth/me`) |
| `auth` integration: no-promotion | SC-002 / FR-007 (existing user, incl. superadmin phone, keeps role on redeem) |
| `building` integration: managers CRUD | FR-011…FR-016 (round-trip, duplicate 409, last-manager 409, self-remove 400, resident→manager flip) |
| `building` integration: authorization | SC-005 / FR-017 (manager without grant on building X → 403 ×3) |
| audit assertions in the above | SC-004 / FR-020 (`invite.issued`, `building.manager_granted`, `building.manager_revoked` rows) |

## 2. End-to-end manual walkthrough (SC-001…SC-006)

Fresh DB (`docker compose down -v && docker compose up -d`), server
(`cd backend && go run ./cmd/server`), app (`cd mobile && flutter run -d
chrome --no-pub`).

1. **Superadmin bootstrap** — App opens setup screen; create account A.
   `GET /auth/me` → `role: "superadmin"`, `buildings: []`. Second setup
   attempt → 409 "حساب مدیریتی قبلاً ساخته شده است." ✔ SC-001 step 1.
2. **Create a manager** — As A, open invite screen, toggle مدیر, issue for
   phone P → 201 shows code **and** role مدیر. Register P with the code →
   account is manager; `A` may now stop being needed. ✔ SC-001 step 2.
3. **No silent promotion** — Issue a manager invite for phone Q already
   registered as resident; redeem via register screen → Q's password resets,
   `/auth/me` still shows resident. ✔ SC-002.
4. **Build + grant** — Manager M1 creates building B (auto-manager). Open B's
   managers screen (row action): M1 listed with grant date (Jalali display).
   Add manager M2 by phone → row appears. Add M2 again → Persian conflict
   "این مدیر از قبل دسترسی دارد." ✔ SC-001 steps 3–4, SC-006 timing.
5. **Protections** — M1 tries removing self → rejected; M2 removed, then with
   B down to one manager, removal of that last manager → 409 conflict. ✔ SC-003.
6. **Resident flip** — Add resident R's phone as manager of B → R's `/auth/me`
   now role manager with B in scope; app shows R in managers list. ✔ FR-013.
7. **Authorization** — Manager M3 (manages other building C only) calls
   GET/POST/DELETE B's managers → 403 all three. Superadmin A → 403 too
   (not a building participant). ✔ SC-005.
8. **Audit** — As DB observer:

```sql
SELECT action, object_type, before_value, after_value
FROM audit_logs
WHERE action IN ('invite.issued','building.manager_granted','building.manager_revoked')
ORDER BY id;
```

   One row per successful issue-with-role / grant / revoke above, actor set,
   role/user ids in before/after. ✔ SC-004.

## 3. Regression probes (must stay green)

- Resident cannot POST `/auth/invites` (403).
- Login/refresh/logout behavior unchanged for all three roles.
- Charge/invoice/payment suites untouched and passing (`go test ./internal/...`).
- Building list for a fresh, ungranted manager renders empty (valid state).

## Notes

- If Postgres is unavailable, integration tests SKIP — do not treat a green
  run with skips on the new files as SC-proof; re-run with Docker up.
- Down-migration check (optional, dev DB only): `go run ./cmd/migrate down 1`
  removes `invite_codes.role`; the `superadmin` enum value intentionally
  remains (data-model.md).
