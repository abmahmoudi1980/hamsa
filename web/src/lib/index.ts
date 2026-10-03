/**
 * Public entry point for the library layer (`#lib`).
 *
 * Prefer importing the specific module you need — `#lib/api/http`,
 * `#lib/format/money` — so the dependency graph stays obvious. This barrel
 * exists for the few types that are genuinely used everywhere.
 */

export type { ApiErrorCode, ApiErrorDetail } from './api/apiError';
export { ApiError } from './api/apiError';
export type { Toman } from './format/money';
export { formatToman, formatTomanWithUnit } from './format/money';
export type { JalaliDate } from './format/jalali';
export { formatJalaliDate, toIsoDate } from './format/jalali';
