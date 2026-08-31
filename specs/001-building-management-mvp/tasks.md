# Tasks: Building Management MVP (P0)

**Input**: Design documents from `/specs/001-building-management-mvp/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/api.md, quickstart.md

**Tests**: Included — the implementation plan (plan.md → Testing) and quickstart.md explicitly define automated verification for the charge engine (spec §25 contract test), repository/integration tests, and Flutter widget/integration tests. Charge-engine and critical-path tests are written before implementation.

**Organization**: Tasks are grouped by the 10 user stories from spec.md to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1…US10)
- Include exact file paths in descriptions

## Path Conventions

- **Backend (Go)**: `backend/` — `cmd/server/`, `internal/<domain>/`, `migrations/`
- **Mobile (Flutter)**: `mobile/` — `lib/core/`, `lib/features/<feature>/`, `integration_test/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create project structure per plan.md: `backend/` (Go) and `mobile/` (Flutter) directories, root `.gitignore`, and `docker-compose.yml` with PostgreSQL 16 (db name `hamsa`, password `hamsa`, port 5432)
- [x] T002 Initialize Go module in `backend/go.mod` (Go 1.23+) with dependencies: gin, gorm + postgres driver, golang-migrate, golang-jwt/v5, go-persian-calendar, uuid; create empty `backend/cmd/server/main.go`
- [x] T003 [P] Create `backend/config.example.yaml` (APP_ENV=dev, DB DSN, JWT secret, SMS/gateway provider keys, file storage path) and config loader in `backend/internal/platform/config/config.go`
- [x] T004 [P] Initialize Flutter project in `mobile/` (Android-only platform) with dependencies in `mobile/pubspec.yaml`: flutter_riverpod, go_router, dio, freezed + json_serializable + build_runner, shamsi_date, persian_datetime_picker, flutter_localizations (fa only), flutter_secure_storage, image_picker, Vazirmatn font asset
- [x] T005 [P] Configure lint/format: `backend/.golangci.yml` and `mobile/analysis_options.yaml`; add `make lint` / `make test` targets in root `Makefile`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 Implement GORM PostgreSQL connection + golang-migrate runner (runs pending migrations on start) in `backend/internal/platform/db/db.go`; wire into `backend/cmd/server/main.go`
- [x] T007 Implement HTTP foundation in `backend/internal/platform/httpx/`: Gin router factory, JSON error envelope (`code`/`message` in Persian/`details`), recovery middleware, request-id + slog structured logging middleware
- [x] T008 [P] Create migration `backend/migrations/0001_core.up.sql` (+ `.down.sql`): `users`, `otp_codes`, `audit_logs` (append-only, no UPDATE/DELETE grants), `notifications`, `files` tables per data-model.md
- [x] T009 [P] Implement `SmsSender` interface + console/log dev sender + Kavenegar HTTP adapter in `backend/internal/platform/sms/sms.go` (selection by config; 6-digit code contract)
- [x] T010 [P] Implement local-disk file storage service in `backend/internal/platform/storage/storage.go` (image/PDF, ≤ 5 MB, UUID paths) + `POST /files` multipart handler
- [x] T011 Implement auth core in `backend/internal/auth/`: user model/repository, OTP issue/verify service (hashed codes, 2-min expiry, max 3 attempts, resend throttle 60 s), JWT access (15 min) + rotating refresh tokens (30 days, hashed in DB, family revocation on reuse) per research.md R4/R8
- [x] T012 Implement auth middleware in `backend/internal/auth/middleware.go`: Bearer parsing, per-request scope resolution (manager → permitted buildings via `user_buildings`; resident → occupied units), object-level authorization helper returning 403 per contracts/api.md
- [x] T013 [P] Implement audit-log service in `backend/internal/audit/`: append(actor, action, object, before/after JSONB) + Gin handler decorator for audited routes (FR-038)
- [x] T014 [P] Implement in-app notification service in `backend/internal/notification/`: create/list/mark-read for `notifications`, `PushNotifier` interface (no-op default) per research.md R10
- [x] T015 Implement Go unit tests for OTP and JWT/refresh rotation in `backend/internal/auth/auth_test.go` (mock SmsSender; clock-injected)
- [x] T016 [P] Create Flutter app shell in `mobile/lib/core/`: `main.dart` with MaterialApp configured for single locale `fa` + RTL + Vazirmatn theme (Persian digits app-wide), `core/l10n/fa.arb` string file with all shared Persian strings, `core/theme/app_theme.dart`
- [x] T017 [P] Implement Dio client in `mobile/lib/core/network/`: base client + auth interceptor (Bearer), 401 → refresh-token interceptor with rotation, error envelope → Persian exception mapping per contracts/api.md conventions
- [x] T018 [P] Implement Jalali helpers in `mobile/lib/core/datetime/jalali.dart`: Jalali↔ISO-8601 conversion, Persian-digit formatting, and a single `JalaliDatePickerField` wrapper around `persian_datetime_picker` — the ONLY date input component in the app
- [x] T019 Implement go_router skeleton in `mobile/lib/core/router/router.dart`: role-based root routing (manager shell vs resident shell vs login), auth-state redirect; implement `mobile/lib/features/auth/` auth state controller (Riverpod) + secure token storage (flutter_secure_storage)
- [x] T020 [P] Create shared widgets in `mobile/lib/shared/`: Persian money display (Toman + thousands separators + Persian digits), status chips (invoice/request/period enums → Persian labels), empty-state, attachment picker, form validation helpers (Iranian mobile format)

