# Data Model: Building Management MVP (P0)

**Feature**: specs/001-building-management-mvp
**Storage**: PostgreSQL 16 — all monetary values are `BIGINT` Toman; all timestamps are `TIMESTAMPTZ` (UTC); all dates are `DATE` (Gregorian, Jalali only at presentation). Soft-delete (`deleted_at`) everywhere financial/transactional records exist (spec §21: records must survive user/unit deletion).

## Entity Relationship Overview

```text
users ──(1:N)── user_buildings (manager scope)
persons ──(1:N)── occupancies ──(N:1)── units ──(N:1)── buildings
units ──(1:N)── occupant_count_history
buildings ──(1:N)── billing_periods ──(1:N)── cost_items ──(1:N)── cost_item_shares (calc result)
billing_periods ──(1:N)── invoices ──(1:N)── invoice_items / payments / invoice_adjustments
units ──(1:1)── unit_balances (current financial position)
buildings ──(1:N)── expenses / announcements / maintenance_requests
users ──(1:N)── notifications / maintenance_requests (submitter) / audit_logs
```

## Entities

### users
Authentication identity; login is mobile number + password (bcrypt). First account is created via setup bootstrap; residents register with one-time manager-issued invite codes (migration 0009).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| phone | VARCHAR(11) | UNIQUE, Iranian mobile format `09xxxxxxxxx` |
| password_hash | VARCHAR(255) | NULL until the user sets a password; bcrypt |
| role | ENUM(`manager`, `resident`) | A user may manage buildings AND reside in others → role resolved per-context via `user_buildings` (manager) and `occupancies` (resident); `role` field records primary role |
| name | VARCHAR(120) | |
| fcm_token | VARCHAR | NULL — used best-effort for push |
| is_active | BOOLEAN | soft-deactivation only; never hard-deleted |
| created_at / updated_at / deleted_at | TIMESTAMPTZ | |

### user_buildings
Manager permission scope (FR-037: manager sees only permitted buildings).

| Field | Type | Rules |
|---|---|---|
| user_id | UUID FK→users | PK(user_id, building_id) |
| building_id | UUID FK→buildings | |
| granted_at | TIMESTAMPTZ | |

### buildings
| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| name | VARCHAR(150) | required |
| address | TEXT | |
| block_count / floor_count / unit_count | INT | ≥ 0 informational |
| built_year | INT | nullable, Jalali year |
| manager_phone / emergency_phone | VARCHAR(11) | |
| notes | TEXT | |
| created_at / updated_at / deleted_at | TIMESTAMPTZ | |

### units
| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| building_id | UUID FK→buildings | indexed |
| number | VARCHAR(20) | UNIQUE per building (FR-003, unique index `(building_id, number)` where `deleted_at IS NULL`) |
| block | VARCHAR(20) | nullable |
| floor | INT | |
| area_m2 | INT | > 0; input to per-area calculation |
| parking_count / parking_numbers | INT / TEXT | |
| storage_count / storage_numbers | INT / TEXT | |
| status | ENUM(`active`,`vacant`,`occupied`,`inactive`) | |
| notes | TEXT | |
| created_at / updated_at / deleted_at | TIMESTAMPTZ | soft-delete preserves financial history |

**Validation**: unit number non-empty; area > 0; occupant count ≥ 0. Change history via `audit_logs` (FR-004).

### persons
A natural person (owner / tenant / non-resident owner). A person may relate to many units; a unit to many persons (FR-005, FR-006).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| building_id | UUID FK→buildings | scoped per building for P0 simplicity |
| full_name | VARCHAR(150) | required |
| phone | VARCHAR(11) | indexed; may link to a `users` row at login |
| national_id | VARCHAR(10) | nullable; checksum-validated when present |
| created_at / updated_at / deleted_at | TIMESTAMPTZ | |

