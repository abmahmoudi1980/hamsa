---
name: persian-rtl-mobile-ui
description: Persian (Farsi) RTL mobile UI conventions for the hamsa app — RTL layout correctness, Persian digits, Jalali dates, Vazirmatn typography, and Persian UX copy. Use when building or reviewing any user-facing screen, string, date, or number display in the hamsa Flutter app.
---

# Persian RTL Mobile UI (hamsa)

The hamsa app is **Persian-only, full RTL** (per plan.md: single `fa` locale, no English in P0). Every user-facing surface must follow these rules.

## RTL

- Start-aligned means **right** in Persian: use logical properties only — `EdgeInsetsDirectional`, `Align`/`CrossAxisAlignment` with start/end, `TextDirection.rtl` inherited from the app `locale`. Never `EdgeInsets.only(left: …)` for visual start/end padding.
- Icons with directional meaning (back, next, chevrons) must mirror in RTL — prefer `BackButton`/platform-adaptive icons; audit custom icons.
- Mixed-direction text (Persian + Latin brand/URLs/numbers): test that bidi isolation doesn't scramble; wrap Latin runs in LTR-isolating spans when needed.

## Digits & dates

- **Displayed numbers and amounts: Persian digits, integer Toman** — convert at presentation edge only (helper in `lib/core/`), store/transport Gregorian ISO-8601/UTC.
- **Dates: Jalali at presentation only** via `shamsi_date`; the only date entry component is the Jalali picker (`persian_datetime_picker`). Never add a Gregorian picker or `showDatePicker`.
- Money formatting goes through the shared money-display widget in `lib/shared/` — no ad-hoc `"${amount} تومان"` strings.

## Typography & copy

- Font: Vazirmatn (bundled); ensure Persian glyph shaping is intact (never set `fontFamily` overrides per-widget that drop the font).
- All UI strings live in the single `fa` localizations file (`l10n.yaml`, `lib/core/l10n/`) — **no hardcoded Persian strings in widgets**.
- Tone: concise, polite Persian; error messages name the problem and the fix ("کد وارد شده منقضی شده است. دوباره تلاش کنید").
- Server error messages arrive in Persian already; map API error envelopes to localized strings, don't concat raw server text with app copy.

## Review gate

Before delivering any screen: correct RTL mirroring, Persian digits on all numbers, Jalali on all dates, zero hardcoded strings, one `flutter run` visual check of the changed screen on Chrome.
