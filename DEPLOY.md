# Hamsa — Deployment & Release Guide

## 1. Backend (Docker Compose)

Stack: `postgres` (db) + `server` (Go backend, port 8080). The Flutter app is
not containerized — it ships as an APK (section 2).

### First deploy / redeploy

> **Production note:** for the public server use `docker-compose.prod.yml`
> (behind the system nginx, secrets via `--env-file .env`, port bound to
> loopback). The steps below describe the root `docker-compose.yml` stack.

```bash
cp backend/config.example.yaml backend/config.yaml   # once; git-ignored
# edit backend/config.yaml — see checklist below
docker compose up -d --build server
curl -s http://localhost:8080/healthz   # → {"status":"ok"}
```

`db.auto_migrate: true` applies pending migrations on every startup. There is
**no automatic rollback**: undoing a schema change requires manually running
the matching `backend/migrations/NNNN_*.down.sql` against the database before
(or after) downgrading the image.

### config.yaml checklist

| Key | Value |
|---|---|
| `app.env` | `production` (disables mock/dev behaviors) |
| `app.addr` | `":8080"` |
| `app.base_url` | Public origin, e.g. `https://app.hamsa-home.ir` (payment callbacks) |
| `db.dsn` | Host must be `postgres` (compose service name), e.g. `host=postgres port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable` |
| `db.auto_migrate` | `true` on deploy machines |
| `auth.jwt_secret` | 32+ random bytes — **change from the example** |
| `payment.provider` | `mock` until Zarinpal merchant credentials exist |
| `storage.path` | `/data/files` (compose mounts the `hamsa_files` volume there) |

The config stays file-based — the server reads `HAMSA_CONFIG=/app/config.yaml`
(mounted `./backend/config.yaml`, read-only). Secrets live in that file only;
never commit it.

### File storage

Uploads (expense receipts, maintenance photos, announcement attachments) are
stored on disk at `/data/files`, backed by the named volume `hamsa_files`.
Back it up alongside the database — the `files` registry table references
paths inside that volume.

### JWT secret rotation

`auth.jwt_secret` signs access + refresh tokens. Rotating it instantly
invalidates every issued token (users must log in again); refresh-token
sessions also break. To rotate: change `auth.jwt_secret` in `config.yaml`, then
`docker compose up -d server`. Rotate on suspected compromise only — there is
no graceful dual-secret window.

### Password auth bootstrap

Login is phone + password. The first account is created via
`POST /api/v1/auth/setup`; manager scopes are granted through
`user_buildings` (or by the manager issuing invite codes).

## 2. Android APK releases

Workflow: `.github/workflows/build-apk.yml` (actions-ubuntu runner).

- **Triggers**: push to `main` (artifact only) and tag pushes `v*`
  (artifact + GitHub Release). Manual `workflow_dispatch` also available.
- **Signing**: the workflow decodes the `ANDROID_KEYSTORE_BASE64` secret to
  `mobile/android/app/upload-keystore.jks` and writes
  `mobile/android/key.properties` from the `ANDROID_KEY_ALIAS`,
  `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_PASSWORD` secrets. Without the
  secrets the Gradle config falls back to debug signing, so builds never fail.
  `key.properties` and `*.jks`/`*.keystore` are git-ignored
  (`mobile/android/.gitignore`) — never commit them.

### One-time: prepare the keystore secret

```bash
keytool -genkey -v -keystore upload-keystore.jks -keyalg RSA \
  -keysize 2048 -validity 10000 -alias hamsa
base64 -w0 upload-keystore.jks   # → GitHub secret ANDROID_KEYSTORE_BASE64
```

Set repo secrets: `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEY_ALIAS`,
`ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_PASSWORD`.
**Keep a backup of the keystore** — losing it means the app can never be
updated in place on installed devices.

### Cutting a release

1. Bump the version in `mobile/pubspec.yaml` (e.g. `version: 1.2.0+3` —
   `name+buildNumber`).
2. Commit and tag:

   ```bash
   git tag v1.2.0
   git push origin main v1.2.0
   ```

3. The workflow runs `pub get → build_runner → analyze → test → build apk`,
   names the APK after the pubspec version
   (`hamsa-1.2.0+3-release.apk`), uploads it as a workflow artifact, and
   attaches it to the GitHub Release for the tag.
