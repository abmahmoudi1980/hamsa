# Implementation Plan: Building Management MVP (P0)

**Branch**: `001-building-management-mvp` | **Date**: 2026-08-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-building-management-mvp/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

A building-management product with three pillars: (1) charge calculation and collection — a charge engine supporting six calculation methods (equal, per-occupant, per-area, fixed, specific-units, combined) that produces immutable per-unit invoices with prior debt, late fees, and credits; (2) daily operations — expenses, maintenance requests, announcements; (3) resident self-service — invoices, payments, request tracking without contacting the manager.

**Technical approach** (from [research.md](research.md)): a Go (Gin + GORM + golang-migrate) REST API backed by PostgreSQL 16, with the charge engine as a pure-Go module (integer Toman, largest-remainder rounding, snapshot-based invoice immutability), password auth (bcrypt) with manager-issued one-time invite codes plus JWT access + rotating refresh tokens, a provider-agnostic payment gateway (Zarinpal adapter), and a Flutter Android client (Riverpod + go_router + Dio) with role-based navigation (manager / resident). All dates stored Gregorian/UTC; Jalali con...

## Technical Context

**Language/Version**: Go 1.23+ (backend), Dart 3.x / Flutter 3.24+ (Android client)

**Primary Dependencies**:
- Backend: `gin-gonic/gin` (HTTP), `gorm.io/gorm` + `gorm.io/driver/postgres` (data access), `golang-migrate/migrate` (migrations), `golang-jwt/jwt/v5` (tokens), `yaa110/go-persian-calendar` (Jalali), `go-playground/validator` (via Gin binding), `google/uuid`
- Frontend: `flutter_riverpod` (state/DI), `go_router` (navigation), `dio` (HTTP + interceptors), `freezed`/`json_serializable` (models), `shamsi_date` (Jalali display), `persian_datetime_picker` (Jalali date input — **the only date picker in the app**), `flutter_localizations` (fa locale, RTL), `persian_fonts`/`vazir` or bundled Vazirmatn font, `flutter_local_notifications` (local display of in-app notifications)

**Localization / UI Language**: **Persian (Farsi) only** — single locale `fa`, full RTL layout, Persian digits for all displayed numbers and amounts, Jalali date picker as the default and only date entry component. No English language support in this phase; all user-facing strings (UI labels, error messages, empty states) are Persian. All app copy lives in a single `fa` string file so additional locales can be added post-P0 without refactoring. The API remains locale-neutral (ISO-8601 dates, UTF-8 JSON); server-returned user-facing messages are Persian.

**Storage**: PostgreSQL 16 (single schema, SQL migrations). Files (receipt images, request photos, announcement attachments) stored on local disk/object storage with paths in DB.

**Testing**:
- Backend: Go `testing` + `testify`; repository/integration tests against ephemeral PostgreSQL (testcontainers-go); charge-engine unit tests as pure functions (table-driven, including the spec's sample scenario); mock `PaymentGateway`/`SmsSender`/`PushNotifier` interfaces
- Frontend: `flutter_test` + `mocktail` (unit/widget), `integration_test` for E2E flows against a running backend

**Target Platform**: Linux server (single Go binary, Docker deployable) + Android 8.0+ (Flutter)

**Project Type**: Web-service API + mobile-app client

**Performance Goals**: Invoice calculation for a 50-unit building completes within seconds (FR-021); API p95 < 500 ms for standard CRUD on typical hardware; app cold start to home screen < 3 s on mid-range Android.

**Constraints**: Resident data isolation enforced server-side on every request (FR-037); issued invoices immutable except via explicit adjustment (BR-03/BR-05); financial records survive user/unit deletion (soft-delete/archive only); amounts integer Toman; Persian-only RTL UI with Jalali date input (no English locale in P0).

**Scale/Scope**: ~15 entities, ~70 API endpoints, ~25 Flutter screens (all Persian/RTL, single `fa` locale); single-building-heavy usage (a few buildings per manager, ≤ 200 units per building in P0).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

`.specify/memory/constitution.md` exists but is an **unfilled template** (placeholders only — no ratified principles, constraints, or governance rules). Therefore:

- **Pre-research gate**: PASS (no active constraints to violate)
- **Post-design re-check (after Phase 1)**: PASS — no constitution gates apply; general engineering principles from the spec (immutability, audit, data isolation) were carried into the design instead
- **Recommendation**: Ratify the constitution before `/speckit.tasks` if the team wants hard governance gates during implementation

## Project Structure

### Documentation (this feature)

```text
specs/001-building-management-mvp/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── api.md           # REST API contract
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── server/
│       └── main.go              # entrypoint: config, DI wiring, router, migrations on start
├── internal/
│   ├── auth/                    # password login, invite codes, JWT issue/verify, refresh rotation, login handlers
│   ├── building/                # buildings, units, people, occupancies, occupant-count history
│   ├── billing/                 # billing periods, cost items, charge engine, invoices, adjustments
│   │   └── engine/              # PURE Go charge calculation (no DB/HTTP deps) + rounding
│   ├── payment/                 # payments, unit balances, gateway interface + zarinpal adapter
│   ├── expense/                 # expenses, categories, financial report
│   ├── maintenance/             # maintenance requests + status workflow
│   ├── announcement/            # announcements, audience targeting
│   ├── notification/            # in-app notifications + PushNotifier interface (FCM adapter)
│   ├── audit/                   # audit-log middleware/service
│   ├── platform/                # config, db (gorm), httpx (error envelope, middleware), storage (files)
│   └── dashboard/               # manager dashboard aggregation + alerts
├── migrations/                  # golang-migrate SQL files (NNNN_name.up.sql / .down.sql)
├── go.mod
└── Dockerfile

mobile/
├── lib/
│   ├── main.dart
│   ├── core/                    # theme (RTL, Persian font/digits), router (go_router), dio client + interceptors, l10n (fa — single locale), Jalali date helpers + Jalali-only date picker wrapper
│   ├── features/
│   │   ├── auth/                # login, registration via invite code, role routing
│   │   ├── buildings/           # manager: buildings, units, people, occupancy
│   │   ├── billing/             # manager: periods, cost items, calculate, review, issue
│   │   ├── charges/             # resident: invoices list/detail
│   │   ├── payments/            # resident: pay flow; manager: record manual payment
│   │   ├── expenses/            # manager: expenses + financial report
│   │   ├── maintenance/         # resident: submit/track; manager: manage workflow
│   │   ├── announcements/       # publish (manager) / list+detail (resident)
│   │   ├── notifications/       # in-app notification center
│   │   ├── home/                # resident home panel
│   │   ├── dashboard/           # manager dashboard + quick actions + alerts
│   │   └── profile/
│   └── shared/                  # reusable widgets (money display, status chips, attachments)
├── integration_test/            # E2E flows against backend
└── pubspec.yaml
```

**Structure Decision**: Mobile + API layout (template Option 3). `backend/` is a single deployable Go module (modular monolith — domains as internal packages, not microservices; P0 scale does not justify service split). `backend/internal/billing/engine` is a deliberately pure package with zero infrastructure imports so the charge engine — the riskiest logic in the product — is exhaustively unit-testable. `mobile/` is a single Flutter app with feature-first folders; manager and resident experiences share one codebase and diverge at the router level by role.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations — table intentionally empty.