### occupancies
Dated, historical person↔unit link; records are **closed (end-dated), never deleted** when a resident changes (FR-007).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| unit_id | UUID FK→units | |
| person_id | UUID FK→persons | |
| relationship | ENUM(`owner`, `tenant`, `non_resident_owner`) | |
| start_date / end_date | DATE | end_date NULL = active |
| is_active | BOOLEAN | derived from end_date; only one active `tenant` per unit at a time |
| created_at | TIMESTAMPTZ | |

**Validation**: `end_date >= start_date`; new active occupancy for a unit with the same relationship closes the previous one.

### occupant_count_history
Independent occupant-count data point feeding the charge engine, with full history (FR-008, BR-04).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| unit_id | UUID FK→units | |
| occupant_count | INT | ≥ 0 |
| effective_from | DATE | |
| created_at / created_by | TIMESTAMPTZ / UUID | |

Current count = row with max `effective_from` ≤ billing period reference date (engine reads **as-of** value, snapshot at calculation).

### billing_periods
| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| building_id | UUID FK→buildings | |
| title | VARCHAR(120) | e.g. "مهر ۱۴۰۵" |
| start_date / end_date / due_date | DATE | `due_date >= end_date >= start_date` |
| late_fee_type | ENUM(`none`,`fixed`,`percent`,`per_day`) | building-level default, overridable per period |
| late_fee_value | BIGINT / NUMERIC(5,2) | interpretation depends on type (fixed Toman / % / Toman-per-day) |
| status | ENUM(`draft`,`calculated`,`issued`,`closed`) | see state transitions |
| calculated_at / issued_at / closed_at | TIMESTAMPTZ | |
| created_at / updated_at | TIMESTAMPTZ | |

**State transitions** (one per building per period month is recommended, not enforced):
`draft → calculated` (engine run) `→ issued` (invoices published) `→ closed` (final). `calculated → draft` allowed (re-open before issue). `issued` is a point of no return for auto-changes: corrections only via `invoice_adjustments`. Back to `draft` from `issued` is **forbidden**.

### cost_items
A cost line of a period (spec §7). Defines method + inclusion rules (BR-06, BR-07).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| period_id | UUID FK→billing_periods | |
| title | VARCHAR(150) | e.g. "شارژ عمومی", "آب", "پارکینگ" |
| total_amount | BIGINT | > 0, Toman |
| method | ENUM(`equal`,`per_occupant`,`per_area`,`fixed`,`specific_units`,`combined`) | |
| fixed_amount_per_unit | BIGINT | for `fixed` |
| combo_weights | JSONB | for `combined`: `[{"method":"equal","weight":50},{"method":"per_area","weight":30},{"method":"per_occupant","weight":20}]` — weights sum to 100 |
| include_vacant | BOOLEAN | default false |
| unit_ids | UUID[] | for `specific_units`; empty = all eligible |
| created_at / updated_at | TIMESTAMPTZ | |

### cost_item_shares
Calculation output per cost item per unit — the reviewable breakdown (BR-08, FR-014).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| cost_item_id | UUID FK→cost_items | |
| unit_id | UUID FK→units | |
| exact_share | NUMERIC(20,4) | exact rational result, kept for audit |
| rounded_share | BIGINT | after largest-remainder; Σ(rounded_share) ≡ total_amount per cost item (BR-09) |
| inputs_snapshot | JSONB | occupant count, area, participating-unit count **as used** |

### invoices
The immutable per-unit bill (spec §11). Created at calculation; frozen at issuance.

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| invoice_number | VARCHAR(30) | UNIQUE, sequential per building (e.g. `BLD-2026-0001`) |
| building_id / period_id / unit_id | UUID FKs | indexed |
| base_amount | BIGINT | Σ invoice_items |
| prior_debt | BIGINT | snapshot of unit balance before this period |
| late_fee_amount | BIGINT | computed per late-fee rules; separate item (FR-016) |
| credit_amount | BIGINT | snapshot of unit credit before this period |
| final_amount | BIGINT | `base_amount + prior_debt + late_fee_amount − credit_amount` (FR-015) |
| issue_date / due_date | DATE | |
| status | ENUM(`unpaid`,`partial`,`paid`,`expired`,`cancelled`) | expired set by due-date job; cancelled preserves row (BR-10) |
| paid_amount | BIGINT | running total of applied payments |
| inputs_frozen_at | TIMESTAMPTZ | time inputs were snapshotted |
| created_at / updated_at | TIMESTAMPTZ | UPDATE restricted to status/paid_amount once `issued` |

