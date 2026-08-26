# Research: Building Management MVP (P0)

**Feature**: specs/001-building-management-mvp
**Date**: 2026-08-26
**Status**: Complete — all NEEDS CLARIFICATION items resolved

## Research Tasks

| # | Unknown / Decision | Status |
|---|--------------------|--------|
| R1 | Go HTTP framework choice | Resolved |
| R2 | Go data-access approach (ORM vs query-first) | Resolved |
| R3 | Database migrations tooling | Resolved |
| R4 | OTP delivery over SMS (Iranian providers) | Resolved |
| R5 | Online payment gateway integration pattern | Resolved |
| R6 | Jalali (Persian) calendar handling | Resolved |
| R7 | Money representation & rounding reconciliation | Resolved |
| R8 | Auth token strategy for a mobile client | Resolved |
| R9 | Flutter state management & architecture | Resolved |
| R10 | Push notification channel for Iran/Android | Resolved |
| R11 | Invoice immutability pattern in PostgreSQL | Resolved |

---

## R1 — Go HTTP framework

- **Decision**: Gin (`github.com/gin-gonic/gin`)
- **Rationale**: The API is a JSON-heavy REST backend consumed by our own Flutter app. Gin offers the largest ecosystem, built-in struct-tag binding/validation (covers ~80% of endpoints without a separate validator wiring), mature middleware (CORS, rate limiting, auth), and the most community examples — best for MVP velocity with a mixed-experience team.
- **Alternatives considered**: Echo (cleaner centralized error handling, richer built-ins; rejected for smaller community/examples), chi + net/http (zero deps, stdlib-aligned; rejected — more hand-written binding/validation boilerplate for a ~70-route API), Fiber (fasthttp incompatibility tax not justified; a building-management API is DB-bound, not router-bound).

## R2 — Data access

- **Decision**: GORM (`gorm.io/gorm`) with PostgreSQL driver, wrapped in a repository layer per domain module. Hot/financial paths (invoice calculation, balance queries) may use raw SQL via GORM's `Raw` when needed for correctness and performance.
- **Rationale**: MVP speed with a rich domain model (15 entities); repository layer keeps GORM swappable and keeps business logic (charge engine) framework-free and unit-testable.
- **Alternatives considered**: sqlc + pgx (best type-safety for money SQL; rejected — much higher codegen ceremony for an MVP with a still-evolving schema), plain database/sql (too much boilerplate).

## R3 — Migrations

- **Decision**: golang-migrate (`github.com/golang-migrate/migrate/v4`) with plain SQL migration files in `backend/migrations/`.
- **Rationale**: De-facto standard, SQL-first (full control over constraints critical for financial data — CHECK constraints, unique indexes), CI-friendly, no ORM lock-in.
- **Alternatives considered**: GORM AutoMigrate (rejected — insufficient constraint control for financial tables), Atlas (heavier than needed).

## R4 — OTP SMS delivery

- **Decision**: Provider-agnostic `SmsSender` interface with a Kavenegar adapter as the default implementation and a **dev-mode console/log sender** for local development and tests.
- **Rationale**: Iranian SMS providers (Kavenegar, SMS.ir, Farapayamak) all expose simple REST verify/send APIs; the interface isolates the choice so credentials/provider can change via config without touching auth logic. OTP: 6 digits, 2-minute validity, max 3 attempts per code, resend throttle 60s, per-phone rate limit.
- **Alternatives considered**: Direct Kavenegar SDK binding everywhere (rejected — vendor lock-in), email OTP (rejected — Iranian users are mobile-first).

## R5 — Payment gateway

- **Decision**: Provider-agnostic `PaymentGateway` interface (Start → redirect URL; Verify → transaction id) with a **Zarinpal adapter** (v4 JSON API, `payment.zarinpal.com/pg/v4/payment/{request,verify}.json`, sandbox at `sandbox.zarinpal.com`) as the default, plus a **mock gateway for tests**.
- **Rationale**: Zarinpal is Iran's most popular gateway with a documented sandbox and stable REST API. Iranian apps frequently switch gateways for fees/reliability, so the interface-first design (verified as a common pattern across gateway SDKs) lets us swap/add IDPay, Zibal, or Vandar via config. Critical implementation rule learned from research: Zarinpal does not return the amount in its callback — the verify step must pass the amount **from our own database** to prevent amount-mismatch attacks. Amounts are Toman in our domain; Zarinpal API uses Rial — conversion (×10) happens only inside the adapter.
- **Alternatives considered**: IDPay adapter first (equally viable; Zarinpal chosen for sandbox quality and popularity), building directly against one provider without an interface (rejected).

## R6 — Jalali calendar

- **Decision**: Store all dates as **UTC timestamps / date types (Gregorian)** in PostgreSQL. Convert to Jalali only at the presentation edges: `github.com/yaa110/go-persian-calendar` (ptime) on the backend for any server-side period logic that needs Jalali month boundaries; `shamsi_date` package in Flutter for display. The API transmits ISO-8601 dates; the app renders **and accepts** Persian dates — every date input in the UI is a **Jalali date picker**, converted to ISO-8601 only at the API boundary. Users never see or enter a Gregorian date.
- **Rationale**: Keeping the storage calendar Gregorian avoids a whole class of date-math bugs (billing periods, due-date comparisons, "past due" alerts). Jalali conversion is a pure display/input concern, and Jalali-only picking matches the Persian-only product scope.
- **Alternatives considered**: Storing Jalali strings (rejected — unusable for comparisons/arithmetic), backend-only conversion with Jalali strings in the API (rejected — inflexible for the client).

## R7 — Money & rounding

