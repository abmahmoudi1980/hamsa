# REST API Contract: Building Management MVP (P0)

**Feature**: specs/001-building-management-mvp
**Audience**: Flutter Android client (the only consumer in P0)

## Conventions

- **Base URL**: `/api/v1`
- **Content type**: `application/json; charset=utf-8` (Persian text in UTF-8)
- **Language**: The product is Persian-only — every user-facing message (error `message`, notification title/body, validation details) is returned **in Persian**; the client renders Persian directly with no translation layer. Structured fields (codes, enums, ids) remain locale-neutral English tokens.
- **Authentication**: `Authorization: Bearer <access_token>` (JWT, 15 min) on every route except the auth endpoints. Refresh via `POST /auth/refresh`.
- **Authorization model**: server resolves per-request what the caller may see — a resident token can only access its own unit's data; a manager token only its permitted buildings. Authorization failures return `403` regardless of resource existence.
- **Dates**: ISO-8601 (`2026-08-26`, `2026-08-26T14:00:00Z`) on the wire — the client **displays and accepts Jalali dates only** (Jalali date picker everywhere; conversion to ISO-8601 happens in the client's date helpers). Users never see or enter a Gregorian date. Money: integer Toman strings in JSON (serialized as string to avoid JS-side precision loss; e.g. `"2600000"`); the client renders amounts with Persian digits and thousands separators.
- **Pagination**: `?page=1&page_size=20` → envelope `{ "items": [...], "page": 1, "page_size": 20, "total": 142 }`.
- **Error envelope** (all non-2xx):

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "شماره واحد در این ساختمان تکراری است",
    "details": [
      { "field": "number", "rule": "unique_per_building" }
    ]
  }
}
```

- **Standard error codes**: `VALIDATION_ERROR` (400), `UNAUTHENTICATED` (401), `FORBIDDEN` (403), `NOT_FOUND` (404), `CONFLICT` (409, e.g. duplicate, illegal state transition), `RATE_LIMITED` (429), `INTERNAL` (500).

## Endpoints

### Auth (P0-10)

| Method | Path | Description |
|---|---|---|
| POST | `/auth/setup` | Body `{ "phone", "password", "name?" }` → `{ "access_token", "refresh_token", "expires_in", "user": { id, name, role } }`. First-user bootstrap: allowed only while zero active users exist (otherwise `409 CONFLICT`); the first account is the manager. |
| POST | `/auth/register` | Body `{ "phone", "code", "password", "name?" }` → session body. Redeems a one-time manager-issued invite code bound to the phone: unknown phones register as residents; existing users get their password reset (recovery path). `401` on invalid/expired/consumed code. |
| POST | `/auth/login` | Body `{ "phone", "password" }` → session body. `401` (same code) for wrong password or unknown phone — no account enumeration. |
| POST | `/auth/password` | Bearer. Body `{ "current_password", "new_password" }` → `204`. Password policy: ≥ 8 characters with at least one letter and one digit. |
| POST | `/auth/invites` | Bearer, manager only. Body `{ "phone" }` → `201 { "code", "expires_in_days": 7 }`. One-time invite code, shown once, redeemable for that phone only. |
| POST | `/auth/refresh` | Body `{ "refresh_token" }` → new token pair (rotates refresh token; reuse of old token revokes the family). |
| POST | `/auth/logout` | Revokes the refresh token family. |
| GET | `/auth/me` | Current user + role context: manager → permitted buildings; resident → occupied units. |

### Buildings & Units (P0-01) — manager

| Method | Path | Description |
|---|---|---|
| GET / POST | `/buildings` | List permitted buildings / create building. |
| GET / PUT | `/buildings/{id}` | Building detail / update. |
| GET | `/buildings/{id}/units` | List units — filter: `?q=&block=&floor=&status=`; sort; paginated (FR search/filter). |
| POST | `/buildings/{id}/units` | Create unit — `409 CONFLICT` on duplicate number (FR-003). |
| GET / PUT / DELETE | `/units/{id}` | Detail (soft delete) / update (audited) / archive. |
| GET | `/units/{id}/history` | Change history (audit entries for this unit). |
| GET | `/units/{id}/balance` | Current unit balance (components per spec §9). |

### People & Occupancy (P0-02) — manager

| Method | Path | Description |
|---|---|---|
| GET / POST | `/buildings/{id}/persons` | List/create person (name, phone, national_id?). |
| PUT / DELETE | `/persons/{id}` | Update / soft-archive. |
| GET / POST | `/units/{id}/occupancies` | Occupancy history / add person-unit link (owner, tenant, non_resident_owner + dates). Adding a new active tenant closes the previous one (FR-007). |
| PATCH | `/occupancies/{id}` | End-date an occupancy (preserve history). |
| GET / POST | `/units/{id}/occupant-count` | History of occupant counts / record new value (FR-008). |

### Billing — periods, cost items, calculation (P0-03) — manager

| Method | Path | Description |
|---|---|---|
| GET / POST | `/buildings/{id}/periods` | List/create billing period (title, dates, due date, late-fee config). |
| GET / PATCH | `/periods/{id}` | Detail / edit (only in `draft`). |
| GET / POST | `/periods/{id}/cost-items` | List/add cost item (title, total_amount, method, combo_weights?, include_vacant, unit_ids?). |
| PUT / DELETE | `/cost-items/{id}` | Edit/remove (only while period `draft`). |
| POST | `/periods/{id}/calculate` | Run charge engine → creates/refreshes `cost_item_shares` + draft invoices; period → `calculated`. Returns per-unit preview. Validates: participating-factor > 0, specific-units non-empty (edge cases). |
| GET | `/periods/{id}/preview` | Review breakdown **before issuance** (BR-08, FR-014): per cost item — method, total, each unit's exact + rounded share; per unit — resulting invoice with prior debt, late fee, credit, final amount. |
| POST | `/periods/{id}/reopen` | `calculated → draft` (discards previews). Rejected with `409` when period already issued. |
| POST | `/periods/{id}/issue` | Issue all invoices: freezes snapshots, assigns invoice numbers, sets `issued_at`, emits `invoice_issued` notifications. Idempotent guard. |
| POST | `/periods/{id}/close` | `issued → closed`. |

### Invoices (P0-03/P0-04)

| Method | Path | Description |
|---|---|---|
| GET | `/buildings/{id}/invoices` | Manager list — filter `?period_id=&unit_id=&status=`; paginated. |
| GET | `/invoices/{id}` | Detail: items (charge/late-fee/adjustment kinds), amounts, dates, status — same response used by resident (authorization scopes it). |
| POST | `/invoices/{id}/cancel` | Manager: cancel with `{ "reason" }` — row preserved (BR-10), audited. |
| POST | `/invoices/{id}/adjustments` | Manager: explicit correction `{ "kind": "debit"\|"credit", "amount", "reason" }` — creates adjustment item; original amounts untouched (FR-017). |
| GET | `/me/invoices` | Resident: own unit's invoices. |
| GET | `/me/invoices/{id}` | Resident: own invoice detail with cost-item methods (transparency). |

### Payments & Balances (P0-04)

| Method | Path | Description |
|---|---|---|
| POST | `/invoices/{id}/payments` | Manager: record manual payment `{ amount, paid_at, tracking_number?, method? }` → updates invoice status (`partial`/`paid`) and unit balance; surplus → credit. |
| POST | `/invoices/{id}/pay` | Resident: start gateway payment `{ amount? }` (defaults to outstanding) → `{ payment_url, payment_id }`; opens gateway page. |
| GET | `/payments/callback` | Gateway return URL — verifies server-side (amount taken **from our DB**, not the callback), marks payment `verified`, updates balances, returns deeplink for app. |
| GET | `/buildings/{id}/payments` | Manager payment ledger — filter by unit/date/method. |
| GET | `/me/payments` | Resident payment history. |

### Expenses & Financial Report (P0-05) — manager

| Method | Path | Description |
|---|---|---|
| GET / POST | `/buildings/{id}/expenses` | List (filter category/date/approval) / create `{ title, category, amount, expense_date, description?, payer_person_id?, receipt_file_id?, approval_status }`. |
| PUT / DELETE | `/expenses/{id}` | Update / soft-delete (audited). |
| GET | `/buildings/{id}/financial-report?month=YYYY-MM` | Monthly income, monthly expense, net, total resident debt, total payments, total expenses (FR-027). |

### Maintenance (P0-06)

| Method | Path | Description |
|---|---|---|
| POST | `/me/maintenance-requests` | Resident submits `{ title, category, description, location, photo_file_id?, priority }` (FR-028). |
| GET | `/me/maintenance-requests` | Resident: own requests with status timeline. |
| GET / PATCH | `/buildings/{id}/maintenance-requests` | Manager list (filter status/priority/category) / batch update — per item: `{ status?, assignee_person_id?, recorded_cost?, note? }`. Illegal transitions → `409`. |
| PATCH | `/maintenance-requests/{id}` | Manager update incl. close; each change notified to submitter (FR-029/FR-030). |

### Announcements (P0-07)

| Method | Path | Description |
|---|---|---|
| GET / POST | `/buildings/{id}/announcements` | Manager list / publish `{ title, body, publish_at?, expire_at?, audience_type, audience_value?, attachment_file_id? }`. |
| PUT / DELETE | `/announcements/{id}` | Edit / delete (manager). |
| GET | `/me/announcements` | Resident: announcements targeting own unit/block/floor/all, within publish/expire window. |
| POST | `/announcements/{id}/read` | Mark read (drives unread count). |

### Notifications

| Method | Path | Description |
|---|---|---|
| GET | `/me/notifications?unreadOnly=true` | In-app notification center, paginated. |
| POST | `/me/notifications/{id}/read` | Mark one read. |
| POST | `/me/notifications/read-all` | Mark all read. |

### Resident Panel (P0-08)

| Method | Path | Description |
|---|---|---|
| GET | `/me/home` | Single call for the resident home screen: payable amount, latest invoice (id, final amount, due date, status), open request count + latest request status, latest announcements, unread announcement count (FR-034). |

### Manager Dashboard (P0-09)

| Method | Path | Description |
|---|---|---|
| GET | `/buildings/{id}/dashboard` | Cards (unit count, debtor units, total debt, month income, month expense, open requests), quick-action targets, alerts list (debtor units, past-due invoices, open requests, pending expenses) (FR-035). |

### Files

| Method | Path | Description |
|---|---|---|
| POST | `/files` | Multipart upload (image/PDF ≤ 5 MB) → `{ id, url }`. Used for receipts, request photos, announcement attachments. |

## Key Contract Behaviors

1. **Immutability guarantee**: no endpoint modifies issued-invoice amounts. The only mutations after issuance are: `cancel`, `adjustments`, payment-driven `status`/`paid_amount` updates.
2. **Resident isolation**: every `/me/*` route resolves the unit scope server-side; there is no route where a resident can name an arbitrary unit/invoice id and read it (object-level authorization check returns `403`).
3. **Calculation correctness surfaced**: the preview response includes both exact and rounded shares and the reconciliation check (Σ rounded = total) so the client can display the BR-08/BR-09 verification.
4. **Payment verification**: gateway callbacks are never trusted for amounts; verification reads the pending payment from the DB and compares against the gateway verify API (Zarinpal requires amount from our side).
5. **Sample-scenario acceptance**: `POST /periods/{id}/calculate` on the spec's §25 scenario must produce unit 1's invoice = 2,600,000 — this is a contract test case.
6. **Persian-only product**: all user-facing text produced by the API (error messages, notification content, validation details) is Persian; enum values, error codes, and object types on the wire stay locale-neutral tokens that the client maps to Persian labels. No English UI exists in P0.