**Immutability**: amount fields and items are written at calculation and **never updated** (BR-03/BR-05; DB trigger guard). Manual correction → `invoice_adjustments`.

### invoice_items
| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| invoice_id | UUID FK→invoices | |
| kind | ENUM(`charge`,`late_fee`,`adjustment`) | late fee and adjustments are separate displayable rows |
| title | VARCHAR(150) | |
| cost_item_id | UUID FK→cost_items | nullable (non-charge kinds) |
| amount | BIGINT | |
| method | ENUM | as used — visible in resident detail |
| adjustment_id | UUID FK→invoice_adjustments | nullable |

### invoice_adjustments
Explicit correction operation (FR-017, BR-03). Never mutates the original amounts.

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| invoice_id | UUID FK→invoices | |
| kind | ENUM(`debit`,`credit`) | debit increases what the unit owes; credit decreases |
| amount | BIGINT | > 0 |
| reason | TEXT | required |
| created_by / created_at | UUID / TIMESTAMPTZ | audited |

### payments
Manual or gateway-verified transactions (spec §12; BR-02).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| building_id / unit_id / invoice_id | UUID FKs | invoice nullable for unallocated payments |
| method | ENUM(`manual`,`gateway`) | |
| amount | BIGINT | > 0, Toman |
| paid_at | DATE | |
| tracking_number | VARCHAR(50) | gateway ref / manual receipt no |
| gateway | VARCHAR(20) | `zarinpal` etc.; NULL for manual |
| recorded_by | UUID FK→users | FR-022 |
| status | ENUM(`recorded`,`verified`,`failed`,`reversed`) | gateway payments start pending → verified on callback verify |
| created_at | TIMESTAMPTZ | never hard-deleted; reversal via status + compensating entry |

**Allocation rule**: a payment applies to the invoice's outstanding amount first; any surplus becomes unit **credit** (edge case: overpayment). Partial payments allowed.

### unit_balances
Current financial position per unit — exactly one row per unit (BR-01), recomputed transactionally on every payment/adjustment/invoice event.

| Field | Type | Rules |
|---|---|---|
| unit_id | UUID PK FK→units | |
| prior_debt | BIGINT | carried debt before the current open invoice |
| current_invoice_amount | BIGINT | |
| late_fee_total | BIGINT | |
| credit | BIGINT | |
| paid_total | BIGINT | |
| balance | BIGINT | `prior_debt + current_invoice_amount + late_fee_total − paid_total − credit` (spec §9 formula) |
| updated_at | TIMESTAMPTZ | |

### expenses
| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| building_id | UUID FK→buildings | |
| title | VARCHAR(150) | required |
| category | ENUM(`water`,`electricity`,`gas`,`elevator`,`cleaning`,`security`,`repair`,`insurance`,`equipment`,`other`) | spec §13 |
| amount | BIGINT | > 0 |
| expense_date | DATE | |
| description | TEXT | |
| payer_person_id | UUID FK→persons | nullable |
| receipt_file | VARCHAR(500) | storage path; image/PDF |
| approval_status | ENUM(`pending`,`approved`,`rejected`) | default pending |
| created_by / created_at / updated_at / deleted_at | UUID / TIMESTAMPTZ | |

### maintenance_requests
| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| building_id / unit_id | UUID FKs | |
| submitted_by | UUID FK→users | the resident (FR-028) |
| title | VARCHAR(150) | required |
| category | ENUM(`elevator`,`utilities`,`electrical`,`water`,`cleaning`,`common_area`,`parking`,`other`) | |
| description | TEXT | |
| location | VARCHAR(200) | free text within building |
| photo_file | VARCHAR(500) | nullable |
| priority | ENUM(`normal`,`important`,`urgent`) | |
| status | ENUM(`new`,`under_review`,`in_progress`,`done`,`closed`) | state machine below |
| assignee_person_id | UUID FK→persons | nullable |
| recorded_cost | BIGINT | nullable, ≥ 0 |
| created_at / updated_at / closed_at | TIMESTAMPTZ | |

