---
name: hamsa-run-dev
description: Build and run the Hamsa building-management stack for manual testing — Go backend on :8080 and the Flutter app on Chrome. Use when asked to run, launch, start, rebuild, or restart the backend and/or frontend (Chrome/web) in D:\apps\hamsa, or when the user wants to test the app themselves.
---

# Run Hamsa backend + Flutter web

## Backend (Go, port 8080)

1. **PostgreSQL must be up on 5432 first.** Docker Desktop's CLI is NOT in the
   shell PATH — check with `netstat -ano | findstr :5432 | findstr LISTENING`.
   If not listening, ask the user to start Docker Desktop (container: db `hamsa`
   per root `docker-compose.yml`).
2. **Config**: `backend/config.yaml` must exist (copy from
   `backend/config.example.yaml`; git-ignored). Dev defaults are correct:
   dev mode (OTP `dev_code` in response, mock payment gateway, console SMS),
   DSN localhost:5432/hamsa, `auto_migrate: true` runs pending migrations.
3. **Build**: `cd backend && go build -o bin/server.exe ./cmd/server`
4. **Run in background** (`async: true`, `timeout: 0`), cwd `backend/`:

   ```
   D:/apps/hamsa/backend/bin/server.exe
   ```

   Use the ABSOLUTE path — relative `./bin/server.exe` fails with
   "command not found" in this bash. `hub start` is unavailable (daemon
   broker ENOENT); use bash background jobs.
5. **Verify**: `curl -s http://localhost:8080/healthz` → `{"status":"ok"}`.
   Migrations apply on first start; expect a few seconds of startup logs.

If a `server.exe` is already listening on 8080 (check
`netstat -ano | findstr :8080 | findstr LISTENING`), it is likely a stale
build: kill with `taskkill /PID <pid> /F` and relaunch after rebuilding.

## Frontend (Flutter web on Chrome, port 5219)

Run in background (`async: true`, `timeout: 0`), cwd `mobile/`:

```
flutter run --no-pub -d chrome --web-port 5219 --dart-define=API_BASE_URL=http://localhost:8080/api/v1
```

- **`--no-pub` is REQUIRED**: implicit `pub get` fails on this machine
  ("authorization failed" against pub.dev). Dependencies are already resolved
  in `.dart_tool`; never run `flutter pub get` here. Same for tests:
  `flutter test --no-pub`.
- `flutter run -d chrome` opens the Chrome window automatically.
- First build takes ~15–45 s. Poll readiness:
  `curl -s -o /dev/null -w "%{http_code}" http://localhost:5219/` → 200.

### Restart after code changes

`flutter run` stays attached and does NOT hot-reload from a detached job.

1. Find and kill the listener: `netstat -ano | findstr :5219 | findstr LISTENING`
   → `taskkill /PID <pid> /T /F` (kills the whole dart/flutter tool tree).
2. Relaunch the command above; re-poll port 5219.

Backend changes: rebuild (step 3 above), kill PID on 8080, relaunch, re-probe
`/healthz`. Frontend does not need a rebuild for backend-only changes.

## Testing the OTP login flow (dev)

```
POST http://localhost:8080/api/v1/auth/otp/request   {"phone":"09xxxxxxxxx"}
→ 200 {"dev_code":"NNNNNN"}        (60 s resend throttle per phone!)
POST http://localhost:8080/api/v1/auth/otp/verify   {"phone":"...","code":"..."}
→ {"access_token", "refresh_token", "user"}
```

- A second `otp/request` for the same phone within 60 s → `429 RATE_LIMITED`
  ("لطفاً کمی صبر کنید...") — by design, not a bug. Wait or change phone.
- First registered user in a fresh DB becomes manager (bootstrap, T024).
- CORS is enabled for `*` in `httpx.CORS()` — Chrome calls work as-is.