**Checkpoint**: Foundation ready — `go test ./...` green, Flutter app boots to login in Persian RTL, migrations run against docker-compose PostgreSQL. User story implementation can now begin in parallel.

---

## Phase 3: User Story 1 — Authenticate and Control Access (Priority: P1) 🎯 MVP

**Goal**: Manager and resident log in with mobile + OTP; residents see only their unit's data; managers only their buildings; sensitive actions audited.

**Independent Test**: A manager and a resident account log in; the resident cannot access another resident's invoice (403) or any manager route (403); audit entries exist for sensitive actions (quickstart Scenario 7).

### Tests for User Story 1 (write first, must FAIL)

- [x] T021 [P] [US1] Go integration test for OTP flow + token refresh + rate limiting in `backend/internal/auth/auth_integration_test.go` (testcontainers PostgreSQL)
- [x] T022 [P] [US1] Go integration test for object-level authorization (resident↔resident 403, resident→manager route 403) in `backend/internal/auth/authz_test.go`

### Implementation for User Story 1

- [x] T023 [US1] Implement auth handlers in `backend/internal/auth/handlers.go`: `POST /auth/otp/request` (rate-limited, dev mode returns code), `POST /auth/otp/verify` (auto-register unknown phone; returns tokens + user context), `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me` per contracts/api.md
- [x] T024 [US1] Implement first-user bootstrap: the first registered user in a fresh deployment is granted manager role (documented in handler); add `user_buildings` grant on building creation later (US2)
- [x] T025 [US1] Wire audit middleware (T013) into auth-sensitive routes and add `user.login` audit action in `backend/internal/auth/handlers.go`
- [x] T026 [US1] Implement Flutter login screens in `mobile/lib/features/auth/`: phone-entry page, OTP-entry page (countdown, resend after 60 s), Persian error states — all strings from `core/l10n/fa.arb`
- [x] T027 [US1] Implement post-login role routing in `mobile/lib/core/router/router.dart`: manager → building list (placeholder), resident → resident shell (placeholder), persisting session across restarts

**Checkpoint**: User Story 1 fully functional — login works end-to-end for both roles with enforced isolation.

---

## Phase 4: User Story 2 — Set Up Building, Floors, and Units (Priority: P1)

**Goal**: Manager registers buildings and units with full attributes; duplicate unit numbers rejected; change history kept; units searchable/filterable.

