# Hamsa (همسا)

Building-management platform for Iranian residential buildings: shared-expense
billing, payments/balances, maintenance requests, announcements — Persian-only,
RTL, Jalali-calendar UI.

| Component | Stack | Path |
|---|---|---|
| Backend API | Go 1.25, Gin, GORM, PostgreSQL 16 | `backend/` |
| Mobile app (manager + resident) | Flutter (Riverpod, go_router, Material 3) | `mobile/` |
| Static website + APK download | plain HTML, served by nginx | `website/` |
| Server config | nginx, compose prod stack | `deploy/`, `docker-compose.prod.yml` |

## Feature areas (backend modules under `backend/internal/`)

- **auth** — phone + password login, one-time invite codes for registration /
  password reset, JWT access + rotating refresh tokens.
- **building** — buildings & units registry, manager scope (`user_buildings`),
  occupants/areas/participation.
- **billing** — billing periods with a draft→issued state machine, cost items
  (equal / per-person / per-area / fixed / specific-units / combined weights),
  snapshot-based charge calculation engine, invoice issue/cancel/adjustments.
- **payment** — resident payments (mock / Zarinpal gateway), unit balances.
- **expense** — manager expenses with receipt attachments and an
  approval workflow (pending → approved/rejected), monthly financial report.
- **maintenance** — resident repair requests with photos, assignment, status.
- **announcement** — building announcements with attachments.
- **notification** — in-app notifications (FCM push best-effort).
- **storage** — file uploads (images/PDF ≤ 5 MB) at `POST /files`, served by
  `GET /files/{id}`; bound to expenses (`receipt_file`), requests, announcements.
- **audit** — append-only audit trail of manager mutations (FR-038).

## Repository layout

```
backend/                 Go API server (cmd/server, internal/*, migrations/)
mobile/                  Flutter app (lib/features/*, RTL, Persian l10n)
website/                 static landing page + hamsa-release.apk
deploy/                  nginx site conf + production server notes
specs/                   feature specs & plans (speckit)
.github/workflows/       CI: APK build/sign/release (build-apk.yml)
docker-compose.yml       dev stack (PostgreSQL)
docker-compose.prod.yml  production stack (postgres + backend behind nginx)
DEPLOY.md                server deployment & APK release guide
Makefile                 dev tasks (make help)
```

## Development

Prerequisites: Go 1.25, Flutter (stable), Docker (for PostgreSQL).

```sh
make up                                   # PostgreSQL on :5432
cp backend/config.example.yaml backend/config.yaml   # git-ignored

# backend (dev config enables console SMS + mock payment gateway)
cd backend && go build -o bin/server ./cmd/server && ./bin/server
# health probe: curl http://localhost:8080/healthz

# mobile (API base URL defaults to the Android emulator loopback)
cd mobile && flutter run \
  --dart-define=API_BASE_URL=http://localhost:8080/api/v1
```

Generated code (json_serializable/freezed) is committed for CI-free analysis;
regenerate with `make mobile-gen` after model changes.

## Tests & checks

```sh
make backend-test   # go test ./... — integration tests need a reachable
                    # PostgreSQL (localhost:5432 or HAMSA_TEST_DSN); they
                    # self-create/drop a throwaway schema per test
make mobile-lint    # flutter analyze
make mobile-test    # flutter test --no-pub
```

`make lint` runs both linters; CI runs the same suite before building the APK.

## Migrations

Plain SQL under `backend/migrations/` (`NNNN_*.up.sql` / `.down.sql`), applied
in order at startup when `db.auto_migrate: true` (dev config: enabled).
Rollback is manual — run the matching `.down.sql` yourself.

## Deployment & releases

- **Server**: `138.124.34.204` runs `docker-compose.prod.yml` under
  `/opt/hamsa` behind system nginx (TLS via certbot); `hamsa-home.ir` serves
  the static site + APK, `app.hamsa-home.ir` proxies the API, uploads persist
  in the `hamsa_files` volume. See `deploy/DEPLOY.md` and `DEPLOY.md`.
- **APK**: `.github/workflows/build-apk.yml` builds on pushes to `main`
  (artifact) and on `v*` tags (signed APK attached to a GitHub Release).
  Signing uses repo secrets with a debug-signing fallback; version comes from
  `mobile/pubspec.yaml`. Steps in `DEPLOY.md` §2.
