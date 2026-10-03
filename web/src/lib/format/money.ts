/**
 * Toman money handling.
 *
 * Invariants (contracts/api.md, research.md R7):
 *   - amounts are INTEGER Toman, never fractions;
 *   - the wire carries them as JSON **strings** to avoid JS float precision
 *     loss, so a `Toman` is a branded string and all arithmetic goes through
 *     BigInt;
 *   - rendering is Persian digits with U+066C grouping;
 *   - a negative amount must be bidi-isolated or the sign detaches.
 *
 * `Number` is never used on an amount. That is not stylistic: a building's
 * annual charges can exceed Number.MAX_SAFE_INTEGER in aggregate, and Toman
 * integers at the top of the range are where a float silently loses a digit.
 */

import {
	groupDigits,
	isolateNumber,
	isStrictlyNumeric,
	normalizeNumericInput,
	THOUSANDS_SEPARATOR,
	toPersianDigits
} from './digits';

/** Integer Toman on the wire (e.g. "2600000"). */
export type Toman = string & { readonly __brand: 'Toman' };

/** Anything accepted where a Toman is expected, before validation. */
export type TomanLike = Toman | string | number | bigint;

/** Maximum amount that can be converted to a JavaScript number without loss. */
const MAX_SAFE_TOMAN = BigInt(Number.MAX_SAFE_INTEGER);

/** Builds a `Toman`, rejecting numeric inputs that are not safe integers. */
export function toman(value: TomanLike): Toman {
	const asBig = toBigInt(value);
	return asBig.toString() as Toman;
}

/** True when the value is an integer Toman amount. */
export function isToman(value: unknown): value is Toman {
	if (typeof value === 'number') return Number.isSafeInteger(value);
	if (typeof value === 'bigint') return true;
	if (typeof value !== 'string' || value === '') return false;
	return /^-?\d+$/.test(value);
}

/**
 * Parses an amount from the wire.
 *
 * Tolerant of the three digit scripts, thousands separators, whitespace and
 * bidi controls — because a client may legitimately send Persian digits, and
 * the backend normalizes them rather than 400-ing (see ManualPaymentInput).
 *
 * Rejects anything else, including a fractional value: the contract guarantees
 * integer Toman, so "1.5" is a contract violation that must surface rather than
 * be silently truncated to 15.
 */
export function parseToman(value: TomanLike | null | undefined): Toman | null {
	if (value === null || value === undefined) return null;
	return parseNumeric(String(value));
}

/**
 * Parses a money FORM FIELD.
 *
 * Same numeric rules as the wire parser, but a malformed value THROWS so the
 * field can show a message, whereas `parseToman` returns null. Empty input is
 * null in both cases: "not filled in" is not "zero".
 */
export function parseTomanInput(input: string): Toman | null {
	if (!isStrictlyNumeric(input)) {
		throw new RangeError('مبلغ باید فقط شامل رقم باشد.');
	}
	const normalized = normalizeNumericInput(input);
	if (normalized === null) return null;
	return normalized as Toman;
}

function parseNumeric(input: string): Toman | null {
	if (!isStrictlyNumeric(input)) return null;
	const normalized = normalizeNumericInput(input);
	if (normalized === null) return null;
	return normalized as Toman;
}

/** BigInt arithmetic, so summing a ledger never loses precision. */
export function addToman(...values: TomanLike[]): Toman {
	let sum = 0n;
	for (const value of values) sum += toBigInt(value);
	return clamp(sum);
}

export function subtractToman(minuend: TomanLike, subtrahend: TomanLike): Toman {
	return clamp(toBigInt(minuend) - toBigInt(subtrahend));
}

/** Non-negative difference: `a - b` floored at zero. */
export function positiveDifference(a: TomanLike, b: TomanLike): Toman {
	const diff = toBigInt(a) - toBigInt(b);
	return clamp(diff > 0n ? diff : 0n);
}

export function sumToman(values: readonly TomanLike[]): Toman {
	return values.reduce<Toman>((acc, v) => addToman(acc, v), '0' as Toman);
}

/** Number only — for charts and percentage math, never for a stored amount. */
export function tomanToNumber(value: TomanLike): number {
	const amount = toBigInt(value);
	if (amount > MAX_SAFE_TOMAN || amount < -MAX_SAFE_TOMAN) {
		throw new RangeError('مبلغ برای تبدیل به عدد خارج از محدوده امن است.');
	}
	return Number(amount);
}

/** Formats a signed amount with Persian digits and U+066C grouping. */
export function formatToman(value: TomanLike): string {
	const big = toBigInt(value);
	const negative = big < 0n;
	const grouped = groupDigits((negative ? -big : big).toString(), true);
	return isolateNumber(toPersianDigits(negative ? `-${grouped}` : grouped));
}

/** `۲٬۶۰۰٬۰۰۰ تومان` — the standard on-screen form. */
export function formatTomanWithUnit(value: TomanLike, unit = 'تومان'): string {
	return `${formatToman(value)} ${unit}`;
}

/** Latin digits + grouping, for copy/paste and CSV export. */
export function formatTomanLatin(value: TomanLike): string {
	const big = toBigInt(value);
	const negative = big < 0n;
	const grouped = groupDigits((negative ? -big : big).toString(), false);
	return negative ? `-${grouped}` : grouped;
}

/** Groups with ASCII commas, for CSV/Excel exports that don't take U+066C. */
export function formatTomanForCsv(value: TomanLike): string {
	return formatTomanLatin(value).split(THOUSANDS_SEPARATOR).join(',');
}

/**
 * Applies a percentage to an amount, rounding half-up to whole Toman.
 * Used by late-fee rules (spec §10, FR-021) and the preview's rough estimate.
 */
export function percentOf(amount: TomanLike, percent: number): Toman {
	const basis = toBigInt(amount) * BigInt(Math.round(percent * 100));
	// Round half away from zero, matching the engine's largest-remainder method.
	const quotient = basis >= 0n ? (basis + 5000n) / 10000n : (basis - 5000n) / 10000n;
	return clamp(quotient);
}

function toBigInt(value: TomanLike): bigint {
	if (typeof value === 'bigint') return value;
	if (typeof value === 'number') {
		if (!Number.isSafeInteger(value)) {
			throw new RangeError(`مبلغ نامعتبر است: ${value}`);
		}
		return BigInt(value);
	}
	if (typeof value === 'string') {
		const normalized = normalizeNumericInput(value);
		if (!isStrictlyNumeric(value) || normalized === null) {
			throw new RangeError(`مبلغ نامعتبر است: ${value}`);
		}
		return BigInt(normalized);
	}
	throw new TypeError('مبلغ نامعتبر است.');
}

function clamp(value: bigint): Toman {
	return value.toString() as Toman;
}