**Independent Test**: Create a building with 10 units; a second unit 1 is rejected with a Persian conflict message; editing a unit produces a history record (quickstart Scenario 1).

### Tests for User Story 2 (write first, must FAIL)

- [x] T028 [P] [US2] Go integration test for building/unit CRUD incl. duplicate-number 409, search/filter, audit trail in `backend/internal/building/building_integration_test.go`

### Implementation for User Story 2

- [x] T029 [P] [US2] Create migration `backend/migrations/0002_buildings.up.sql`: `buildings`, `units` (unique index `(building_id, number)` where `deleted_at IS NULL`), `user_buildings` per data-model.md
- [x] T030 [P] [US2] Implement models + repositories for buildings and units in `backend/internal/building/` (GORM, soft delete)
- [x] T031 [US2] Implement building/unit service in `backend/internal/building/service.go`: validation (area > 0, statuses), duplicate-number check, manager-scope filter; auto-grant creator in `user_buildings`
- [x] T032 [US2] Implement handlers in `backend/internal/building/handlers.go`: buildings CRUD, unit list (`?q=&block=&floor=&status=` paginated), unit CRUD, `GET /units/{id}/history` (audit entries), all audited via T013 decorator, Persian error messages
- [x] T033 [P] [US2] Implement Flutter building screens in `mobile/lib/features/buildings/`: building list + create/edit form
- [x] T034 [P] [US2] Implement Flutter unit screens in `mobile/lib/features/buildings/`: searchable/filterable unit list, unit create/edit form (all fields incl. parking/storage numbers), unit change-history view

**Checkpoint**: User Story 2 functional — full building/unit registry independently testable via UI and API.

---

## Phase 5: User Story 3 — Manage Owners and Residents with Occupancy History (Priority: P1)

**Goal**: People linked to units (owner/tenant/non-resident owner) with dated occupancy; resident changes preserve history; occupant count tracked independently with history.

**Independent Test**: Register a tenant for unit 1, replace them with a new tenant — previous occupancy and occupant-count history remain intact (quickstart Scenario 1, step 3).

### Tests for User Story 3 (write first, must FAIL)

- [x] T035 [P] [US3] Go integration test for occupancy lifecycle (new active tenant closes previous, history preserved, occupant-count as-of reads) in `backend/internal/building/occupancy_integration_test.go`

### Implementation for User Story 3

- [x] T036 [P] [US3] Create migration `backend/migrations/0003_people.up.sql`: `persons`, `occupancies` (end-dating only), `occupant_count_history` per data-model.md
- [x] T037 [P] [US3] Implement models + repositories in `backend/internal/building/` for persons, occupancies, occupant-count history
- [x] T038 [US3] Implement occupancy service in `backend/internal/building/occupancy_service.go`: one active tenant per unit rule, national-id checksum validation, occupant-count as-of query (max effective_from ≤ reference date), resident-change audit (`resident.changed`)
- [x] T039 [US3] Implement handlers in `backend/internal/building/occupancy_handlers.go`: persons CRUD, occupancy add/end-date, occupant-count GET/POST per contracts/api.md
- [x] T040 [P] [US3] Implement Flutter person management in `mobile/lib/features/buildings/`: person list/create/edit per building
- [x] T041 [P] [US3] Implement Flutter occupancy + occupant-count UI in `mobile/lib/features/buildings/`: unit occupancy tab (add person with relationship + Jalali start date via T018 picker, end occupancy), occupant-count editor with history list

**Checkpoint**: User Story 3 functional — people/occupancy data complete and historical; charge-engine inputs available.

---

## Phase 6: User Story 4 — Charge Formula, Calculate, Review, Issue Invoices (Priority: P1) 🎯 CORE

**Goal**: The charge engine: periods with multiple cost items (methods A–F), per-unit calculation with late fees/prior debt/credits, reviewable preview, immutable issuance, explicit corrections.

