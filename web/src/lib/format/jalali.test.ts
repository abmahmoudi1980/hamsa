import { describe, expect, it } from 'vitest';
import {
	fromIsoDate,
	isJalaliLeapYear,
	isValidJalali,
	jalaliDaysInMonth,
	JALALI_MONTH_NAMES,
	toGregorian,
	toIsoDate,
	toJalali,
	type JalaliDate
} from './jalali';

/**
 * The gate that makes hand-written date math acceptable (research R3).
 *
 * Every day from 1300-01-01 to 1500-12-29 — roughly 73,000 dates — must
 * satisfy jalaliOf(gregorianOf(j)) === j. A single mismatch anywhere in the
 * range means a manager picks a wrong due date somewhere, so this is not a
 * sampling test.
 */
describe('Jalali round-trip', () => {
	const START_YEAR = 1300;
	const END_YEAR = 1500;

	it('round-trips every day of 1300–1500 SH', () => {
		const mismatches: string[] = [];
		let checked = 0;

		for (let year = START_YEAR; year <= END_YEAR; year++) {
			const monthCount = isJalaliLeapYear(year) ? 12 : 12;
			for (let month = 1; month <= monthCount; month++) {
				const days = jalaliDaysInMonth(year, month);
				for (let day = 1; day <= days; day++) {
					const jalali: JalaliDate = { year, month, day };
					const round = toJalali(toGregorian(jalali));
					checked += 1;
					if (round.year !== year || round.month !== month || round.day !== day) {
						if (mismatches.length < 10) {
							mismatches.push(
								`${year}-${month}-${day} -> ${round.year}-${round.month}-${round.day}`
							);
						}
					}
				}
			}
		}

		expect(checked).toBeGreaterThan(73000);
		expect(mismatches).toEqual([]);
	});

	it('produces a strictly increasing day sequence across the range', () => {
		// A conversion that silently repeats or skips a day would still pass a
		// round-trip at sampled dates, so this walks the Gregorian side day by
		// day and asserts the Jalali side only ever moves forward.
		let cursor = new Date(1921, 2, 22); // ~1300-01-01
		let previous = toJalali(cursor);

		for (let i = 0; i < 4000; i++) {
			const next = new Date(cursor.getFullYear(), cursor.getMonth(), cursor.getDate() + 1);
			const current = toJalali(next);
			const forward =
				current.year > previous.year ||
				(current.year === previous.year &&
					(current.month > previous.month ||
						(current.month === previous.month && current.day === previous.day + 1)));
			expect(
				forward,
				`non-monotonic step at ${next.toDateString()} (${previous.year}-${previous.month}-${previous.day} -> ${current.year}-${current.month}-${current.day})`
			).toBe(true);
			previous = current;
			cursor = next;
		}
	});
});

describe('known anchor dates', () => {
	// Independently verifiable anchors: Nowruz (1 Farvardin) is 21 March in a
	// common year and 21/22 March around leap years; these pin the epoch.
	const cases: Array<[JalaliDate, [number, number, number]]> = [
		[{ year: 1404, month: 1, day: 1 }, [2025, 3, 21]],
		[{ year: 1403, month: 1, day: 1 }, [2024, 3, 20]],
		[{ year: 1400, month: 1, day: 1 }, [2021, 3, 21]],
		[{ year: 1399, month: 1, day: 1 }, [2020, 3, 20]],
		[{ year: 1405, month: 12, day: 29 }, [2027, 3, 20]]
	];

	it.each(cases)('%o -> %o', (jalali, [year, month, day]) => {
		const g = toGregorian(jalali);
		expect([g.getFullYear(), g.getMonth() + 1, g.getDate()]).toEqual([year, month, day]);
	});

	it('puts a mid-year date in the right Jalali month', () => {
		const g = toGregorian({ year: 1404, month: 6, day: 31 });
		const back = toJalali(g);
		expect(back).toEqual({ year: 1404, month: 6, day: 31 });
	});
});