**State transitions** (spec §14): `new → under_review → in_progress → done → closed`, with `under_review/in_progress/done → closed` allowed (early close). No backwards transitions. Each transition emits a notification (FR-033) and an audit entry (FR-038).

### announcements
| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| building_id | UUID FK→buildings | |
| title / body | VARCHAR(200) / TEXT | required |
| audience_type | ENUM(`all`,`block`,`floor`,`unit`) | spec §15 |
| audience_value | VARCHAR(20) / UUID | block name / floor no / unit_id per type |
| publish_at / expire_at | TIMESTAMPTZ | visible only in window (edge case: expired → hidden, retained) |
| attachment_file | VARCHAR(500) | nullable |
| created_by / created_at | UUID / TIMESTAMPTZ | |

### announcement_reads
Tracks unread counts for the resident panel (FR-034).

| Field | Type | Rules |
|---|---|---|
| announcement_id / user_id | UUID FKs | PK(announcement_id, user_id) |
| read_at | TIMESTAMPTZ | |

### notifications
In-app guaranteed channel (FR-032/FR-033).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| user_id | UUID FK→users | indexed with is_read |
| type | ENUM(`invoice_issued`,`due_soon`,`overdue`,`payment_recorded`,`request_submitted`,`request_status_changed`,`announcement_published`) | |
| title / body | VARCHAR(200) / TEXT | |
| ref_type / ref_id | VARCHAR(30) / UUID | deep-link target (invoice, request, announcement) |
| is_read / read_at | BOOLEAN / TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | |

### audit_logs
Append-only (spec §20, FR-038).

| Field | Type | Rules |
|---|---|---|
| id | BIGINT PK | |
| user_id | UUID | nullable (system actions) |
| action | VARCHAR(60) | e.g. `unit.updated`, `invoice.issued`, `payment.recorded` |
| object_type / object_id | VARCHAR(40) / UUID | |
| before_value / after_value | JSONB | required for sensitive ops (formula change, invoice issue/cancel, payment) |
| created_at | TIMESTAMPTZ | no UPDATE/DELETE ever — enforced by DB grants + trigger |

### invite_codes
One-time manager-issued codes: registration of new residents and password recovery (migration 0009).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| phone | VARCHAR(11) | the only phone the code redeems for; indexed (phone, created_at DESC) |
| code_hash | VARCHAR(64) | SHA-256, never plaintext |
| created_by | UUID FK→users | issuing manager |
| expires_at | TIMESTAMPTZ | +7 days |
| consumed_at | TIMESTAMPTZ | single redemption, enforced by conditional UPDATE |

### files
Unified attachment registry (receipts, photos, announcement attachments).

| Field | Type | Rules |
|---|---|---|
| id | UUID PK | |
| path | VARCHAR(500) | storage location |
| content_type / size_bytes | VARCHAR(100) / BIGINT | size ≤ 5 MB, images + PDF |
| uploaded_by / created_at | UUID / TIMESTAMPTZ | |

## Cross-Cutting Validation Rules
- **Soft delete**: `users`, `units`, `persons`, `expenses` soft-delete; `invoices`, `payments`, `audit_logs`, `occupancies`, `invite_codes` never delete.
- **Uniqueness**: unit number per building; invoice number globally; one unconsumed invite redemption per code.
- **Snapshot rule**: everything the charge engine consumes (occupant count, area, participation, balances) is copied into `cost_item_shares.inputs_snapshot` / invoice fields at calculation time — the single mechanism guaranteeing BR-03/BR-05 and the spec's reliability requirement.
- **Audit coverage** (FR-038): unit create/edit, resident change, charge-formula (cost item) change, invoice issue, invoice cancel, payment record, expense record, maintenance status change.
