import { describe, expect, it } from 'vitest';
import {
	addToman,
	formatToman,
	formatTomanForCsv,
	formatTomanLatin,
	formatTomanWithUnit,
	isToman,
	parseToman,
	parseTomanInput,
	percentOf,
	positiveDifference,
	subtractToman,
	sumToman,
	toman,
	tomanToNumber
} from './money';
import { latinDigitsOnly, stripBidiControls, toPersianDigits } from './digits';

/**
 * Research R1: this suite pins the EXACT output of the money formatter. The
 * plan explicitly forbade assuming whether the browser's ICU emits U+066C or an
 * ASCII comma as the group separator for `fa-IR` — so the rendering here is
 * computed from the digit table (matching the mobile client's `groupToman`)
 * and asserted literally. If a future change swaps in `Intl.NumberFormat`, this
 * suite is what will catch the regression.
 */
describe('formatToman', () => {
	// The formatted value is bidi-isolated; tests assert the visible glyphs.
	const visible = (s: string) => stripBidiControls(s);

	it('renders Persian digits with U+066C grouping', () => {
		expect(visible(formatToman('2600000'))).toBe('۲٬۶۰۰٬۰۰۰');
		expect(visible(formatToman('0'))).toBe('۰');
		expect(visible(formatToman('7'))).toBe('۷');
		expect(visible(formatToman('1000'))).toBe('۱٬۰۰۰');
		expect(visible(formatToman('999'))).toBe('۹۹۹');
		expect(visible(formatToman('1000000'))).toBe('۱٬۰۰۰٬۰۰۰');
	});

	it('uses U+066C, not an ASCII comma, as the group separator', () => {
		expect(visible(formatToman('1000'))).toContain('٬');
		expect(visible(formatToman('1000'))).not.toContain(',');
	});

	it('wraps the value in a bidi isolate', () => {
		// U+2066 … U+2069, required so a negative sign cannot detach from the
		// digit groups (the exact bug fixed in money_text.dart).
		const out = formatToman('2600000');
		expect(out.codePointAt(0)).toBe(0x2066);
		expect(out.codePointAt(out.length - 1)).toBe(0x2069);
	});

	it('renders a negative amount with the sign inside the isolate', () => {
		const out = formatToman('-15700000');
		expect(visible(out)).toBe('-۱۵٬۷۰۰٬۰۰۰');
		// The sign must sit INSIDE the isolate, not outside it.
		expect(out.startsWith('⁦-')).toBe(true);
		expect(out.endsWith('⁩')).toBe(true);
	});

	it('appends the Toman unit', () => {
		expect(formatTomanWithUnit('2600000')).toBe('⁦۲٬۶۰۰٬۰۰۰⁩ تومان');
	});

	it('renders Latin digits for copy/paste', () => {
		expect(formatTomanLatin('2600000')).toBe('2٬600٬000');
		expect(formatTomanLatin('-5000')).toBe('-5٬000');
	});

	it('renders ASCII commas for CSV export', () => {
		expect(formatTomanForCsv('2600000')).toBe('2,600,000');
	});
});

describe('toman', () => {
	it('accepts string, number and bigint inputs', () => {
		expect(toman('2600000')).toBe('2600000');
		expect(toman(2600000)).toBe('2600000');
		expect(toman(2600000n)).toBe('2600000');
		expect(toman('9007199254740992')).toBe('9007199254740992');
	});

	it('rejects an unsafe number', () => {
		// Silently losing a digit on a Toman amount is not acceptable.
		expect(() => toman(Number.MAX_SAFE_INTEGER + 2)).toThrow(RangeError);
	});

	it('rejects a non-integer number', () => {
		expect(() => toman(1.5)).toThrow(RangeError);
	});

	it('rejects garbage', () => {
		expect(() => toman('abc' as never)).toThrow(RangeError);
	});
});

describe('isToman', () => {
	it('accepts safe integers in any accepted representation', () => {
		expect(isToman('2600000')).toBe(true);
		expect(isToman(2600000)).toBe(true);
		expect(isToman('-2600000')).toBe(true);
		expect(isToman('0')).toBe(true);
	});

	it('rejects malformed values but accepts large integer strings', () => {
		expect(isToman('1.5')).toBe(false);
		expect(isToman('')).toBe(false);
		expect(isToman('12a')).toBe(false);
		expect(isToman(null)).toBe(false);
		expect(isToman(undefined)).toBe(false);
		expect(isToman('99999999999999999999')).toBe(true);
	});
});

