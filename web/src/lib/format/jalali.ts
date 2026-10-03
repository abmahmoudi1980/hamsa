/**
 * Jalali (Solar Hijri) calendar conversion and Persian display formatting.
 *
 * Why this is hand-written rather than a library (research R3): a library
 * brings its own locale opinions about digits and separators, which is exactly
 * what the product constrains (Persian digits, U+066C grouping, Persian-only
 * UI). The conversion is therefore an integer-arithmetic implementation of the
 * standard Jalaali algorithm — the same one used by the widely-deployed
 * jalaali reference implementation — with no floating point anywhere.
 *
 * It is gated by an exhaustive round-trip suite over every day of 1300–1500 SH
 * (~73,000 cases) in jalali.test.ts. An earlier, "cleverer" day-number
 * implementation in this same file failed that suite immediately, so the gate is
 * doing its job.
 *
 * Storage and the wire stay Gregorian ISO-8601 (research.md R6). Everything here
 * is presentation- and input-only; `toIsoDate` is the single conversion used at
 * the API boundary.
 */

import { toPersianDigits } from './digits';

export interface JalaliDate {
	/** Solar Hijri year, e.g. 1404. */
	year: number;
	/** 1–12. */
	month: number;
	/** 1–31 (1–30 in Esfand of a leap year). */
	day: number;
}

export const JALALI_MONTH_NAMES: readonly string[] = [
	'فروردین',
	'اردیبهشت',
	'خرداد',
	'تیر',
	'مرداد',
	'شهریور',
	'مهر',
	'آبان',
	'آذر',
	'دی',
	'بهمن',
	'اسفند'
] as const;

/** Weekday names, indexed to match `Date.getDay()` (0 = Sunday). */
export const JALALI_WEEKDAY_NAMES: readonly string[] = [
	'یکشنبه',
	'دوشنبه',
	'سه‌شنبه',
	'چهارشنبه',
	'پنجشنبه',
	'جمعه',
	'شنبه'
] as const;

/** Years where the Jalaali leap cycle changes its pattern. */
const LEAP_BREAKS = [
	-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394,
	2456, 3178
] as const;

/**
 * Reads a known-present element without a non-null assertion.
 * Keeps `noUncheckedIndexedAccess` satisfied while staying explicit.
 */
function at(items: readonly number[], index: number): number {
	const value = items[index];
	if (value === undefined) throw new RangeError(`LEAP_BREAKS has no index ${index}`);
	return value;
}

/** Years this implementation is defined for (strictly inside LEAP_BREAKS). */
const MIN_JALALI_YEAR = at(LEAP_BREAKS, 0);
const MAX_JALALI_YEAR = at(LEAP_BREAKS, LEAP_BREAKS.length - 1) - 1;

/** Integer division that truncates toward zero. */
function div(a: number, b: number): number {
	return Math.trunc(a / b);
}

/** Remainder matching `div`'s truncation (JS `%` would disagree on negatives). */
function mod(a: number, b: number): number {
	return a - div(a, b) * b;
}

interface JalaliCalendarInfo {
	/** 0 when the year is a leap year. */
	leap: number;
	/** The Jalaali year this describes. */
	jalaliYear: number;
	/** Gregorian day-of-March on which Farvardin 1 falls (20 or 21). */
	march: number;
}

/**
 * Leap status and the Gregorian date of Nowruz for the Jalaali year overlapping
 * a given Gregorian year.
 *
 * The leap count from the calendar's epoch accumulates along the 33-year cycle
 * (8 leap years per cycle) and is corrected at every point where the pattern
 * shifts, which is why LEAP_BREAKS exists.
 */