**Independent Test**: Spec §25 scenario — 10 units, 10,000,000 equal + 8,000,000 per-occupant (20 occupants, unit 1 = 4) → unit 1 invoice = 2,600,000 verifiable in preview; issued invoices unchanged after base-data edits (quickstart Scenarios 2–3).

### Tests for User Story 4 (write first, must FAIL)

-[x] T042 [P] [US4] Pure table-driven engine tests in `backend/internal/billing/engine/engine_test.go`: all six methods, spec §25 (2,600,000), largest-remainder reconciliation (Σ shares ≡ total, BR-09), combined weights, zero-participating-factor errors, vacant-unit inclusion rules
-[x] T043 [P] [US4] Go integration test for period lifecycle in `backend/internal/billing/billing_integration_test.go`: draft→calculate→preview→issue→close, reopen only before issue, invoice immutability after occupant-count/area change, cancel preserves row, adjustments don't mutate originals

### Implementation for User Story 4

-[x] T044 [P] [US4] Implement pure charge engine in `backend/internal/billing/engine/` (ZERO infra imports): method types (equal, per_occupant, per_area, fixed, specific_units, combined), rational share computation, largest-remainder rounding (deterministic tie-break: lowest unit id), late-fee computation (none/fixed/percent/per_day) per research.md R7
-[x] T045 [P] [US4] Create migration `backend/migrations/0004_billing.up.sql`: `billing_periods`, `cost_items`, `cost_item_shares`, `invoices` (+ immutability trigger guard on amount columns when issued), `invoice_items`, `invoice_adjustments` per data-model.md
-[x] T046 [P] [US4] Implement models + repositories for periods, cost items, shares, invoices, items, adjustments in `backend/internal/billing/`
-[x] T047 [US4] Implement calculation service in `backend/internal/billing/calc_service.go`: snapshot inputs (occupant counts/areas/participation/balances at calculation time) into `inputs_snapshot`, produce `cost_item_shares` + draft invoices with prior debt, late fee, credit, final amount (FR-015 formula), sequential invoice numbering per building
-[x] T048 [US4] Implement period state machine + issue/cancel/correct services in `backend/internal/billing/period_service.go`: calculate (→calculated), reopen (calculated→draft), issue (freezes, emits `invoice_issued` notifications via T014, audited), close, cancel invoice (row preserved, BR-10), adjustments (debit/credit notes, originals untouched)
-[x] T049 [US4] Implement handlers in `backend/internal/billing/handlers.go`: periods CRUD (edit only in draft), cost items CRUD (draft only), `POST /periods/{id}/calculate`, `GET /periods/{id}/preview` (per-unit exact+rounded shares with reconciliation flag, BR-08), issue/reopen/close, invoice cancel/adjustments, manager invoice list/detail — Persian messages, 409 on illegal transitions
-[x] T050 [P] [US4] Implement Flutter billing screens in `mobile/lib/features/billing/`: period list + create/edit (title, Jalali start/end/due via T018, late-fee type/value)
-[x] T051 [P] [US4] Implement Flutter cost-item editor in `mobile/lib/features/billing/`: title, amount (Persian digits), method selector incl. combined weights editor, include-vacant toggle, unit picker for specific_units
-[x] T052 [US4] Implement Flutter calculate/preview screen in `mobile/lib/features/billing/`: run calculation, per-unit breakdown table (exact + rounded shares, Σ reconciliation indicator), issue confirmation flow
-[x] T053 [P] [US4] Implement Flutter invoice detail screen (shared manager/resident) in `mobile/lib/features/charges/`: items by kind (charge/late fee/adjustment), method labels, Jalali dates, status chip, cancel/adjust actions (manager only)

**Checkpoint**: CORE story functional — spec §25 produces exactly 2,600,000; immutability guarantees hold.

---

## Phase 7: User Story 5 — Record Payments and Track Unit Balances (Priority: P2)