- **Decision**: Store all amounts as **BIGINT in Toman** (no fractions). Charge-share rounding uses the **largest remainder method**: compute exact shares as rationals, floor each to Toman, distribute the remaining Toman units one-by-one to units with the largest fractional remainders (tie-break: lowest unit id) so Σ shares ≡ cost total exactly (BR-09).
- **Rationale**: Integer Toman is the natural unit in Iranian building management; floats are unacceptable for money. Largest-remainder guarantees the reconciliation invariant and is deterministic and auditable.
- **Alternatives considered**: Rounding each share half-up and adjusting the last unit (rejected — biases the last unit), NUMERIC(18,2) (rejected — no real sub-Toman need; integers simplify the engine).

## R8 — Auth tokens

- **Decision**: Short-lived **JWT access token (15 min)** + long-lived **opaque refresh token (rotating, 30 days)** stored hashed in the DB, sent by the Flutter app in `Authorization: Bearer`. Role/context (user id, role: manager/resident, permitted building/unit ids) resolved server-side per request from claims + DB — claims carry only identity, never authorization data.
- **Rationale**: Standard for mobile APIs; rotation + hashed storage limits refresh-token leakage damage; server-side permission resolution keeps resident-data isolation (FR-037) enforceable in one place.
- **Alternatives considered**: Session cookies (rejected — mobile app, not browser), long-lived JWT only (rejected — no revocation), full opaque-token auth (viable but adds a DB hit per request).

## R9 — Flutter architecture

- **Decision**: Flutter (Android-first) with **Riverpod** (state management + DI), **go_router** (navigation), **Dio** (HTTP with auth interceptor & token refresh), **freezed + json_serializable** (immutable models), **shamsi_date** (Jalali display), **persian_datetime_picker** (Jalali date input). Feature-first folder structure (`lib/features/<feature>/`), repository pattern mirroring the backend domains.
- **Localization decision**: **Persian-only UI** — the app is configured as a single-locale (`fa`, RTL) product for this phase: `flutter_localizations` with only `fa` supported (English is explicitly out of scope for P0), RTL layout enforced app-wide, Persian digits (`۰۱۲۳۴۵۶۷۸۹`) in all displayed numbers and amounts, and **Jalali as the default and only date picker** (`persian_datetime_picker`) — no Gregorian picker anywhere in the UI. All user-facing strings (labels, errors, empty states) are written directly in Persian via a single `fa` arb/string file so a second locale can be added later without refactoring.
- **Rationale**: Riverpod offers compile-safe DI and testability without Bloc's boilerplate; go_router fits the two-role app shell (manager vs resident navigation stacks); Dio interceptors centralize token refresh and error mapping. Android-only for P0 (per spec — native app and other platforms out of scope). Persian-first with Jalali-only date entry matches the user base; keeping one locale removes translation overhead and reduces P0 surface area, while the single string-file keeps future localization possible.
- **Alternatives considered**: Bloc (equally valid; more ceremony), Provider (superseded by Riverpod), MVVM with stacked (smaller ecosystem), multi-locale fa+en from day one (rejected — user explicitly scoped this phase to Persian only), Material default Gregorian picker with Jalali conversion layer (rejected — confusing UX for the target user).

## R10 — Push notifications

- **Decision**: **FCM (Firebase Cloud Messaging)** as the best-effort push channel behind a `PushNotifier` interface, with **in-app notifications as the guaranteed channel** (polling on app foreground + notification list screen). FCM is optional-at-runtime: if unconfigured, all flows still satisfy spec FR-032/FR-033 via in-app delivery.
- **Rationale**: The spec explicitly makes push conditional on infrastructure. FCM is the default Android choice, but Google services reach from Iran can be unreliable — hence in-app-first design and an interface so a local alternative (e.g., a lightweight self-hosted push such as ntfy/UnifiedPush) can be added later without rework.
- **Alternatives considered**: SMS notifications (rejected — cost; may be revisited post-P0), self-hosted UnifiedPush only (rejected — requires user-side app installation).

## R11 — Invoice immutability (BR-03/BR-05)

- **Decision**: Issued invoices are **append-only snapshots**: at calculation time, each invoice row copies the inputs used (occupant count, area, participating units) into invoice line items, so later changes to unit data cannot affect it. Corrections create a new **invoice adjustment** record (credit/debit note referencing the original invoice) — the original row is never updated in amount terms. DB rule: no UPDATE on `invoices.amount`/`invoice_items` once `status = issued` (enforced in service layer + trigger guard).
- **Rationale**: Satisfies FR-017, BR-03, BR-05, BR-10 and the "no invoice changes after base-data change" reliability requirement with a simple, auditable pattern; adjustment notes give an explicit correction operation as required.
- **Alternatives considered**: Event-sourced billing (rejected — overkill for P0), recalculating from live data with versioning (rejected — complex, riskier).

## Summary of Resolved Technical Context

| Item | Choice |
|------|--------|
| Backend | Go (Gin, GORM, golang-migrate) |
| Database | PostgreSQL 16 |
| Auth | Mobile OTP (SMS interface + Kavenegar), JWT access + rotating refresh |
| Payments | `PaymentGateway` interface + Zarinpal adapter + mock |
| Charge engine | Pure-Go module, integer Toman, largest-remainder rounding |
| Calendar | Gregorian storage, Jalali presentation (ptime / shamsi_date) |
| Frontend | Flutter Android, Riverpod + go_router + Dio + freezed |
| Notifications | In-app guaranteed + FCM best-effort behind interface |
| Testing | Go: `testing` + testify + testcontainers-postgres; Flutter: flutter_test + mocktail, integration_test for E2E |
| Deploy target | Single Linux server / Docker container running the Go binary; Android APK distribution |
