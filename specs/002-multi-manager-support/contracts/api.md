# REST API Contract Delta: Multi-Manager Support with Superadmin Bootstrap

**Feature**: specs/002-multi-manager-support
**Base**: All conventions, error envelope, auth model, and unlisted endpoints
remain as specified in
[specs/001-building-management-mvp/contracts/api.md](../../001-building-management-mvp/contracts/api.md).
This file documents only what changes or is added. During implementation the
001 tables are updated in place with these rows.

## Changed endpoints

| Method | Path | Change |
|---|---|---|
| POST | `/auth/setup` | Body unchanged. The first account is now role `superadmin` (was `manager`). Still allowed only while zero active users exist, else `409 CONFLICT`. Response session body carries `"role": "superadmin"`. |
| POST | `/auth/register` | Body unchanged. Redeeming invite assigns the **invite's role** to a NEW user (`resident` \| `manager`). For an EXISTING phone (including the superadmin) behavior is unchanged: password reset only, never a role change. Errors unchanged (`401` invalid/expired/consumed code). |
| POST | `/auth/invites` | Now `superadmin` or `manager` (residents `403`). Body `{ "phone", "role"? }` — `role` defaults `"resident"`; accepted values `"resident"` \| `"manager"`; any other value (incl. `"superadmin"`) → `400 VALIDATION_ERROR` `"نقش دعوت نامعتبر است."`. Response **201** `{ "code", "role", "expires_in_days": 7 }` (clean cutover; old shape removed). Audits `invite.issued` (best-effort) with `{phone, role}`. All invite guarantees unchanged: phone-bound, one-time, 7-day expiry, hashed storage. |
| GET | `/auth/me` | Role `superadmin` returns `"buildings": []` (no scope lookup). Manager/resident behavior unchanged. |

## New endpoints — Building managers (superadmin-eligible? **no**: manager-only, per-building)

Mounted under `/buildings/{id}` with `Authorization: Bearer` + role
`manager` + object check `IsManagerOf(caller, {id})`. A manager **not granted
to that building** gets `403 FORBIDDEN` on all three; the superadmin also
gets `403` (excluded from building management by design).

### GET `/buildings/{id}/managers`

`200` →

```json
{
  "items": [
    {
      "user_id": "uuid",
      "phone": "09120000000",
      "name": "مدیر ساختمان",
      "role": "manager",
      "granted_at": "2026-09-02T10:15:00Z"
    }
  ]
}
```

Ordered by `granted_at` ascending. Superadmin never appears.

### POST `/buildings/{id}/managers`

Body `{ "phone": "09xxxxxxxxx" }` → `201` with the created row (same shape as
list item). `granted_at` = now.

| Outcome | Status | Code | Message (fa) |
|---|---|---|---|
| Grant already exists | 409 | `CONFLICT` | `این مدیر از قبل دسترسی دارد.` |
| Phone is not an active user | 404 | `NOT_FOUND` | `کاربری با این شماره یافت نشد.` |
| Phone belongs to the superadmin | 400 | `VALIDATION_ERROR` | `نمی‌توان سرپرست ارشد را مدیر ساختمان کرد.` |
| Caller not a manager of `{id}` | 403 | `FORBIDDEN` | `دسترسی غیرمجاز است.` |

Side effect: if the target's role was `resident` it becomes `manager`
(raise-only, explicit, audited); the response's `role` reflects the
resulting role. Audits `building.manager_granted` (`object_id` = building id,
`after_value` = `{user_id, role}`).

### DELETE `/buildings/{id}/managers/{userId}`

`204` on success (no body).

| Outcome | Status | Code | Message (fa) |
|---|---|---|---|
| `userId` is the caller | 400 | `VALIDATION_ERROR` | `برای حذف دسترسی خود ابتدا مدیر دیگری معرفی کنید.` |
| Would leave 0 managers | 409 | `CONFLICT` | `حداقل یک مدیر باید برای ساختمان باقی بماند.` |
| Grant does not exist | 404 | `NOT_FOUND` | `این مدیر دسترسی‌ای برای این ساختمان ندارد.` |
| Caller not a manager of `{id}` | 403 | `FORBIDDEN` | `دسترسی غیرمجاز است.` |

Last-manager check is atomic (single guarded DELETE — research R9). Audits
`building.manager_revoked` (`before_value` = `{user_id}`, `after_value` = null).

## Non-endpoints (guaranteed absent)

- No role-demotion, superadmin-create/replace/delete, or self-revoke route.
- `/auth/invites` never accepts `superadmin` as a role value.
- The managers routes never accept the superadmin as caller or target.

## Client consumption (Flutter)

- `POST /auth/invites` — invite screen sends chosen `role`
  (ساکن/مدیر segmented button) and renders `role` in the result banner.
- Managers screen consumes all three new routes through the shared
  `apiClientProvider` Dio client; server Persian messages rendered via the
  existing `describeError` envelope mapping (no client-side re-translation).
- Reachability: invite screen for `role ∈ {manager, superadmin}`; managers
  row-action for `role == manager` (building list is server-scoped to
  granted buildings).