function jalaliCalendarInfo(gregorianYear: number, wantLeap: boolean): JalaliCalendarInfo {
	const year = gregorianYear - 621;

	let leapCount = -14;
	let jump = 0;

	// Walk the break years in order, accumulating leap years between them.
	// `prevBreak` ends up as the last break at or below `year`, which is the
	// anchor the leap-year count and `n` are measured from.
	let prevBreak = at(LEAP_BREAKS, 0);
	for (const nextBreak of LEAP_BREAKS.slice(1)) {
		jump = nextBreak - prevBreak;
		if (year < nextBreak) break;
		leapCount += div(jump, 33) * 8 + div(mod(jump, 33), 4);
		prevBreak = nextBreak;
	}

	let n = year - prevBreak;
	leapCount += div(n, 33) * 8 + div(mod(n, 33) + 3, 4);
	if (mod(jump, 33) === 4 && jump - n === 4) leapCount += 1;

	// Leap years in the Gregorian calendar over the same span.
	const leapGregorian = div(gregorianYear, 4) - div((div(gregorianYear, 100) + 1) * 3, 4) - 150;

	const march = 20 + leapCount - leapGregorian;

	let leap = 0;
	if (wantLeap) {
		if (jump - n < 6) n = n - jump + div(jump + 4, 33) * 33;
		leap = mod(mod(n + 1, 33) - 1, 4);
		if (leap === -1) leap = 4;
	}

	return { leap, jalaliYear: year, march };
}

/**
 * Days since the Jalaali epoch, for a Gregorian date.
 * Integer arithmetic only — a float here would corrupt dates near the epoch.
 */
function gregorianToDayNumber(year: number, month: number, day: number): number {
	let d =
		div((year + div(month - 8, 6) + 100100) * 1461, 4) +
		div(153 * mod(month + 9, 12) + 2, 5) +
		day -
		34840408;
	d = d - div(div(year + 100100 + div(month - 8, 6), 100) * 3, 4) + 752;
	return d;
}

/** Inverse of gregorianToDayNumber: day number → Gregorian calendar date. */
function dayNumberToGregorian(dayNumber: number): {
	year: number;
	month: number;
	day: number;
} {
	let j = 4 * dayNumber + 139361631;
	j = j + div(div(4 * dayNumber + 183187720, 146097) * 3, 4) * 4 - 3908;

	const i = div(mod(j, 1461), 4) * 5 + 308;
	const day = div(mod(i, 153), 5) + 1;
	const month = mod(div(i, 153), 12) + 1;
	const year = div(j, 1461) - 100100 + div(8 - month, 6);

	return { year, month, day };
}

/** Day number of Nowruz (1 Farvardin) for a Jalaali year. */
function nowruzDayNumber(jalaliYear: number): number {
	const info = jalaliCalendarInfo(jalaliYear + 621, false);
	return gregorianToDayNumber(info.jalaliYear + 621, 3, info.march);
}

/**
 * Days from Nowruz of `year` to the given month/day.
 * Months 1–6 are 31 days and 7–11 are 30; the closed form matches that.
 */
function offsetFromNowruz(year: number, month: number, day: number): number {
	return (month - 1) * 31 - div(month, 7) * (month - 7) + day - 1;
}

/** True when the year is a leap year in the Jalaali calendar. */
export function isJalaliLeapYear(year: number): boolean {
	return jalaliCalendarInfo(year + 621, true).leap === 0;
}

/** Number of days in a Jalaali month (Esfand is 29, or 30 in a leap year). */
export function jalaliDaysInMonth(year: number, month: number): number {
	if (month <= 6) return 31;
	if (month <= 11) return 30;
	return isJalaliLeapYear(year) ? 30 : 29;
}