**Goal**: Manual + online payments, partial payments, surplus credit; per-unit balance always reconciles to the §9 formula.

**Independent Test**: 2,600,000 invoice + 1,500,000 manual payment → status `partial`, 1,100,000 outstanding; online remainder → `paid` with receipt; overpayment → credit (quickstart Scenario 4).

### Tests for User Story 5 (write first, must FAIL)

- [x] T054 [P] [US5] Go integration tests in `backend/internal/payment/payment_integration_test.go`: partial payment status transitions, overpayment→credit, balance formula reconciliation, gateway verify uses DB amount (mock gateway success/failure/amount-mismatch)

### Implementation for User Story 5

- [x] T055 [P] [US5] Create migration `backend/migrations/0005_payments.up.sql`: `payments`, `unit_balances` (one row per unit, BR-01) per data-model.md
- [x] T056 [P] [US5] Implement models + repositories in `backend/internal/payment/`
- [x] T057 [US5] Implement balance service in `backend/internal/payment/balance_service.go`: transactional recompute on payment/adjustment/invoice events; components + `balance` per spec §9; never hard-delete payments (reversal via status)
- [x] T058 [US5] Implement `PaymentGateway` interface + Zarinpal adapter (v4 API, Toman→Rial ×10 inside adapter, amount from DB at verify) + mock gateway (dev) in `backend/internal/payment/gateway/` per research.md R5
- [x] T059 [US5] Implement payment services + handlers in `backend/internal/payment/`: manual recording (manager, audited), `POST /invoices/{id}/pay` start, `GET /payments/callback` verify (marks verified/failed, updates invoice status + balance, emits `payment_recorded` notification), manager ledger + `GET /me/payments`, `GET /units/{id}/balance`
- [x] T060 [P] [US5] Implement Flutter manager payment screens in `mobile/lib/features/payments/`: record manual payment (Jalali paid_at via T018), payment ledger with filters
- [x] T061 [P] [US5] Implement Flutter resident payment flow in `mobile/lib/features/payments/`: outstanding amount + pay button → gateway redirect/webview → result → receipt view; payment history list

**Checkpoint**: User Story 5 functional — the full financial collection loop works end-to-end.

---

## Phase 8: User Story 6 — Register Expenses and Financial Report (Priority: P2)

**Goal**: Manager records categorized expenses with receipts; basic financial report with consistent totals.

**Independent Test**: After recording expenses, report totals (income/expense/net/debt) match underlying records (quickstart Scenario 5).

### Tests for User Story 6 (write first, must FAIL)

- [x] T062 [P] [US6] Go integration tests in `backend/internal/expense/expense_integration_test.go`: CRUD + receipt file, approval statuses, financial-report aggregation accuracy vs seeded payments/expenses/balances

### Implementation for User Story 6

- [x] T063 [P] [US6] Create migration `backend/migrations/0006_expenses.up.sql`: `expenses` per data-model.md
- [x] T064 [P] [US6] Implement models + repository in `backend/internal/expense/`
- [x] T065 [US6] Implement expense service + handlers in `backend/internal/expense/`: CRUD (soft delete, audited, receipt via T010 files), category enum, approval workflow, `GET /buildings/{id}/financial-report?month=YYYY-MM` aggregating monthly income/expense/net, total debt, total payments, total expenses (FR-027)
- [x] T066 [P] [US6] Implement Flutter expense screens in `mobile/lib/features/expenses/`: list with category/approval filters, create/edit form (categories → Persian labels, Jalali date, receipt attach)
- [x] T067 [P] [US6] Implement Flutter financial report screen in `mobile/lib/features/expenses/`: month selector (Jalali), summary cards with Persian-digit Toman amounts

**Checkpoint**: User Story 6 functional — expenses and report independently testable.

---

## Phase 9: User Story 7 — Submit and Manage Maintenance Requests (Priority: P3)