describe('leap years', () => {
	// The Iranian calendar produces runs of leap years: 1395, 1399, 1403 and
	// 1408 are all leap, which is why a naive "every 4th year" rule is wrong.
	it('treats 1399 as a leap year', () => {
		expect(isJalaliLeapYear(1399)).toBe(true);
		expect(jalaliDaysInMonth(1399, 12)).toBe(30);
	});

	it('treats 1403 as a leap year', () => {
		expect(isJalaliLeapYear(1403)).toBe(true);
		expect(jalaliDaysInMonth(1403, 12)).toBe(30);
	});

	it('treats 1404 as a common year', () => {
		expect(isJalaliLeapYear(1404)).toBe(false);
		expect(jalaliDaysInMonth(1404, 12)).toBe(29);
	});

	it('treats 1402 as a common year', () => {
		expect(isJalaliLeapYear(1402)).toBe(false);
		expect(jalaliDaysInMonth(1402, 12)).toBe(29);
	});

	it('rejects 12/30 in a common year', () => {
		expect(isValidJalali(1404, 12, 30)).toBe(false);
		expect(isValidJalali(1404, 12, 29)).toBe(true);
	});

	it('accepts 12/30 in a leap year', () => {
		expect(isValidJalali(1399, 12, 30)).toBe(true);
	});
});

describe('month lengths', () => {
	it('uses 31 days for months 1–6 and 30 for 7–11', () => {
		for (let month = 1; month <= 6; month++) {
			expect(jalaliDaysInMonth(1404, month)).toBe(31);
		}
		for (let month = 7; month <= 11; month++) {
			expect(jalaliDaysInMonth(1404, month)).toBe(30);
		}
	});

	it('names all twelve months in Persian', () => {
		expect(JALALI_MONTH_NAMES).toHaveLength(12);
		expect(JALALI_MONTH_NAMES[0]).toBe('فروردین');
		expect(JALALI_MONTH_NAMES[11]).toBe('اسفند');
	});
});

describe('isValidJalali', () => {
	it('rejects out-of-range parts', () => {
		expect(isValidJalali(1404, 0, 1)).toBe(false);
		expect(isValidJalali(1404, 13, 1)).toBe(false);
		expect(isValidJalali(1404, 1, 0)).toBe(false);
		expect(isValidJalali(1404, 1, 32)).toBe(false);
	});

	it('rejects non-integers', () => {
		expect(isValidJalali(1404.5, 1, 1)).toBe(false);
		expect(isValidJalali(1404, 1.5, 1)).toBe(false);
	});
});

describe('ISO boundary conversion', () => {
	it('formats a local date without shifting it across UTC', () => {
		// toISOString() would roll this back a day in a negative-offset timezone.
		expect(toIsoDate(new Date(2026, 7, 26))).toBe('2026-08-26');
		expect(toIsoDate(new Date(2026, 0, 1))).toBe('2026-01-01');
		expect(toIsoDate(new Date(2026, 11, 31))).toBe('2026-12-31');
	});

	it('parses an ISO date back to the same local day', () => {
		const parsed = fromIsoDate('2026-08-26');
		expect(parsed).not.toBeNull();
		expect(toIsoDate(parsed!)).toBe('2026-08-26');
	});

	it('returns null for malformed or impossible dates', () => {
		expect(fromIsoDate('2026-02-31')).toBeNull();
		expect(fromIsoDate('2026-13-01')).toBeNull();
		expect(fromIsoDate('not-a-date')).toBeNull();
		expect(fromIsoDate('')).toBeNull();
		expect(fromIsoDate(null)).toBeNull();
	});

	it('truncates a full ISO timestamp to its date part', () => {
		expect(toIsoDate(fromIsoDate('2026-08-26T14:00:00Z')!)).toBe('2026-08-26');
	});
});