describe('parseToman', () => {
	it('parses the wire format', () => {
		expect(parseToman('2600000')).toBe('2600000');
	});

	it('accepts Persian digits and separators from user input', () => {
		expect(parseToman('۲٬۶۰۰٬۰۰۰')).toBe('2600000');
		expect(parseToman('2600000')).toBe('2600000');
		expect(parseToman('2,600,000')).toBe('2600000');
	});

	it('preserves a negative sign', () => {
		expect(parseToman('-۱۵۰۰۰')).toBe('-15000');
	});

	it('returns null for empty input, so "not filled in" is distinguishable', () => {
		expect(parseToman('')).toBeNull();
		expect(parseToman('   ')).toBeNull();
		expect(parseToman(null)).toBeNull();
	});

	it('rejects stray characters instead of silently changing the amount', () => {
		expect(parseToman('12abc34')).toBeNull();
	});

	it('returns null for a fractional wire value rather than truncating it', () => {
		// The contract guarantees integer Toman, so "1.5" is a violation worth
		// surfacing — silently paying 15 would be worse.
		expect(parseToman('1.5')).toBeNull();
	});
});

describe('parseTomanInput', () => {
	it('returns null for an empty field', () => {
		expect(parseTomanInput('')).toBeNull();
	});

	it('accepts a bidi-isolated or LRM-marked value pasted from the UI', () => {
		const rendered = formatToman('2600000');
		expect(parseTomanInput(rendered)).toBe('2600000');
	});

	it('accepts Persian digits and separators', () => {
		expect(parseTomanInput('۲٬۶۰۰٬۰۰۰')).toBe('2600000');
		expect(parseTomanInput('2,600,000')).toBe('2600000');
	});

	it('rejects stray characters rather than silently changing the amount', () => {
		// The reason this parser exists separately from the lenient one.
		expect(() => parseTomanInput('12abc34')).toThrow(RangeError);
		expect(() => parseTomanInput('1000000 تومان')).toThrow(RangeError);
		expect(() => parseTomanInput('1-2')).toThrow(RangeError);
	});

	it('accepts a value beyond the safe-number range as an exact integer string', () => {
		expect(parseTomanInput('99999999999999999999')).toBe('99999999999999999999');
	});
});

describe('arithmetic', () => {
	it('adds without float precision loss', () => {
		expect(addToman('100', '200')).toBe('300');
		expect(sumToman(['1', '2', '3'])).toBe('6');
		expect(sumToman([])).toBe('0');
	});

	it('stays exact across the safe-integer boundary', () => {
		// Two halves summing to exactly MAX_SAFE_INTEGER: a float sum of large
		// magnitudes is not trustworthy here, so the ledger must not use one.
		expect(addToman('4503599627370495', '4503599627370496')).toBe('9007199254740991');
	});

	it('keeps sums exact beyond the safe-number range', () => {
		expect(addToman('9007199254740991', '1')).toBe('9007199254740992');
		expect(addToman('99999999999999999999', '1')).toBe('100000000000000000000');
	});

	it('subtracts and floors differences at zero', () => {
		expect(subtractToman('100', '30')).toBe('70');
		expect(positiveDifference('100', '30')).toBe('70');
		expect(positiveDifference('30', '100')).toBe('0');
		expect(positiveDifference('100', '100')).toBe('0');
	});

	it('converts to Number only while the amount remains exactly representable', () => {
		expect(() => tomanToNumber('9007199254740992')).toThrow(RangeError);
		expect(tomanToNumber('9007199254740991')).toBe(Number.MAX_SAFE_INTEGER);
	});
});

describe('percentOf', () => {
	it('computes a percentage and rounds to whole Toman', () => {
		expect(percentOf('1000000', 10)).toBe('100000');
		expect(percentOf('1000000', 2.5)).toBe('25000');
		expect(percentOf('100', 1)).toBe('1');
		expect(percentOf('100', 0.5)).toBe('1'); // rounds half up
		expect(percentOf('100', 0)).toBe('0');
	});

	it('handles a percentage of zero', () => {
		expect(percentOf('1000000', 0)).toBe('0');
	});
});

describe('digit helpers', () => {
	it('maps Latin digits to Persian', () => {
		expect(toPersianDigits('0123456789')).toBe('۰۱۲۳۴۵۶۷۸۹');
		expect(toPersianDigits('unit 101')).toBe('unit ۱۰۱');
	});

	it('normalizes Persian, Arabic-Indic and Latin digits to Latin', () => {
		expect(latinDigitsOnly('۱۲۳')).toBe('123');
		expect(latinDigitsOnly('١٢٣')).toBe('123'); // Arabic-Indic
		expect(latinDigitsOnly('1۲٣')).toBe('123');
		expect(latinDigitsOnly('no digits')).toBe('');
	});
});