**Goal**: Residents submit requests (photo, priority); manager runs the status workflow with assignee/cost; resident tracks without contacting the manager.

**Independent Test**: Resident submits "آسانسور لرزش دارد" (urgent, photo) → manager processes to closed → resident sees every status change (quickstart Scenario 6).

### Tests for User Story 7 (write first, must FAIL)

- [x] T068 [P] [US7] Go integration tests in `backend/internal/maintenance/maintenance_integration_test.go`: legal state transitions only (409 on backwards/illegal), notifications emitted per transition, resident isolation (own requests only), audit entries

### Implementation for User Story 7

- [x] T069 [P] [US7] Create migration `backend/migrations/0007_maintenance.up.sql`: `maintenance_requests` per data-model.md
- [x] T070 [P] [US7] Implement models + repository in `backend/internal/maintenance/`
- [x] T071 [US7] Implement request service in `backend/internal/maintenance/service.go`: state machine (new→under_review→in_progress→done→closed, early close allowed), assignee/cost/notes updates, per-transition notification to submitter + audit (`maintenance.status_changed`)
- [x] T072 [US7] Implement handlers in `backend/internal/maintenance/handlers.go`: resident `POST /me/maintenance-requests` + list with status timeline; manager list (filters) + `PATCH /maintenance-requests/{id}` per contracts/api.md
- [x] T073 [P] [US7] Implement Flutter resident maintenance screens in `mobile/lib/features/maintenance/`: submit form (category/priority Persian selectors, photo attach, location) + my-requests list with status timeline
- [x] T074 [P] [US7] Implement Flutter manager maintenance screens in `mobile/lib/features/maintenance/`: request list with filters, detail with assignee picker, cost entry, status actions, close

**Checkpoint**: User Story 7 functional — full maintenance workflow independently testable.

---

## Phase 10: User Story 8 — Publish Announcements and Notifications (Priority: P3)

**Goal**: Manager publishes targeted announcements (building/block/floor/unit) with expiry; residents see only targeted ones; unread tracking; event notifications work.

**Independent Test**: Block-A announcement visible only to block-A residents; unread badge counts; `invoice_issued` notification already observable (quickstart Scenario 6 step 3).

### Tests for User Story 8 (write first, must FAIL)

- [x] T075 [P] [US8] Go integration tests in `backend/internal/announcement/announcement_integration_test.go`: audience targeting (all/block/floor/unit), publish/expire window visibility, read-tracking/unread counts

### Implementation for User Story 8

- [x] T076 [P] [US8] Create migration `backend/migrations/0008_announcements.up.sql`: `announcements`, `announcement_reads` per data-model.md
- [x] T077 [P] [US8] Implement models + repository in `backend/internal/announcement/`
- [x] T078 [US8] Implement announcement service + handlers in `backend/internal/announcement/`: manager CRUD (audited), audience-scope resolution for residents (join occupancies/units), publish/expire window filter, `POST /announcements/{id}/read`, emits `announcement_published` notifications; resident list `GET /me/announcements`
- [x] T079 [P] [US8] Implement Flutter manager announcement screens in `mobile/lib/features/announcements/`: list + publish form (audience type/value, Jalali publish/expire pickers via T018, attachment)
- [x] T080 [P] [US8] Implement Flutter resident announcements + notification center in `mobile/lib/features/announcements/` and `mobile/lib/features/notifications/`: announcement list/detail (targeted only), unread badge, mark-read; notification center (list, unread filter, deep links via go_router, mark-all-read)

**Checkpoint**: User Story 8 functional — communication loop complete.

---

## Phase 11: User Story 9 — Resident Self-Service Panel (Priority: P2)

**Goal**: Resident home answers "what do I owe, by when, what happened to my request?" — with the full main menu (خانه، شارژها، پرداخت‌ها، تعمیرات، اطلاعیه‌ها، پروفایل).

**Independent Test**: A resident with one issued invoice and one open request logs in and answers both questions entirely from the panel (quickstart Scenario 8.1).

