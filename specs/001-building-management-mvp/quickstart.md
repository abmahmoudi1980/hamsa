# Quickstart: Building Management MVP (P0) — Validation Guide

**Feature**: specs/001-building-management-mvp

This guide proves the feature works end-to-end. Implementation details live in `tasks.md`; API details in [contracts/api.md](contracts/api.md); entity details in [data-model.md](data-model.md).

## Prerequisites

- Go 1.23+, Docker (for PostgreSQL), Flutter SDK (stable, Android toolchain)
- An Android device/emulator (API 26+)
- (Optional, for real gateway) Zarinpal sandbox merchant ID — all scenarios below work with the **mock gateway** enabled by `APP_ENV=dev`

## Setup

```bash
# 1. Database
docker run -d --name hamsa-pg -e POSTGRES_PASSWORD=hamsa -e POSTGRES_DB=hamsa -p 5432:5432 postgres:16

# 2. Backend (from backend/)
cp config.example.yaml config.yaml   # APP_ENV=dev → mock SMS (OTP printed to server log), mock gateway
go run ./cmd/server                    # runs migrations on start, serves on :8080

# 3. Mobile app (from mobile/)
flutter pub get
flutter run --dart-define=API_BASE_URL=http://<host>:8080/api/v1
```

Dev-mode conveniences: OTP codes are logged by the server (mock SMS) and also returned in the `/auth/otp/request` response; online payments are auto-verified by the mock gateway without leaving the app.

**UI language check (applies to every scenario below)**: the entire app is Persian (Farsi), RTL, with Persian digits everywhere, and **every date input is a Jalali date picker** (billing period dates, payment dates, expense dates, request dates) — no English text and no Gregorian picker anywhere in the UI. Deviating from either is a validation failure, not a cosmetic issue.

## Scenario 1 — Manager sets up a building and people (P0-01, P0-02)

1. Log in with any phone number → OTP from the server log → first user is granted the manager role.
2. Create building "برج هامسا" (10 units). Add unit 1 (area 80 m², floor 1) … unit 10; try to create a second unit 1 → **expect rejection "شماره واحد تکراری"**.
3. Add person "رضا محمدی" as tenant of unit 1 with occupant count 4 (and others to reach 20 total occupants across the building).

**Pass**: units list is searchable/filterable; occupancy history and occupant-count history show the entries; unit 1 edit produces a history record.

## Scenario 2 — Charge calculation to issuance (P0-03) — *the core acceptance*

1. Create billing period with two cost items:
   - "هزینه عمومی" — 10,000,000 Toman, method `equal`
   - "آب" — 8,000,000 Toman, method `per_occupant`
2. `Calculate` → open **preview** (BR-08): verify unit 1 shows 1,000,000 + 1,600,000 = **2,600,000** (spec §25), and that Σ rounded shares per item equals the item total.
3. Change unit 1's occupant count to 5 while the period is `calculated`, recalculate → preview updates to 1,600,000 → 2,000,000 water share. Reopen → set occupant count back to 4 → recalculate → 2,600,000.
4. `Issue` invoices → residents receive in-app notifications; invoice numbers are sequential.

**Pass**: preview matches 2,600,000 before issue; issuance succeeds; **then** edit unit 1's occupant count again → the issued invoice amount is unchanged (BR-05).

## Scenario 3 — Immutability & correction (BR-03, BR-10)

1. With an issued invoice: change occupant count / area of unit 1 → **invoice amount must not change**.
2. Add a correction: credit adjustment of 100,000 with reason → invoice shows a separate adjustment row; original rows untouched.
3. Cancel another unit's invoice with a reason → status `cancelled`, row and history retained; a new corrected period can be issued.

## Scenario 4 — Payments and balances (P0-04)

1. Manager records a manual payment of 1,500,000 against unit 1's 2,600,000 invoice → invoice status `partial`, unit balance shows 1,100,000 outstanding.
2. Resident (log in with resident's phone) pays the remainder online (mock gateway) → payment `verified`, invoice `paid`, receipt viewable in payment history.
3. Overpay a fully-paid invoice → surplus appears as unit credit applied to the next period's invoice.

**Pass**: unit balance components (prior debt, current charge, late fee, payments, credit, balance) always reconcile with the formula in spec §9.

## Scenario 5 — Expenses & financial report (P0-05)

1. Register expenses: "برق لابی" (electricity, 2,000,000, with receipt image), "نگهبانی" (security, 8,000,000).
2. Open financial report for the month → income/expense/net/debt totals match the recorded data.

## Scenario 6 — Maintenance & announcements (P0-06, P0-07)

1. Resident submits an urgent "آسانسور لرزش دارد" request with photo → appears `new` to the manager.
2. Manager assigns a technician → `in_progress` → records cost 500,000 → `done` → closes. Resident sees every status change in the app without contacting anyone.
3. Manager publishes an announcement targeting block A only → only residents of block A see it; unread badge counts; marking read clears it.

## Scenario 7 — Access control & audit (P0-10)

1. Resident A requests resident B's invoice id (`GET /invoices/{B's id}`) → **403**.
2. Resident cannot open any manager route → 403.
3. Manager audit view: every sensitive action from scenarios 1–6 is logged with user, action, timestamp, object, and before/after values (unit edits, formula/cost-item changes, invoice issue/cancel, payments, expenses, request status changes).

## Scenario 8 — Dashboards (P0-08, P0-09)

1. Resident home: payable amount, latest invoice + due date + status, open request count, latest announcements with unread count — all from `/me/home`.
2. Manager dashboard: with 7 debtor units and 3 open requests in the data, cards and alerts show exactly those numbers (SC-007).

## Automated Verification

- Backend: `go test ./...` from `backend/` — includes the spec §25 table-driven charge-engine test (unit 1 = 2,600,000), largest-remainder reconciliation tests, immutability tests, and repository integration tests against ephemeral PostgreSQL.
- Mobile: `flutter test` from `mobile/` — includes widget tests asserting the `fa` locale/RTL layout and that date fields use the Jalali picker; `flutter test integration_test/` runs scenarios 2, 4, and 6 end-to-end against a locally running dev backend.

## Overall MVP Acceptance

The feature is accepted when all 14 items of spec §26 pass manually by a non-technical manager following only in-app guidance (no technical-team intervention), and the automated suites above are green.
