# Implementation Plan: Web Frontend (Manager & Resident Web App)

**Branch**: `003-web-frontend` | **Date**: 2026-10-03 | **Status**: Phases 0–6 implemented
**Spec**: to be written (`spec.md` — Phase 0 produces it)

**Input**: Feature request — *"Review the building-management project and write a plan to
create a web frontend version."* Framework subsequently fixed to **Svelte** by the
requester (see [Decision D1 — resolved](#open-decisions-for-the-reviewer)).

---

## Summary

Hamsa today has a **complete, contract-frozen Go backend** (~60 REST endpoints, fully
implemented against `specs/001-building-management-mvp/contracts/api.md` plus the
`002-multi-manager-support` delta) and a **Flutter Android client** that consumes it.
There is **no browser client** — the only HTML in the repo is the static marketing
landing page in `website/`.

This plan builds `web/` — a dedicated **SvelteKit SPA** that consumes the existing API
unchanged. It serves both roles: a **manager console** (dense data tables,
charge-calculation review grid, bulk payment recording) and a **resident portal**
(invoices, online payment, maintenance requests, announcements).

**Technical approach** — treat the backend as a frozen contract and re-implement only
the presentation layer. Five decisions carry the plan:

1. **Dedicated SvelteKit SPA, not Flutter Web.** The API contract is the integration
   surface, not the Flutter code. The Flutter screens are mobile-idiomatic (bottom-nav
   shell, FAB, single-column forms), so reusing them means rewriting ~90% of the UI
   anyway — while paying Flutter Web's ~2.5–4 MB initial payload on exactly the
   Iranian mobile networks that make the APK the primary channel today. SvelteKit 3 in
   SPA mode gives file-based routing and per-route code splitting at a fraction of that
   payload.
2. **Serve from `app.hamsa-home.ir`, where `/api/` is already proxied by nginx.**
   Production becomes **same-origin** — zero CORS surface, no token in a cross-site
   request path, and the existing `deploy/nginx-hamsa.conf` needs only a `root` +
   SPA-fallback line added.
3. **Jalali and Persian-digit formatting come from the browser's own `Intl`.** No
   calendar dependency. One small, exhaustively-tested conversion module is the only
   hand-written date math, and the picker is the single date-input component
   (the rule `mobile/lib/core/datetime/jalali.dart` already enforces app-wide).
4. **Four pre-existing API/DTO gaps found during this review are fixed in Phase 0** —
   one of which is a live bug that makes resident payment history render empty on
   Android today. The web client hits all four, so they are fixed before, not after,
   the web work starts. **Done.**
5. **The web earns its keep on three surfaces mobile cannot do well**: the
   charge-preview reconciliation grid (BR-08/BR-09), sortable/filterable ledgers, and
   a print-optimised invoice (`@media print` → browser "Save as PDF" gives residents a
   shareable receipt for free).

### Progress

| Phase | Scope | State |
|---|---|---|
| 0 | Contract deltas, pre-flight backend fixes | **Done** — F1–F5 shipped with tests |
| 1 | Toolchain, RTL shell, design tokens, fonts, CI | **Done** |
| 2 | `lib/` layer: money, Jalali, digits, error envelope, refresh queue | **Done** — 78 tests |
| 3 | Auth screens, session restore, routing skeleton, design system | Partial — login + restore |
| 4 | Registry: buildings, units, people, occupancy | **Done** — 17baa74 |
| 5 | Billing: periods, cost items, calculation, preview grid | **Done** — 3fe7459 |
| 6 | Money movement: invoices, payments, balances, ledger, resident pay | **Done** — 7b5c64f |
| 7–9 | Expenses, maintenance, dashboards, launch | Not started |

---

## Technical Context

**Language/Version**: TypeScript 6 (strict, `noUncheckedIndexedAccess`), Node 22.17+

**Primary Dependencies**: `svelte` 5.57 (runes), `@sveltejs/kit` 3.0, `vite` 8,
`@sveltejs/adapter-static` 4 (SPA mode), `@tanstack/svelte-query` 6, `tailwindcss` 4
(logical properties), `zod` 4, `vitest` 4 (node + browser projects), `playwright` 1.60,
`subset-font` (font subsetting, build-time only)

> **SvelteKit 3 migration notes** (both discovered the hard way during Phase 1, and both
> caught by the toolchain rather than assumed):
> - `$lib` is **removed**. Source aliases are now Node subpath imports: `#lib`,
>   `#i18n`, declared in `package.json` → `imports` and mirrored in `tsconfig` → `paths`.
> - `svelte.config.js` is **gone**. The adapter and compiler options now live in
>   `vite.config.ts` under the `sveltekit()` plugin, and `$app/tsconfig` ships an
>   **empty** `paths` map, so aliases must be declared explicitly.

**Storage**: only the display user and the active-building preference live in
`localStorage`; the refresh token is in an httpOnly cookie, never JavaScript-reachable.
All domain data lives in PostgreSQL 16 behind the existing API. No IndexedDB, no offline
queue (out of scope — see Out of Scope).

**Testing**: `vitest` for the pure library layer (money, Jalali, digits, validation,
error-envelope mapping) and component tests; `playwright` E2E against a **real backend
+ real PostgreSQL** via `docker compose` — deliberately mirroring the backend's own
`integration_harness_test.go` culture ("real data, not mocks"). No MSW/mock server:
the whole point is contract fidelity.

**Target Platform**: Evergreen browsers (Chrome/Edge/Firefox/Safari ≥ 2 versions
back). Desktop-first, usable at 1024 px; mobile-web degrades to the resident portal
only (managers use the APK). `dir="rtl"`, `lang="fa"`.

**Project Type**: SPA + existing REST API — third client in a modular-monolith repo
(`backend/internal/*`, `mobile/lib/features/*`, new `web/src/lib/*`).

**Performance Goals**: initial JS ≤ **250 KB gzip** (route-level code splitting, no
big table/editor libs in the entry chunk); LCP < 2.5 s on Fast-3G; table sort/filter
interaction < 100 ms at 500 rows (client-side paging over server pages of 100);
no layout shift on async lists (fixed-height skeletons). *Current entry payload: ~43 KB
JS gzip + 12 KB CSS gzip + 18 KB font per weight.*

**Constraints**:

- **The API contract is frozen.** Phase 0 is the only window for backend changes;
  after that the web is a consumer only. Any later change needs a
  `specs/00X/contracts/api.md` delta like `002` did.
- **Persian-only product.** No English UI, no i18n framework, no locale switch. This
  is a contract requirement (`contracts/api.md` §Conventions), not a shortcut.
- **Jalali in, Jalali out.** Wire format stays ISO-8601 Gregorian (research R6);
  users never see or type a Gregorian date.
- **Money is integer Toman, serialized as a JSON string** to avoid JS float
  precision loss. Never `parseInt` → `number` → arithmetic on a value above
  `Number.MAX_SAFE_INTEGER` without a guard; the shared money type is
  `type Toman = string` with BigInt helpers in `format/money.ts`.
- **No new backend business logic.** Zero domain-model changes.
- **Windows dev machine.** PowerShell, no `make` guarantee in PATH, `flutter pub get`
  is broken locally (documented in `002/tasks.md`). Node/npm work.

**Scale/Scope**: 1 new top-level directory (`web/`), ~60–70 routes, ~520 Persian
strings, 0 migrations, ~6 small backend files touched (Phase 0 only).

**Storage**: only the display user and the active-building preference live in
`localStorage`; the refresh token is in an httpOnly cookie, never JavaScript-reachable.
All domain data lives in PostgreSQL 16 behind the existing API. No IndexedDB, no offline
queue (out of scope — see Out of Scope).

**Testing**: `vitest` for the pure library layer (money, Jalali, digits, validation,
error-envelope mapping) and component tests; `playwright` E2E against a **real backend
+ real PostgreSQL** via `docker compose` — deliberately mirroring the backend's own
`integration_harness_test.go` culture ("real data, not mocks"). No MSW/mock server:
the whole point is contract fidelity.

**Target Platform**: Evergreen browsers (Chrome/Edge/Firefox/Safari ≥ 2 versions
back). Desktop-first, usable at 1024 px; mobile-web degrades to the resident portal
only (managers use the APK). `dir="rtl"`, `lang="fa"`.

**Project Type**: SPA + existing REST API — third client in a modular-monolith repo
(`backend/internal/*`, `mobile/lib/features/*`, new `web/src/features/*`).

**Performance Goals**: initial JS ≤ **250 KB gzip** (route-level code splitting, no
big table/editor libs in the entry chunk); LCP < 2.5 s on Fast-3G; table sort/filter
interaction < 100 ms at 500 rows (client-side paging over server pages of 100);
no layout shift on async lists (fixed-height skeletons).

**Constraints**:

- **The API contract is frozen.** Phase 0 is the only window for backend changes;
  after that the web is a consumer only. Any later change needs a
  `specs/00X/contracts/api.md` delta like `002` did.
- **Persian-only product.** No English UI, no i18n framework, no locale switch. This
  is a contract requirement (`contracts/api.md` §Conventions), not a shortcut.
- **Jalali in, Jalali out.** Wire format stays ISO-8601 Gregorian (research R6);
  users never see or type a Gregorian date.
- **Money is integer Toman, serialized as a JSON string** to avoid JS float
  precision loss. Never `parseInt` → `number` → arithmetic on a value above
  `Number.MAX_SAFE_INTEGER` without a guard; the shared money type is
  `type Toman = string` with parse helpers that throw on unsafe values.
- **No new backend business logic.** Zero domain-model changes.
- **Windows dev machine.** PowerShell, no `make` guarantee in PATH, `flutter pub get`
  is broken locally (documented in `002/tasks.md`). Node/npm assumed working.

**Scale/Scope**: 1 new top-level directory (`web/`), ~60–70 routes, ~520 Persian
strings, 0 migrations, ~4 small backend files touched (Phase 0 only).

---

## Phase 0 — Review Findings (read this first)

These came out of reading the API surface, the DTO handlers, and the Flutter
repositories. **Four are pre-existing defects** that affect the mobile client today.
They gate the web work because the web consumes every one of them. All of F1, F3, F4 and
F5 are now **fixed and tested**; F2 needs the Phase 7 client-side work.

### F1 — Resident payment history is empty on Android (**live bug**) — FIXED

`backend/internal/payment/handlers.go` (`myPayments`) responded:

```json
{ "payments": [ ... ], "total": 42 }
```

but `mobile/lib/features/payments/payment_repository.dart` read:

```dart
final items = (res.data?['items'] as List? ?? const [])  // ← key did not exist
```

The `?? const []` swallowed it silently, so `/home/payments` rendered an empty list for
every resident. `buildingPayments` read `payments` correctly — that is why the manager
ledger worked and the resident history did not.

**Shipped**: both endpoints now emit the standard `{items, page, page_size, total}`
envelope (F4), plus `payments` retained as a documented deprecated alias for one release
cycle so already-installed APKs keep working. The Dart side reads `items` via a shared
`_readPage` helper. Regression test:
`backend/internal/payment/browserreturn_integration_test.go::TestMePaymentsUsesStandardEnvelope`.

### F2 — Attachments cannot be loaded by a browser

`storage.Register` mounts `GET /files/:id` behind `authMW` — Bearer header only
(`backend/internal/platform/storage/handler.go:37`). The upload response returns
`"url": "/files/" + id`, which is unusable as `<img src>`, `<a href download>`, a PDF
preview, or a print target: the browser will not attach an `Authorization` header.

Receipt images are the primary evidence in an expense audit trail, so this is not
cosmetic. Resolution (client-side, no backend change): a `useFileUrl` hook that `fetch`es
with the Bearer header and returns an object-URL, with an LRU cache and
`URL.revokeObjectURL` on eviction. Recorded as a P1 follow-up for a short-lived signed
download token once volume makes blob-URL caching the wrong shape. *Phase 7 — pending.*

**Guard in place**: an ESLint rule already rejects literal `/files/...` URLs so the trap
cannot be re-introduced.

### F3 — The payment gateway has no browser landing page — FIXED

`gatewayCallback` returned raw JSON with `"deeplink": "hamsa://payments/<id>"`. When
Zarinpal redirected the browser to
`https://app.hamsa-home.ir/api/v1/payments/callback?Authority=…`, a web user saw a JSON
blob with no way back.

**Shipped**: `return_path` rides as a query parameter on the gateway callback URL, so
nothing is persisted on the payment row — no migration, and nothing to clean up if the
user abandons the redirect. The service re-validates the path as same-origin at callback
time, so a tampered `return_path` cannot become an open redirect. Mobile omits
`return_path` and its JSON + `hamsa://` response is byte-identical (additive, non-breaking).
Configured via `app.web_return_url`; empty disables the browser hand-off entirely.
Gated by `returnurl_test.go` (9 cases incl. `//evil.example`, `/\evil.example`,
`https://evil.example`) plus three integration tests covering the happy path, the
tampered path, and the Android-only posture.

### F4 — Three different pagination envelope shapes on one API — FIXED

| Endpoint | Envelope | Page-size param |
|---|---|---|
| `units`, `invoices` (mgr/resident), `announcements`, `notifications`, `expenses`, `maintenance-requests` | `{items, page, page_size, total}` | `page_size` |
| `GET /buildings/{id}/payments`, `GET /me/payments` | `{payments, total}` — **no `page` echoed** | **`size`** |
| `GET /buildings/{id}/periods` | `{items}` — no pagination at all | — |

(`contracts/api.md` documents exactly one envelope, so the code had drifted from the
contract.)

**Shipped**: both payment endpoints now emit the standard envelope, with `page_size`
honoured and `size` kept as a deprecated alias for one release cycle so an un-updated APK
keeps working. `/periods` stays unpaginated by design: a building has tens of periods,
not thousands, and the client sorts locally.

### F5 — CORS is `Access-Control-Allow-Origin: *`

`httpx.CORS()` (`platform/httpx/middleware.go:99`) ships a code comment that reads
*"tighten allowed origins before exposing beyond localhost"* — written when the only
client was Flutter. Consequences:

- **Production**: irrelevant by design, because the SPA is served from the same
  origin that nginx proxies `/api/` from (Decision D2). No CORS preflight at all.
- **Development**: required. Vite on `:5173` calling `:8080` is cross-origin, so the
  dev config must allow `http://localhost:5173` and `http://127.0.0.1:5173`.
  Convert the hardcoded `*` to a config-driven `cors.allowed_origins` list, defaulting
  to dev origins and empty in prod (where nothing is cross-origin).

### F6 — research R8's "session cookies rejected" is now stale — RESOLVED

`specs/001/.../research.md` R8 rejected session cookies with the reason *"mobile app,
not browser"*. That premise no longer holds. **Implemented**, not deferred:

- Access token in memory only; strict CSP; no `innerHTML` on any server string; no
  third-party script tags.
- The refresh token is issued as an `httpOnly` + `SameSite=Lax` cookie (plus `Secure`
  in every non-dev environment), scoped to `/api/v1/auth`. The Android client still
  receives it in the JSON body; a request declaring `X-Hamsa-Client: web` gets it only
  in the cookie, so the body never hands page JavaScript the long-lived token.
- Cookie-authenticated state changes (`/auth/refresh`, `/auth/logout`) additionally
  require the double-submit `X-CSRF-Token`; `SameSite=Lax` also blocks the cross-site
  POST a CSRF attack needs. The readable CSRF cookie is scoped to `/`.

No secret is reachable from JavaScript, so the XSS token-theft vector is removed.

---

## Open Decisions for the Reviewer

Three judgement calls worth confirming before Phase 1 starts.

**D1 — Framework: SvelteKit SPA vs. `flutter build web`.** — **RESOLVED: SvelteKit 3**
Per the requester's instruction, the client is **SvelteKit 3 + Svelte 5** rather than
the React originally proposed. The *reasoning* for a dedicated web client is unchanged
and worth recording:

The API is a frozen, well-documented contract (`001/contracts/api.md` + the `002`
delta), so a new client integrates against *documentation*, not internal code — the
Flutter repository layer is ~22 short files with no logic worth porting. The Flutter
screens are mobile-idiomatic; making them desktop-capable (tables, split panes, keyboard
forms) is a rewrite of the UI layer either way, so the reuse argument mostly evaporates.
What survives the rewrite is **string parity and screen inventory**, and both are captured
as artifacts (`contracts/api.md`, `web/src/i18n/fa.ts`, and the screen-parity table in
`quickstart.md`).

Rejected: Flutter Web's ~2.5–4 MB initial payload (skwasm/canvaskit runtime + app code)
on the exact networks where the APK exists as a download; weaker table/data-grid
ergonomics; and a second toolchain for a feature that will outgrow the mobile shell.

SvelteKit 3 was chosen over other SPA frameworks for the same reasons React was: file-based
routing with per-route code splitting, a small runtime, and first-class SPA mode via
`adapter-static`. The runes model also removes the wrapper problem that makes
component libraries heavy.

**D2 — Where the SPA is served.** Recommendation: **`app.hamsa-home.ir`**, the host
that already reverse-proxies the entire backend. The SPA and `/api/v1` become
same-origin in production: no CORS preflight, no third-party cookie concerns, and the
existing nginx block needs only `root` + `location / { try_files … /index.html }` added.
`hamsa-home.ir` keeps the static landing page and the APK download untouched.
Rejected: `hamsa-home.ir/app/` (shares the marketing origin, mixes static and app
caching rules); a third subdomain (needs a new cert + DNS + `DEPLOY.md` churn for no
benefit).

**D3 — Table stack.** — **RESOLVED: headless, TanStack Table (Phase 4)**
The unit list, invoice list, ledger and expense list all need server-side pagination
*and* client-side sort within a page, and one of them (cost-item preview) needs row
expansion. The table stays headless, styled with Tailwind, because that keeps RTL
control and bundle size in our hands. Rejected: a heavyweight all-in-one grid
component (poor RTL control, large bundle, fights Tailwind logical properties).

---

## Research Items (Phase 0 of the speckit workflow)

These are the questions Phase 0 must answer, each recorded in
`specs/003-web-frontend/research.md` with a decision, rationale, and rejected
alternatives — matching the R-numbering convention of `001`/`002`.

| # | Question | Resolution |
|---|---|---|
| R1 | Does `Intl.NumberFormat('fa-IR')` emit U+066C (٬) or ASCII `,` as the group separator across the ICU versions in our target browsers? | **Not used, deliberately.** Grouping is computed from the digit table (`groupDigits`) so output is byte-identical to the mobile client and independent of the host ICU. `money.test.ts` asserts the exact string, including that U+066C is present and ASCII `,` is not. |
| R2 | Is `Intl.DateTimeFormat('fa-IR-u-ca-persian')` stable across evergreen browsers? | **Used** for `formatJalaliDate`/`formatJalaliDateTime` only — where the platform's own Persian locale is exactly what we want to match. Conversion and input are hand-rolled, so a formatting difference can never corrupt a stored date. |
| R3 | Jalali input: hand-rolled picker or a maintained library? | **Hand-rolled**, and the gate proved its worth: the first implementation of the day-number arithmetic **failed** the exhaustive round-trip suite immediately (a leap boundary), which is precisely the class of bug that would otherwise surface as a manager picking a wrong due date. Replaced with the standard integer-arithmetic algorithm; the suite now passes all ~73k cases plus a day-by-day monotonicity walk. |
| R4 | Refresh-token single-flight in the browser: how do concurrent 401s behave? | Mirror the mobile `QueuedInterceptor` (queued, not parallel). Every in-flight request awaits one shared refresh promise, then replays with the new token. |
| R5 | Query invalidation map — which mutations dirty which lists? | Derived per endpoint in `src/lib/query/keys.ts` (e.g. `issue` invalidates invoices + balances + dashboard + notifications). Written as a table in `data-model.md`, reviewed before Phase 5. |
| R6 | Server-side or client-side sorting/paging for lists > 200 rows? | Server-side paging (already supported); client-side sort *within* the current page only. Rejected: fetching all rows — the API's `page_size` cap is 100 and units/ledger can grow. |
| R7 | Print/PDF invoice — how much fidelity? | A dedicated `/invoice/:id/print` route with `@media print` (hide chrome, show cost-item table, QR-free, page-break-safe). Browser "Save as PDF" gives the PDF. No PDF library. |
| R8 | Multi-building manager UX: the mobile app has no building switcher (route is `/manager` with no `buildingId` on the dashboard). | Web **adds** a global building switcher in the top bar, persisted to `localStorage`, with the active `buildingId` in the URL (`/m/:buildingId/...`) so every view is linkable and back-button-correct. Purely additive; the API already scopes by `building_id`. |
| R9 | Screen-parity contract with the Flutter app. | Every one of the 42 `*_screen.dart` files maps to a web route. The mapping table lives in `specs/003-web-frontend/quickstart.md` so parity gaps are auditable rather than assumed. |
| R10 | Do we need a web-specific error boundary / offline state? | Yes: a route-level `ErrorBoundary` rendering the Persian `error.message` from the envelope, and an `offlineManager` that pauses mutations when `navigator.onLine` is false (Iranan networks drop). |

---

## Constitution Check

*Gate: must pass before Phase 0 research. Re-check after Phase 1.*

`.specify/memory/constitution.md` is an **unfilled template** (placeholders only — no
ratified principles or gates). Therefore:

- **Pre-research gate**: PASS (no active constraints to violate).
- **Post-design re-check (after Phase 1)**: PASS — the design carries forward the
  invariants this repo has actually enforced so far, which become the web's
  acceptance criteria:
  - **Contract fidelity** — the web is a client of `001/contracts/api.md`; every
    Phase 0 delta is written into `003/contracts/api.md`, and Phase 1 includes a
    generated route/endpoint inventory diffed against the backend.
  - **Persian-only, Jalali-only, RTL-only** — no English string ships; `dir="rtl"`
    is set before first paint to avoid a flash; `grep` gates in CI forbid
    `ml-/mr-/pl-/pr-/left-/right-/text-left/text-right` in `src/`.
  - **Resident isolation is a server property** — the web never computes
    authorization. It renders whatever `/me/*` returns and relies on the existing
    server-side scope resolution; no client-side permission branching beyond
    hiding a nav item the server would reject anyway.
  - **Money never touches a float** — `Toman` is a branded `string`; arithmetic goes
    through `BigInt` helpers in `src/lib/format/money.ts`.

---

## Project Structure

### Documentation (this feature)

```text
specs/003-web-frontend/
├── plan.md              # This file
├── spec.md              # Phase 0 — user stories + functional requirements (web surfaces)
├── research.md          # Phase 0 — R1–R10 above, with decisions + rejected alternatives
├── data-model.md        # Phase 1 — TS domain types per endpoint, query-key table, invalidation map
├── quickstart.md        # Phase 1 — dev setup + the 42-screen Flutter→web route parity table
├── contracts/
│   └── api.md           # Phase 0 delta — F1/F3/F4/F5 changes only; everything else inherited
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 — the task breakdown below
```

### Source Code

```text
web/
├── vite.config.ts                 # adapter-static SPA mode + dev proxy /api,/files → :8080
│                                  # (kills dev CORS; SvelteKit 3 config lives here)
├── tsconfig.json                  # strict, noUncheckedIndexedAccess, #lib/#i18n paths
├── eslint.config.js               # + RTL / money / date / auth-gated-file guards
├── playwright.config.ts
├── scripts/build-fonts.mjs        # subsets mobile/assets/fonts/*.ttf → woff2 (370 KB → 53 KB)
├── static/fonts/                  # committed woff2 subsets: Vazirmatn-{400,500,700}
├── e2e/                           # seed.ts (real API, no DB dumps) · auth.setup.ts · *.e2e.ts
└── src/
    ├── app.html                   # <html lang="fa" dir="rtl">, Vazirmatn preload, no FOUC
    ├── i18n/
    │   ├── fa.ts                  # typed Persian dictionary, ~520 keys, mirroring app_fa.arb
    │   └── index.ts               # #i18n entry
    ├── lib/                       # framework-free; the only code with unit tests that matter
    │   ├── config.ts              # API base + active-building preference (R8)
    │   ├── api/
    │   │   ├── http.ts            # fetch wrapper: base URL, JSON, 401→refresh→replay
    │   │   ├── apiError.ts        # {error:{code,message,details[]}} → ApiError + per-field map
    │   │   ├── tokenStore.ts      # access in memory · refresh in httpOnly cookie
    │   │   ├── refreshQueue.ts    # single-flight refresh (R4)
    │   │   └── endpoints/         # auth · buildings · units · persons · occupancies ·
    │   │                          # periods · costItems · invoices · payments · expenses ·
    │   │                          # maintenance · announcements · notifications · dashboard · files
    │   ├── auth/auth.svelte.ts     # session state as runes (unknown→authenticated|guest)
    │   ├── format/
    │   │   ├── money.ts           # Toman branded string, BigInt math, U+066C grouping
    │   │   ├── jalali.ts          # jy/jm/jd ↔ Gregorian (R3), Intl-backed formatting
    │   │   ├── digits.ts          # Persian/Arabic-Indic ⇄ Latin, bidi isolates, strict check
    │   │   └── labels.ts          # enum → Persian label/tone tables
    │   ├── query/
    │   │   ├── queryClient.ts     # staleTime, retry-on-transient-only
    │   │   └── keys.ts            # query-key tree + the invalidation map (R5)
    │   ├── validation/            # phone / password / invite-code validators
    │   └── upload.ts              # POST /files multipart + object-URL cache (F2)
    ├── components/                # Phase 3: ui/ primitives + app composites
    │                              # (DataTable, MoneyText, StatusChip, JalaliDatePicker, …)
    └── routes/                    # file-based routing; SPA mode (ssr=false, prerender=false)
        ├── +layout.svelte         # QueryClientProvider + session-expiry wiring
        ├── +layout.ts             # ssr=false · prerender=false · trailingSlash='never'
        ├── +page.svelte           # startup: restore session → role's own shell
        ├── login/ register/ setup/
        └── m/ … r/ …              # manager console / resident portal (Phase 3–9)
```

### Supporting changes (non-`web/`)

```text
backend/internal/payment/handlers.go        # F4 envelope + F3 return_path/302 + pageSizeFromQuery
backend/internal/payment/service.go         # F3 return_url plumbing (callbackURLFor, BrowserReturnURL)
backend/internal/payment/returnurl_test.go  # NEW — open-redirect guards
backend/internal/payment/browserreturn_integration_test.go  # NEW — F1/F3/F4 over real HTTP
backend/internal/platform/config/config.go # F5 cors.allowed_origins + app.web_return_url
backend/internal/platform/httpx/middleware.go # F5 CORS(allowedOrigins)
backend/internal/platform/httpx/router.go    # RouterOptions (non-breaking signature)
backend/internal/platform/httpx/middleware_test.go  # NEW — CORS allowlist behaviour
backend/cmd/server/main.go                  # wire RouterOptions + web_return_url
backend/config.example.yaml                # new keys, documented
backend/migrations/                        # NONE — no schema change in this feature
deploy/nginx-hamsa.conf                    # D2: root + SPA fallback on app.hamsa-home.ir (Phase 9)
.github/workflows/build-web.yml            # new CI, mirrors build-apk.yml
Makefile                                   # web-* targets; lint/test now include web
DEPLOY.md                                  # web deploy + rollback section (Phase 9)
README.md                                  # component table gains a `web/` row (Phase 9)
AGENTS.md                                  # SPECKIT pointer → 003 plan
mobile/lib/features/payments/payment_repository.dart   # F1: read `items` via _readPage
specs/001-building-management-mvp/contracts/api.md    # pagination shapes documented truthfully
```

**Structure Decision**: No new backend packages and no new top-level app — `web/`
sits beside `mobile/` as a peer client, mirroring its `features/*` layout so
screen-parity review is a directory diff. The `lib/` layer is separated from
`features/` on purpose: everything worth unit-testing (money, Jalali, error mapping,
token refresh) is framework-free and has no React import, which is what makes the
exhaustive `001`-style contract tests cheap. Component primitives live in
`components/ui/`, domain composites in `components/`, so a feature never imports
another feature's internals.

---

## Delivery Phases

Nine phases. Phases 1–3 are foundations that every later phase depends on; Phases 4–9
are each independently demoable. Each phase ends at a runnable, committed state — no
half-integrated branch.

### Phase 0 — Contract deltas and pre-flight fixes (no UI)

Goal: the API is web-ready and the live mobile bug is dead.

- F1 fix: `myPayments` envelope → `{items, page, page_size, total}`; Flutter
  `myPayments` reads the standard envelope in the same commit. Regression test in
  `payment_integration_test.go`.
- F4 fix: `buildingLedger` → standard envelope + `page_size` (keep accepting `size`
  as a deprecated alias for one release so an un-updated APK keeps working).
- F3: optional allowlisted `return_url` on `POST /invoices/{id}/pay`; `302` from the
  callback when present; `POST` body and existing JSON response otherwise unchanged.
- F5: config-driven CORS allowlist; dev config allows `localhost:5173` /
  `127.0.0.1:5173`; prod config empty.
- Write `spec.md`, `research.md`, `contracts/api.md` delta, `checklists/requirements.md`.
- **Exit**: `go test ./...` green; `flutter analyze`/`test` green; `contracts/api.md`
  states one pagination envelope; curl shows both payment endpoints returning it.

### Phase 1 — Application skeleton, toolchain, CI

- Vite 7 + React 19 + TS strict + Tailwind 4 + ESLint + Vitest + Playwright wired up.
- `index.html` with `dir="rtl"` **in the static HTML** (no direction flash), Vazirmatn
  `woff2` self-hosted and preloaded, self-hosted only (no third-party font CDN).
- Tailwind design tokens ported from `AppTheme`: seed `#00695C`, cream surface
  `#FAF9F6`, radii 16/12/20, spacing scale, Vazirmatn type scale (body line-height 1.7,
  titles 1.3 — Persian glyphs sit taller).
- `Makefile` targets + `build-web.yml` CI (typecheck → lint → unit test → build →
  upload artifact; publish on `v*` tags).
- ESLint custom rules: **ban directional Tailwind utilities** (`ml-*`, `pr-*`,
  `text-left`, …) in favour of logical properties; ban `Date.parse` on wire dates
  without the ISO helper; ban raw `<img src="/files/...">` (must go through
  `useFileUrl`, F2).
- **Exit**: `npm run build` emits < 250 KB gzip entry chunk; CI green; dev server
  proxies `/api/v1` and `/healthz` responds through it.

### Phase 2 — `lib/` layer: money, Jalali, digits, API error mapping

The highest-risk, highest-leverage phase. Nothing else is safe until the Jalali
conversion is proven.

- `money.ts`: `Toman` branded string; `formatToman` (research R1 outcome, asserted to
  an exact expected string), `parseToman`, `groupToman` with U+066C, bidi isolates
  (U+2066…U+2069) around negative amounts — the exact Flutter bug documented at
  `mobile/lib/shared/formatters/money_text.dart:55` must be covered by a test.
- `jalali.ts`: conversion + Persian month names + `formatJalaliDate` /
  `formatJalaliLongDate` / `formatJalaliDateTime` / `jalaliMonthName`.
- **Exhaustive round-trip suite: every day 1300-01-01 → 1500-12-29** (~73,000 cases)
  asserting `jalaliOf(gregorianOf(j)) === j` and monotonicity of the day sequence.
  This is the gate that makes hand-rolled date math (R3) acceptable.
- `apiError.ts`: envelope → `ApiError {code, message, details[]}`, plus
  `fieldErrors()` mapping `details[].field` → RHF form errors.
- `digits.ts`: Persian/Arabic-Indic ⇄ Latin, tolerant of U+066C and `,` and bidi
  controls on input (mirrors `parseToman`/`latinDigitsOnly`).
- **Exit**: all four suites green; `npm run test` is fast and deterministic.

### Phase 3 — Auth, session, routing skeleton, design system

- `tokenStore` + `refreshQueue` (R4): single-flight refresh mirroring the mobile
  `QueuedInterceptor`; concurrent 401s queue rather than stampede; refresh failure
  clears the session and redirects to `/login`; the original request replays with the
  new access token.
- `/login`, `/register` (invite-code redemption), `/setup` (first-run superadmin),
  `/m/invite` (manager/superadmin role toggle — the `002` feature), change-password.
- Route guards mirroring `app_router.dart`'s `_redirect`: unauthenticated → `/login`;
  `manager|supersadmin` → `/m/:buildingId`; `resident` → `/r`; cross-section access
  bounced to the caller's own home (client-side convenience only — the server is the
  authority).
- `ManagerLayout` (sidebar, topbar, **building switcher** per R8) and
  `ResidentLayout`; route-level `ErrorBoundary` (R10) and offline manager (R10).
- `components/ui/*` — the ~15 headless primitives, RTL-correct.
- **Exit**: a manager can log in, switch buildings, and see an empty-but-correct shell
  with the sidebar reflecting real role and building scope; a 401 storm resolves in
  exactly one refresh call (test asserts the call count).

### Phase 4 — Registry: buildings, units, people, occupancy

**Status — implemented.** Built with Svelte 5 runes + `@tanstack/svelte-query`
rather than the `react-table`/RHF named below, per the D1 resolution (SvelteKit).
Covers: building edit + managers screen (add/list/remove, last-manager and
self-removal guards surfaced verbatim), unit list with the server-side
`?q=&block=&floor=&status=` filter bar + paging, unit create/edit/archive, unit
history, occupancy history + add/end, occupant-count history + record, and person
list/create/edit/archive. A shared `JalaliDateInput` parses tolerant Solar-Hijri
input to ISO (unit-tested in `dateInput.test.ts`).

The first feature phase; establishes the list/form/filter pattern reused everywhere.

- Buildings: list, create/edit, **managers screen** (list/add/remove with the
  last-manager and self-removal guard errors surfaced verbatim in Persian).
- Units: **server-side filter bar** (`?q=&block=&floor=&status=`) over a
  `@tanstack/react-table` — this is the direct payoff of the web build and the
  reference implementation for every later list.
- Unit form, unit history (`GET /units/:id/history`), occupancy history, add-occupancy
  form, occupant-count records.
- Persons: list, create/edit, archive.
- **Exit**: a manager can do P0-01 and P0-02 entirely from the browser, with the
  `409` duplicate-unit-number error landing on the right form field.

### Phase 5 — Billing: periods, cost items, calculation, **preview grid**

**Status — implemented.** Built on the Phase 4 patterns (Svelte 5 runes +
`@tanstack/svelte-query`). Covers period list/create/edit (draft-only edit),
the cost-item form for all six methods including the `combined` weight editor
(live sum check) and a searchable `specific_units` multi-select, `calculate` /
`reopen` / `issue` (confirm names the invoice count) / `close` gated by status,
and the preview grid: per-unit invoice rows, per-row drill-down into each cost
item's exact and rounded share, and the BR-08/BR-09 reconciliation footer with
per-item and overall residual. Added `lib/billing/reconcile.ts` +
`weights.ts` (pure, unit-tested) and fixed a latent label bug: the wire method
token is `per_occupant`, not `per_person`.

The crown jewel and the highest-value divergence from mobile.

- Period list + form (start/end/due date via `JalaliDatePicker`, late-fee config).
- Cost-item form: method select (`equal` / `per_occupant` / `per_area` / `fixed` /
  `specific_units` / `combined`), the `combined` **weight editor** (percentages
  summing to 100, with a live sum check), `include_vacant`, and the specific-units
  **searchable multi-select** — block/floor/number filters over the unit list, far
  better than a mobile picker.
- `POST /periods/:id/calculate`, then the **preview grid**: per-unit rows with the
  resulting invoice (prior debt, late fee, credit, final amount); expand a row to see
  each cost item's method, total, exact share and rounded share; a sticky
  **reconciliation footer** showing Σ rounded shares vs. cost-item total per row group
  and overall — the client-side rendering of BR-08/BR-09 that the contract explicitly
  requires the client to display.
- `reopen` → `issue` (with a confirm dialog naming the invoice count) → `close`, each
  gated by the period's state and by the API's `409` guard.
- **Exit**: the §25 sample scenario reproduces in the browser — unit 1 shows
  **۲٬۶۰۰٬۰۰۰ تومان**, and reconciliation shows zero residual. This is an E2E test.

### Phase 6 — Money movement: invoices, payments, balances, ledger

**Status — implemented.** Manager invoice list (`?period_id=&unit_id=&status=`,
server-side paging) and detail (amounts, item breakdown, adjustments,
record-manual-payment with partial support, cancel-with-reason, adjustment
notes); a shared print view at `/invoice/:id/print`; the payment ledger
(`?unit_id=&method=&from=&to=`) with a `/m/payments/new` invoice picker for
recording; the unit balance view with all seven §9 components, the formula
rendered live and a reconciliation check; and the resident portal
(`/r/invoices`, `/r/invoices/:id` pay-online via `return_path`, `/r/payments`,
`/payment/result`). Pure logic lives in `lib/payments/balance.ts` with 16 unit
tests. A pre-existing backend defect was fixed en route: `Payment.UnitNumber`
was tagged `gorm:"-"`, so the ledger's `unit_number` subquery was dropped and
the column always rendered empty.

- Manager invoice list (`?period_id=&unit_id=&status=`), invoice detail with item
  breakdown, cancel-with-reason, debit/credit adjustments.
- **`/invoice/:id/print`** — print-optimised, page-break-safe invoice (R7).
- Payment ledger (`GET /buildings/:id/payments`) with unit/method/date filters and
  server-side paging; "record payment" opened from a ledger row **or** from an
  invoice (the `002`-era flow mobile discovered the hard way).
- Unit balance view — all seven components of spec §9 with a live
  `مانده = بدهی قبلی + قبض جاری + جریمه − پرداخت − بستانکاری` breakdown.
- Resident pay flow: `POST /invoices/:id/pay` with `return_url` → gateway → the F3
  `302` → `/payment/result` which polls/reads the verified payment and renders a
  Persian success/failure panel plus a print receipt.
- **Exit**: a full issue → partial payment → balance-recompute → pay-online loop works
  in a browser, E2E-covered against the mock gateway.

### Phase 7 — Expenses and financial report

- Expense list with category/date/approval filters; create/edit with **receipt upload
  and preview** (`POST /files` then `useFileUrl`, F2); approve/reject workflow.
- Monthly financial report (`?month=YYYY-MM`, Jalali month selector) — income,
  expense, net, total resident debt, total payments, total expenses.
- **Exit**: P0-05 complete in the browser; receipt images render from blob URLs.

### Phase 8 — Maintenance, announcements, notifications

- Manager: maintenance list filtered by status/priority/category, detail with the
  status machine (`new → under_review → in_progress → resolved → closed`), assignee,
  recorded cost, note; announcements publish/edit/delete with audience targeting
  (all / block / floor / specific unit) and attachment.
- Resident: submit a request with photo upload, track status with a timeline; read
  targeted announcements and mark them read.
- Notification center: paginated, unread-only filter, mark-one/mark-all, with an
  unread badge in the top bar.
- **Exit**: P0-06, P0-07 and the notification surface complete.

### Phase 9 — Dashboards, print polish, launch hardening

- Manager dashboard: the six cards, quick actions, and the alerts list (debtor units,
  past-due invoices, open requests, pending expenses) with links into the filtered
  lists — **every alert card deep-links to a pre-filtered table**, so a manager goes
  from "۷ واحد بدهکار" to the seven-row list in one click.
- Resident home (`GET /me/home`): payable amount, latest invoice, open requests,
  latest announcements, unread count.
- Responsive pass (1024 px floor; resident portal usable at 375 px), accessibility
  pass (keyboard-only run-through of billing and payment), Lighthouse budget gate.
- E2E suite green against a real backend; `DEPLOY.md` web section; README `web/` row;
  `AGENTS.md` pointer → this plan; nginx SPA-fallback + gzip + immutable-hashed-asset
  caching; rollback procedure.
- **Exit**: deployable to `app.hamsa-home.ir`, documented, reversible.

---

## Testing & Acceptance Strategy

Mirroring the repo's existing posture — the backend tests against **real PostgreSQL**
rather than mocks, and `002/tasks.md` writes failing tests before implementation. The
web follows suit.

**Layer 1 — pure logic (Vitest, no DOM).** Highest value per line of code:
the 73k-case Jalali round-trip; money formatting/parsing including negative-balance
bidi and Persian-digit input; error-envelope → field-error mapping.

**Layer 2 — components (Vitest + Testing Library).** `StatusChip` enum→Persian maps,
`JalaliDatePicker` emission as ISO, cost-item weight editor's 100% validation, the
reconciliation footer arithmetic, the request-header/breadcrumb.

**Layer 3 — E2E (Playwright, real backend + real PostgreSQL).** One spec per user
story, seeded through the real API (`e2e/seed.ts` creates a building, units, people, a
period and an issued invoice via HTTP — no DB dumps, no fixture JSON), with
`storageState` reuse so the suite stays fast.

**Mandatory gates**, each a checklist item rather than a judgement call:

| Gate | Assertion |
|---|---|
| Contract parity | Generated endpoint inventory diffed against `001`/`003` contracts — zero undeclared calls |
| Persian-only | No Latin-letter UI string reaches render; CI greps for hardcoded literals outside `i18n/fa.ts` |
| RTL purity | CI fails on `ml-/mr-/pl-/pr-/left-/right-/text-left/text-right` in `src/` |
| Money safety | No `parseInt`/`Number` on a `Toman` value; ESLint rule enforced |
| Jalali correctness | Exhaustive 1300–1500 SH round-trip suite |
| Sample scenario | Billing E2E reproduces spec §25: unit 1 = ۲٬۶۰۰٬۰۰۰, reconciliation residual 0 |
| Resident isolation | A resident E2E asserts a foreign `building_id`/`invoice_id` in the URL yields `403` and no data render |
| Immutability | Issued-invoice amounts unchanged after editing unit area/occupant count (mirrors BR-03/BR-05) |
| Single refresh | A 401 storm triggers exactly one `POST /auth/refresh` |
| Bundle budget | Entry chunk ≤ 250 KB gzip, enforced in CI |

---

## Risks and Mitigations

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | Hand-rolled Jalali conversion has a leap-year or boundary bug (research R3) | Medium | Exhaustive 1300–1500 SH round-trip suite as a Phase 2 gate; fall back to a library if it fails |
| 2 | ICU group separator differs across browsers (research R1), breaking money display | Medium | Assert the exact string in a unit test in Phase 2; `formatToParts` + manual separator swap as the fallback |
| 3 | XSS token theft (F6/R8) | Low | Access token in memory only; refresh token in an httpOnly + SameSite=Lax cookie; double-submit CSRF on cookie-auth writes; strict CSP; no third-party scripts; no `innerHTML` on server strings |
| 4 | Feature parity drifts from the mobile app as both evolve | Medium | The 42-screen parity table in `quickstart.md` (R9) is re-checked each phase; `features/*` directory parity makes drift visible in review |
| 5 | Backend changes in Phase 0 break the shipped APK (which users update slowly) | Medium | Additive-only deltas; `size` kept as a deprecated alias for one release; Flutter fixes ship in the same commits |
| 6 | Bundle creeps past budget as tables/editors land | Medium | Route-level `lazy()`, TanStack Table is headless, no PDF/grid library; CI budget gate |
| 7 | Scope creep into the "Out of Scope" list below | Medium | The plan is explicitly bounded by `mvp-p0.md` §27; additions require a new `00X` spec |
| 8 | Persian print output breaks (RTL pagination is notoriously fiddly) | Low | Isolated in Phase 9 behind a dedicated print route; verified by an actual printed-to-PDF E2E check |

---

## Out of Scope

Carried forward from `mvp-p0.md` §27, plus web-specific exclusions:

Advanced parking management · common-area booking · contractor contracts · full
double-entry accounting · inventory · access control · license-plate recognition ·
IoT · AI · multi-tier board management · CRM · advanced financial reporting ·
**offline-first / PWA service worker** · **native push notifications in the browser**
(FCM Web Push needs VAPID keys and a service worker — the in-app notification center
covers P0) · **multi-language UI** (explicitly Persian-only per contract) ·
**native mobile app** (Flutter stays the native client) · **white-label theming per
building**.

---

## Complexity Tracking

> Fill only if the Constitution Check has violations that must be justified.

No constitution violations — the table is intentionally empty.

One deliberate deviation from the existing mobile architecture is recorded here rather
than hidden: the Flutter app is Riverpod + `go_router` with feature-scoped providers.
The web uses TanStack Query + React Router, so **screen-level logic is not shared
across the two clients**. The justification is that `mobile/lib`'s reusable logic is
almost entirely *presentation* (widget composition, theming, format strings); the
genuinely reusable asset is the **contract plus the string table**, both of which are
captured as artifacts (`contracts/api.md`, `i18n/fa.ts` + the parity table) rather than
as code. Attempting to share code across a Dart and a TypeScript runtime would mean a
build step, a serialization boundary, and two bug surfaces for the same rules — worse
than a clean re-implementation behind a frozen contract.

---

## Immediate Next Actions

1. **Review the four Phase 0 defects** (F1–F4) — F1 is a live bug and can ship as its
   own hotfix commit ahead of any web work.
2. **Confirm or override D1–D3** above. D1 (React vs. Flutter Web) is the only one
   that would materially change the plan.
3. **Create `spec.md`** from the phase list above: one user story per phase group, with
   the acceptance gates as functional requirements.
4. ~~**Decide on F6/P1** (httpOnly refresh cookie) — whether it is in scope for this
   feature or a follow-up.~~ **RESOLVED — implemented**: refresh token via httpOnly
   cookie + double-submit CSRF (see F6 above).
5. Point `AGENTS.md`'s SPECKIT block at `specs/003-web-frontend/plan.md` once created.
