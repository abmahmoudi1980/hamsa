/**
 * Financial-report month helpers (FR-027).
 *
 * The API aggregates by Gregorian month (`?month=YYYY-MM`), while the product
 * shows Jalali only. Mirroring the Flutter client, the UI lets the manager pick
 * any Jalali date and derives two things from it: the Gregorian month key sent
 * on the wire, and the Jalali month label shown in the picker.
 */

import { toPersianDigits } from '#lib/format/digits';
import { jalaliMonthName, toJalali } from '#lib/format/jalali';

/** Gregorian `YYYY-MM` key the report endpoint expects, from a local date. */
export function monthKey(date: Date): string {
	const year = String(date.getFullYear()).padStart(4, '0');
	const month = String(date.getMonth() + 1).padStart(2, '0');
	return `${year}-${month}`;
}

/** `مرداد ۱۴۰۴` — Jalali month label for a reference date. */
export function reportMonthLabel(date: Date): string {
	const jalali = toJalali(date);
	return `${jalaliMonthName(jalali.month)} ${toPersianDigits(String(jalali.year))}`;
}
