/**
 * Parse/format a Jalali date *field* value (R6: the wire is Gregorian ISO; the
 * user only ever types and sees Solar Hijri).
 *
 * Input is deliberately tolerant: any digit script, `/`, `-`, `.` or spaces as
 * separators, and single-digit month/day (`۱۴۰۳/۵/۱۲`). Output is always the
 * canonical ISO `YYYY-MM-DD`, or null when the parts are not a real date.
 */

import { toPersianDigits } from './digits';
import {
	fromIsoDate,
	isSupportedJalaliYear,
	isValidJalali,
	toGregorian,
	toIsoDate,
	toJalali
} from './jalali';

/** Persian/Arabic-Indic digits → Latin, keeping every other character. */
function toLatinDigits(input: string): string {
	let out = '';
	for (const ch of input) {
		const code = ch.codePointAt(0) ?? 0;
		if (code >= 0x30 && code <= 0x39) out += ch;
		else if (code >= 0x06f0 && code <= 0x06f9) out += String.fromCharCode(code - 0x06f0 + 0x30);
		else if (code >= 0x0660 && code <= 0x0669) out += String.fromCharCode(code - 0x0660 + 0x30);
		else out += ch;
	}
	return out;
}

/** `1403/05/12` (or `۱۴۰۳/۵/۱۲`) → `2024-08-02`; null when invalid. */
export function parseJalaliInput(input: string): string | null {
	const cleaned = input.trim();
	if (cleaned === '') return null;

	const parts = toLatinDigits(cleaned)
		.split(/[^0-9]+/)
		.filter((part) => part !== '');

	let year: number;
	let month: number;
	let day: number;
	if (parts.length === 3) {
		year = Number(parts[0]);
		month = Number(parts[1]);
		day = Number(parts[2]);
	} else {
		// Compact form: exactly eight digits, YYYYMMDD.
		const digits = toLatinDigits(cleaned).replace(/\D/g, '');
		if (digits.length !== 8) return null;
		year = Number(digits.slice(0, 4));
		month = Number(digits.slice(4, 6));
		day = Number(digits.slice(6, 8));
	}

	if (!isSupportedJalaliYear(year) || !isValidJalali(year, month, day)) return null;
	return toIsoDate(toGregorian({ year, month, day }));
}

/** ISO `YYYY-MM-DD` → the Persian-digit `۱۴۰۳/۰۵/۱۲` shown in the field. */
export function formatJalaliInput(iso: string): string {
	const date = fromIsoDate(iso);
	if (!date) return '';
	const j = toJalali(date);
	const month = String(j.month).padStart(2, '0');
	const day = String(j.day).padStart(2, '0');
	return toPersianDigits(`${j.year}/${month}/${day}`);
}
