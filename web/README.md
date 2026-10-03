# web — Hamsa web client

Browser client for the Hamsa building-management platform: a manager console
(dense tables, charge-calculation review, payment ledger) and a resident portal
(invoices, online payment, maintenance requests, announcements).

Persian-only, RTL, Jalali calendar. Plan and rationale:
[`specs/003-web-frontend/plan.md`](../specs/003-web-frontend/plan.md).

## Stack

| Concern      | Choice                                                        |
| ------------ | ------------------------------------------------------------- |
| Framework    | SvelteKit 3 (SPA mode) + Svelte 5 runes                       |
| Build        | Vite 8, `@sveltejs/adapter-static`                            |
| Server state | TanStack Query 5 (`@tanstack/svelte-query`)                   |
| Validation   | Zod 4                                                         |
| Styling      | Tailwind CSS 4, CSS-first `@theme` tokens, logical properties |
| Tests        | Vitest (unit + browser component), Playwright (E2E)           |
| Language     | TypeScript 6, strict, `noUncheckedIndexedAccess`              |

Source aliases are Node subpath imports (SvelteKit 3 removed `$lib`):
`#lib` → `src/lib`, `#i18n` → `src/i18n`.

## Prerequisites

Node ≥ 22.17, plus a running backend (see the root `README.md`) for anything
that talks to the API.

```sh
make up        # PostgreSQL on :5432 (from the repo root)
```

## Development

```sh
npm install
npm run dev    # http://localhost:5173
```

The dev and preview servers proxy `/api` and `/files` to `http://localhost:8080`,
so every request is same-origin and the browser needs no CORS grant — matching
production, where the app and the API share one nginx vhost. Point the proxy
elsewhere with `HAMSA_API_ORIGIN`.

The first account is created through `POST /auth/setup`, which the app exposes at
`/setup` (the superadmin bootstrap from `002-multi-manager-support`).

## Scripts

| Command                        | Purpose                                                   |
| ------------------------------ | --------------------------------------------------------- |
| `npm run dev`                  | Dev server with API proxy                                 |
| `npm run check`                | `svelte-check` type + a11y diagnostics                    |
| `npm run lint`                 | Prettier check + ESLint (incl. the RTL/money/date guards) |
| `npm run format`               | Prettier write                                            |
| `npm run test:unit`            | Vitest — unit (node) and component (browser) projects     |
| `npm run test:e2e`             | Playwright against a built app                            |
| `npm run build`                | Static output to `build/`                                 |
| `npm run preview`              | Serve the build with SPA fallback (prod-equivalent)       |
| `node scripts/build-fonts.mjs` | Re-subset the Vazirmatn woff2 faces                       |
| `node scripts/serve-spa.mjs`   | Static server with the production fallback semantics      |

## Layout

```text
src/
├── app.html                 # dir="rtl" lang="fa" + font preload
├── i18n/fa.ts               # Persian strings (mirrors mobile's app_fa.arb)
├── lib/
│   ├── api/                 # http, tokenStore, refreshQueue, apiError, endpoints/
│   ├── auth/                # session state (runes)
│   ├── format/              # money, jalali, digits, enum label tables
│   ├── query/               # query keys + the invalidation map
│   ├── validation/          # phone, password, invite code
│   └── config.ts            # API base + preferences
└── routes/                  # file-based routes; SPA mode
```

## Deep links and hosting

`adapter({ fallback: 'index.html' })` writes the SPA shell to `build/index.html`,
but a static host answers 404 for any path it does not recognise. Without a
rewrite, refreshing a bookmarked route like `/m/buildings` would break — and every
manager-console navigation produces such a URL.

So the app is served **same-origin with the API** and relies on a single
fallback rule:

- **production** — nginx, in `deploy/nginx-hamsa.conf`:
  `try_files $uri $uri/ /index.html;`
- **preview / E2E** — `scripts/serve-spa.mjs`, which mirrors that rule so CI
  exercises the deployed behaviour rather than a dev-server shortcut.

`scripts/serve-spa.mjs` also applies the cache headers production needs:
`_app/immutable/**` is content-hashed and served `immutable`, the HTML shell is
`no-cache`, and everything else is short-lived.

## Conventions the tooling enforces

These are product invariants that are easy to break by accident, so each is
pinned by an ESLint rule in `eslint.config.js`:

- **RTL purity** — no `ml-/mr-/pl-/pr-/left-/right-` or `text-left/right`
  utilities. Use the logical equivalents (`ms-`, `pe-`, `start-`, `text-start`).
- **Money never touches a float** — no `parseInt`/`parseFloat`. Amounts are
  `Toman` (a branded string) and go through `#lib/format/money`, which uses
  BigInt. The API sends amounts as strings precisely to avoid precision loss.
- **Jalali in, Jalali out** — no `Date.parse` on wire dates. Use
  `fromIsoDate`/`toIsoDate`; the wire is Gregorian ISO and the UI is Jalali.
- **Auth-gated files** — no literal `/files/...` URLs. `GET /files/{id}` needs a
  bearer token, so the browser must fetch it and render an object URL.

## Testing

Unit tests run in a `node` project and cover the pure layer, which is where the
real risk lives:

- `jalali.test.ts` round-trips **every day of 1300–1500 SH** (~73k cases) and
  walks a day-by-day monotonicity check. This is what makes hand-written Jalali
  arithmetic acceptable.
- `money.test.ts` asserts the _exact_ formatted output, including the U+066C
  separator and the bidi isolate around negative amounts.
- `refreshQueue.test.ts` pins single-flight token refresh — concurrent 401s must
  trigger exactly one exchange, or token rotation logs the user out mid-session.
- `apiError.test.ts` maps the `{error:{code,message,details}}` envelope,
  including field-level errors for forms.

Component tests run in a real browser (`@vitest/browser-playwright`). E2E runs
against a built app with a real backend and a real PostgreSQL — no mocked API.