### Tests for User Story 9 (write first, must FAIL)

- [ ] T081 [P] [US9] Go integration test for `GET /me/home` in `backend/internal/dashboard/home_test.go`: data consistency with underlying invoices/requests/announcements + strict resident isolation

### Implementation for User Story 9

- [ ] T082 [US9] Implement `GET /me/home` aggregation endpoint in `backend/internal/dashboard/handlers.go`: payable amount, latest invoice (amount/due date Jalali-ready ISO/status), open request count + latest status, latest announcements, unread announcement count (FR-034)
- [ ] T083 [US9] Implement Flutter resident home screen in `mobile/lib/features/home/`: financial status card, requests summary card, announcements card with unread badge
- [ ] T084 [US9] Implement Flutter resident shell + remaining menu in `mobile/lib/features/home/` + `mobile/lib/features/charges/` + `mobile/lib/features/profile/`: bottom navigation with 6 Persian menu items, resident charges list (reusing T053 detail), payment history entry point, profile view/edit

**Checkpoint**: User Story 9 functional — resident self-service complete (third defining outcome of P0).

---

## Phase 12: User Story 10 — Manager Dashboard with Alerts (Priority: P3)

**Goal**: One-glance building status: cards, quick actions, alerts (debtor units, past-due invoices, open requests, pending expenses).

**Independent Test**: With 7 debtor units and 3 open requests seeded, dashboard cards and alerts show exactly those numbers (quickstart Scenario 8.2).

### Tests for User Story 10 (write first, must FAIL)

- [ ] T085 [P] [US10] Go integration test for `GET /buildings/{id}/dashboard` in `backend/internal/dashboard/dashboard_test.go`: card/alert counts match seeded state

### Implementation for User Story 10

- [ ] T086 [US10] Implement dashboard aggregation endpoint in `backend/internal/dashboard/dashboard_service.go`: unit count, debtor units, total debt, month income/expense, open requests + alerts list + quick-action targets (FR-035)
- [ ] T087 [US10] Implement Flutter manager dashboard in `mobile/lib/features/dashboard/`: card grid (Persian-digit Toman), alerts section, quick-action buttons deep-linking to issue charge / record expense / record payment / record request / send announcement

**Checkpoint**: All 10 user stories functional and independently testable.

---