/** True for a Gregorian leap year (divisible by 4, not 100 unless also 400). */
export function isGregorianLeapYear(year: number): boolean {
	return (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;
}

/** Converts a Gregorian instant to its Jalaali calendar date (local time). */
export function toJalali(date: Date): JalaliDate {
	const dayNumber = gregorianToDayNumber(date.getFullYear(), date.getMonth() + 1, date.getDate());

	// The Jalaali year overlapping this Gregorian year, and its Nowruz.
	const info = jalaliCalendarInfo(date.getFullYear(), true);
	const nowruz = gregorianToDayNumber(info.jalaliYear + 621, 3, info.march);
	const offset = dayNumber - nowruz;

	if (offset >= 0 && offset <= 185) {
		// Months 1–6: 31 days each.
		return {
			year: info.jalaliYear,
			month: 1 + div(offset, 31),
			day: mod(offset, 31) + 1
		};
	}

	if (offset >= 0) {
		// Still in this Jalaali year, past month 6 (months 1–6 are 186 days).
		const into = offset - 186;
		return { year: info.jalaliYear, month: 7 + div(into, 30), day: mod(into, 30) + 1 };
	}

	// Before this year's Nowruz: the day belongs to the previous Jalaali year,
	// and specifically to its months 7–12 — which is why the same 186-day
	// offset applies. A Gregorian January date always lands here.
	const previousYear = info.jalaliYear - 1;
	const into = dayNumber - nowruzDayNumber(previousYear) - 186;
	return { year: previousYear, month: 7 + div(into, 30), day: mod(into, 30) + 1 };
}

/** Converts a Jalaali calendar date to a local Gregorian `Date` at midnight. */
export function toGregorian(date: JalaliDate): Date {
	const dayNumber = nowruzDayNumber(date.year) + offsetFromNowruz(date.year, date.month, date.day);
	const g = dayNumberToGregorian(dayNumber);
	return new Date(g.year, g.month - 1, g.day);
}

/** True when the parts form a real date (rejects e.g. 12/30 in a common year). */
export function isValidJalali(year: number, month: number, day: number): boolean {
	if (!Number.isInteger(year) || !Number.isInteger(month) || !Number.isInteger(day)) return false;
	if (month < 1 || month > 12 || day < 1) return false;
	return day <= jalaliDaysInMonth(year, month);
}

/** True when the year is inside this implementation's supported range. */
export function isSupportedJalaliYear(year: number): boolean {
	return year >= MIN_JALALI_YEAR && year <= MAX_JALALI_YEAR;
}

export function jalaliMonthName(month: number): string {
	return JALALI_MONTH_NAMES[month - 1] ?? '';
}

export function jalaliWeekdayName(date: Date): string {
	return JALALI_WEEKDAY_NAMES[date.getDay()] ?? '';
}

/**
 * The only wire-format conversion: Gregorian → `YYYY-MM-DD`.
 *
 * Built from local date parts, never `toISOString()`, so a date cannot shift by
 * a day in a timezone east or west of UTC.
 */
export function toIsoDate(date: Date): string {
	const y = String(date.getFullYear()).padStart(4, '0');
	const m = String(date.getMonth() + 1).padStart(2, '0');
	const d = String(date.getDate()).padStart(2, '0');
	return `${y}-${m}-${d}`;
}

/** Parses an ISO `YYYY-MM-DD` into a local `Date`; null when malformed. */
export function fromIsoDate(iso: string | null | undefined): Date | null {
	if (!iso) return null;
	const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso.slice(0, 10));
	if (!match) return null;
	const [, y, m, d] = match;
	const date = new Date(Number(y), Number(m) - 1, Number(d));
	// Reject overflow such as 2026-02-31, which Date would silently roll over.
	if (
		date.getFullYear() !== Number(y) ||
		date.getMonth() !== Number(m) - 1 ||
		date.getDate() !== Number(d)
	) {
		return null;
	}
	return date;
}

/**
 * Formats a date for display using the browser's own Persian calendar
 * (research R2), so digits, ordering and punctuation match the platform's
 * Persian locale exactly.
 */
export function formatJalaliDate(date: Date): string {
	return new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).format(date);
}

/** `۱ شهریور ۱۴۰۴` — month name spelled out, for prose contexts. */
export function formatJalaliLongDate(date: Date): string {
	const j = toJalali(date);
	return `${toPersianDigits(String(j.day))} ${jalaliMonthName(j.month)} ${toPersianDigits(String(j.year))}`;
}

/** Date + 24-hour time, both in Persian digits. */
export function formatJalaliDateTime(date: Date): string {
	const time = new Intl.DateTimeFormat('fa-IR', {
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	}).format(date);
	return `${formatJalaliDate(date)} ${time}`;
}

/** Persian calendar month label, e.g. `شهریور ۱۴۰۴`. */
export function jalaliMonthLabel(year: number, month: number): string {
	return `${jalaliMonthName(month)} ${toPersianDigits(String(year))}`;
}
