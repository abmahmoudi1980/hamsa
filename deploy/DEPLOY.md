# Hamsa — Production Deployment (hamsa-home.ir)

Server: `138.124.34.204` (Ubuntu 24.04), deploy user `root` (SSH key auth).
Domain: `hamsa-home.ir` → server IP, TLS via Let's Encrypt (certbot, nginx plugin).

## Architecture

```
Internet ── 80  ──► nginx  (301 → https)
        ── 443 ──► nginx  TLS (Let's Encrypt)
                    ├── static site  /var/www/hamsa  (index.html + hamsa-release.apk)
                    └── /api/ ──► 127.0.0.1:8080 ──► docker: hamsa-api
                                                    └── docker: hamsa-pg (postgres 16)
```

- Backend: `backend/Dockerfile` (multi-stage Go build, alpine runtime, non-root
  user, migrations baked into image). Prod config baked at
  `backend/config.prod.yaml` (`HAMSA_CONFIG` points to it); secrets injected via
  environment only.
- Compose: `docker-compose.prod.yml` at `/opt/hamsa/` on the server; secrets in
  `/opt/hamsa/.env` (`POSTGRES_PASSWORD`, `JWT_SECRET`; chmod 600).
- Config env overrides honored by the backend: `HAMSA_APP_ENV`, `HAMSA_DB_DSN`,
  `HAMSA_JWT_SECRET`, `HAMSA_SMS_PROVIDER`, `HAMSA_SMS_SMSIR_API_KEY`,
  `HAMSA_SMS_SMSIR_TEMPLATE_ID`, `HAMSA_PAYMENT_PROVIDER`.
- Backend ports bind to loopback only (`127.0.0.1:8080`); nginx is the only
  public front. Volumes: `hamsa_pgdata` (DB), `hamsa_files` (uploaded files at
  `/data/files`).

## Current provider settings (placeholders, switch when credentials exist)

- SMS: `console` — OTP codes are written to the container log only.
- Payment: `mock` — auto-verifies; switch to `zarinpal` with a merchant id.
- `base_url` in the image is `https://hamsa-home.ir` (payment callbacks).

## Operations (on the server)

```sh
cd /opt/hamsa
docker compose -f docker-compose.prod.yml --env-file .env ps      # status
docker compose -f docker-compose.prod.yml --env-file .env logs -f backend
docker compose -f docker-compose.prod.yml --env-file .env restart backend
```

Boot persistence: `docker.service` and `nginx.service` are enabled; compose
services use `restart: unless-stopped`.

## Releasing a new backend image

Build locally (repo root, `backend/`):

```sh
cd backend && docker build -t hamsa-backend:latest .
docker save hamsa-backend:latest | gzip | \
  ssh root@138.124.34.204 'gunzip | docker load'
ssh root@138.124.34.204 'cd /opt/hamsa && \
  docker compose -f docker-compose.prod.yml --env-file .env up -d'
```

## Releasing the website

```sh
rsync -az --chmod=F644 website/ root@138.124.34.204:/var/www/hamsa/
```

The APK is built with the production API URL:

```sh
cd mobile && flutter build apk --release \
  --dart-define=API_BASE_URL=https://hamsa-home.ir/api/v1
cp build/app/outputs/flutter-apk/app-release.apk website/hamsa-release.apk
```

## nginx

- Site conf (repo copy is the source of truth): `deploy/nginx-hamsa.conf`
  → `/etc/nginx/sites-available/hamsa-home.ir` (symlinked into sites-enabled).
- `client_max_body_size 25m` for uploads; APK cached immutable, HTML no-cache.
- Cert renewal: `certbot.timer` (enabled); renewals use HTTP-01 on :80.

## Coexistence with x-ui/xray (VPN on this server)

The x-ui panel manages xray on this box. Port 443 was freed by removing the
panel's 443 inbound; nginx now owns :443. Caveat: if an x-ui inbound is ever
created again on port 443, it will collide with nginx — keep the panel's
inbounds off :443. A backup of the pre-change xray runtime config is at
`/usr/local/x-ui/bin/config.json.bak-hamsa`.