## Phase 13: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T088 [P] Implement scheduled jobs in `backend/internal/platform/jobs/`: due-soon notifications, overdue notifications + invoice `expired` status (runs daily; testable via injected clock)
- [ ] T089 [P] Implement FCM adapter for `PushNotifier` in `backend/internal/notification/fcm.go` (best-effort, config-gated; in-app remains the guaranteed channel per FR-032)
- [ ] T090 [P] Create `backend/Dockerfile` (multi-stage Go build) + root `README.md` with run instructions mirroring quickstart.md setup
- [ ] T091 Security hardening pass: verify object-level authorization on every `/me/*` and manager route, OTP/payment rate limits, file-upload validation, audit-log completeness sweep vs FR-038 list
- [ ] T092 [P] Add Flutter widget tests asserting `fa` locale, RTL layout, and Jalali-only date fields across key screens in `mobile/test/`
- [ ] T093 Implement Flutter integration tests for quickstart scenarios 2, 4, 6 in `mobile/integration_test/` against a dev backend
- [ ] T094 Verify performance goal: invoice calculation for a 50-unit building completes within seconds (FR-021) — add timing assertion to `backend/internal/billing/billing_integration_test.go`
- [ ] T095 Run the full quickstart.md validation (all 8 scenarios + §26's 14 MVP acceptance items) and record results

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — **BLOCKS all user stories**
- **User Stories (Phases 3–12)**: All depend on Phase 2 completion; then parallelizable or sequential in priority order
- **Polish (Phase 13)**: Depends on the stories it hardens (T088 after US4/US5; T089 after US8; T093–T095 after US1–US10)

### User Story Dependencies

- **US1 (auth)**: Foundation only — no story dependencies. All other stories consume its middleware/context.
- **US2 (buildings/units)**: Foundation + US1 (manager context). No other story dependency.
- **US3 (people/occupancy)**: US2 (units exist). Independently testable.
- **US4 (charge engine)**: US2 + US3 (units, occupant counts). The core story — invoices exist here.
- **US5 (payments/balances)**: US4 (invoices to pay). Uses notification service from foundation.
- **US6 (expenses/report)**: US2 (+ payments data for report accuracy). Can start after US2; full report needs US5.
- **US7 (maintenance)**: US1 + US2 (building/unit context). Independent of financial stories.
- **US8 (announcements)**: US1 + US2. Independent.
- **US9 (resident panel)**: Aggregates US4, US5, US7, US8 data — implement last among stories for full value, but each section degrades gracefully if earlier stories are incomplete.
- **US10 (manager dashboard)**: Aggregates US4–US8 data — same graceful-degradation note.

### Within Each User Story

- Tests FIRST (must fail before implementation)
- Migrations/models before services; services before handlers; backend before Flutter screens for that story
- Story complete (backend + mobile) before moving to the next priority

### Parallel Opportunities

- All [P] tasks within a phase run in parallel (different files)
- Backend and Flutter tracks of a story run in parallel once the story's API contract tasks are done
- Different user stories can be worked on in parallel by different developers after Phase 2 (respecting the story dependencies above)

---

## Parallel Example: User Story 4

```bash
# Launch all US4 test tasks together:
Task: T042 "Pure table-driven engine tests in backend/internal/billing/engine/engine_test.go"
Task: T043 "Period lifecycle integration test in backend/internal/billing/billing_integration_test.go"

# Launch all US4 migration/model tasks together:
Task: T044 "Pure charge engine in backend/internal/billing/engine/"
Task: T045 "Migration 0004_billing in backend/migrations/"
Task: T046 "Billing models + repositories in backend/internal/billing/"

# Launch all US4 Flutter screens together (after T049):
Task: T050 "Period screens in mobile/lib/features/billing/"
Task: T051 "Cost-item editor in mobile/lib/features/billing/"
Task: T053 "Invoice detail in mobile/lib/features/charges/"
```

---

## Implementation Strategy

### MVP First (User Stories 1–4)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phases 3–6: US1 auth → US2 registry → US3 occupancy → US4 charge engine
4. **STOP and VALIDATE**: quickstart Scenarios 1–3 pass (spec §25 = 2,600,000; immutability holds)
5. Deploy/demo — the manager can already calculate and issue correct invoices, the product's core promise

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. US1–US4 → validate → **MVP demo (charge calculation & issuance)**
3. US5 → validate → collection loop complete (the first P0 outcome: "محاسبه و وصول قبض")
4. US6 + US7 + US8 → validate → operations & communication (second outcome: "مدیریت هزینه و خرابی")
5. US9 + US10 → validate → resident self-service + manager overview (third outcome: "پیگیری ساکن بدون تماس با مدیر")
6. Phase 13 polish → full quickstart.md + §26 acceptance

### Parallel Team Strategy

With multiple developers after Phase 2:

- Developer A: US2 → US3 → US4 (backend financial core)
- Developer B: US7 → US8 → US6 (operations track)
- Developer C: US1 Flutter screens → US9 → US10 (resident/manager UX track)
- US5 after US4 lands; integration checkpoints at each story boundary

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] labels map to spec.md user stories US1–US10 for traceability
- Every story phase is independently completable and testable — stop at any checkpoint to validate
- Charge engine (T044) must remain a pure package with zero infrastructure imports (plan.md structure decision)
- All UI strings Persian from `mobile/lib/core/l10n/fa.arb`; all server user-facing messages Persian; Jalali picker (T018) is the only date input component
- Commit after each task or logical group; keep migrations sequential (0001–0008 order above)
