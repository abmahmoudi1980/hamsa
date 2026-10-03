/**
 * Digit and separator conversion for Persian display and tolerant input.
 *
 * The product renders Persian digits (۰-۹) everywhere, including inside form
 * inputs, and accepts all three digit forms a user might paste. Money grouping
 * uses U+066C (Arabic thousands separator) to match the mobile client exactly
 * (mobile/lib/shared/formatters/money_text.dart).
 */

const PERSIAN_ZERO = 0x06f0;
const ARABIC_ZERO = 0x0660;
const LATIN_ZERO = 0x30;

/** U+066C ARABIC THOUSANDS SEPARATOR — the group separator Persian text expects. */
export const THOUSANDS_SEPARATOR = '٬';

/** U+200E LEFT-TO-RIGHT MARK. */
const LRM = '‎';
/** U+2066 LEFT-TO-RIGHT ISOLATE … U+2069 POP DIRECTIONAL ISOLATE. */
const LRI = '⁦';
const PDI = '⁩';

const PERSIAN_DIGITS = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'] as const;

/** Maps a single Latin digit character to its Persian equivalent. */
function persianDigit(ch: string): string | null {
	const code = ch.charCodeAt(0);
	if (code < LATIN_ZERO || code > LATIN_ZERO + 9) return null;
	return PERSIAN_DIGITS[code - LATIN_ZERO] ?? null;
}

/** Replaces every Latin digit with its Persian counterpart; other text passes through. */
export function toPersianDigits(input: string): string {
	let out = '';
	for (const ch of input) {
		out += persianDigit(ch) ?? ch;
	}
	return out;
}

/**
 * Extracts only ASCII digits, normalizing Persian (U+06F0–06F9) and
 * Arabic-Indic (U+0660–0669) digits to Latin.
 *
 * Everything else — signs, separators, bidi controls, letters — is dropped, so
 * this is the safe way to read digits out of arbitrary user input.
 */
export function latinDigitsOnly(input: string): string {
	let out = '';
	for (const ch of input) {
		const code = ch.codePointAt(0) ?? 0;
		if (code >= LATIN_ZERO && code <= LATIN_ZERO + 9) {
			out += String.fromCharCode(code);
		} else if (code >= PERSIAN_ZERO && code <= PERSIAN_ZERO + 9) {
			out += String.fromCharCode(LATIN_ZERO + (code - PERSIAN_ZERO));
		} else if (code >= ARABIC_ZERO && code <= ARABIC_ZERO + 9) {
			out += String.fromCharCode(LATIN_ZERO + (code - ARABIC_ZERO));
		}
	}
	return out;
}

/**
 * Groups a plain digit string in threes with U+066C.
 * `2600000` → `۲٬۶۰۰٬۰۰۰` when Persian digits are requested.
 */
export function groupDigits(digits: string, persian = true): string {
	let out = '';
	for (let i = 0; i < digits.length; i++) {
		out += digits[i];
		const remaining = digits.length - i;
		if (remaining > 1 && (remaining - 1) % 3 === 0) out += THOUSANDS_SEPARATOR;
	}
	return persian ? toPersianDigits(out) : out;
}

/**
 * Wraps a possibly-negative number in a bidi isolate.
 *
 * Necessary because Persian digits are strongly right-to-left while a minus
 * sign and the digit groups read left-to-right: without isolation a negative
 * balance renders as `۵٬۷۰۰٬۵-۱۵` (sign detached, groups permuted). The mobile
 * client fixes the same bug with the same control characters.
 */
export function isolateNumber(text: string): string {
	return `${LRI}${text}${PDI}`;
}

/** Strips bidi control characters, so an isolated value can be parsed back. */
export function stripBidiControls(input: string): string {
	let out = '';
	for (const ch of input) {
		const code = ch.codePointAt(0) ?? 0;
		// LRM, RLM, LRI, RLI, FSI, PDI.
		if (code === 0x200e || code === 0x200f) continue;
		if (code >= 0x2066 && code <= 0x2069) continue;
		out += ch;
	}
	return out;
}

/**
 * Normalizes a user-typed number into a canonical `-?\d+` string.
 *
 * Accepts Persian/Arabic-Indic digits, both thousands separators, a Unicode
 * minus (U+2212), and bidi controls. Returns null when there are no digits at
 * all, so callers can distinguish "empty" from "invalid".
 */
export function normalizeNumericInput(input: string): string | null {
	const cleaned = stripBidiControls(input).replace(/−/g, '-');
	const negative = cleaned.includes('-');
	const digits = latinDigitsOnly(cleaned);
	if (digits === '') return null;
	return negative ? `-${digits}` : digits;
}

/** Anything permitted alongside digits in a form field. */
const NUMERIC_ALLOWED = /^[0-9٠-٩۰-۹٬,\s+-]*$/;

/**
 * True when the input holds only digits (in any of the three scripts), an
 * optional sign, thousands separators and whitespace.
 *
 * This is the guard a money form needs: `normalizeNumericInput` is lenient and
 * would turn "12abc34" into 1234, but a typo must not silently change a
 * payment amount. Reject here, accept leniently nowhere else.
 */
export function isStrictlyNumeric(input: string): boolean {
	const cleaned = stripBidiControls(input).replace(/−/g, '-');
	const trimmed = cleaned.trim();
	return (
		trimmed === '' ||
		(/^-?[0-9٠-٩۰-۹](?:[0-9٠-٩۰-۹٬,\s]*[0-9٠-٩۰-۹])?$/.test(trimmed) &&
			NUMERIC_ALLOWED.test(trimmed))
	);
}

/**
 * Extracts digits (normalizing Persian/Arabic-Indic to Latin) plus at most one
 * decimal point.
 *
 * Used by percentage fields, where a fraction like `۲.۵` is meaningful. It is
 * deliberately stricter than `normalizeNumericInput`: letters and stray dots are
 * dropped, and the caller decides whether the result is a valid number.
 */
export function latinDecimalOnly(input: string): string {
	let out = '';
	let seenDot = false;
	for (const ch of stripBidiControls(input)) {
		const code = ch.codePointAt(0) ?? 0;
		if (code >= LATIN_ZERO && code <= LATIN_ZERO + 9) {
			out += ch;
		} else if (code >= PERSIAN_ZERO && code <= PERSIAN_ZERO + 9) {
			out += String.fromCharCode(LATIN_ZERO + (code - PERSIAN_ZERO));
		} else if (code >= ARABIC_ZERO && code <= ARABIC_ZERO + 9) {
			out += String.fromCharCode(LATIN_ZERO + (code - ARABIC_ZERO));
		} else if (ch === '.' && !seenDot) {
			out += '.';
			seenDot = true;
		}
	}
	return out;
}

/** Left-to-right mark, for placing a currency suffix after an isolated number. */
export const LTR_MARK = LRM;
