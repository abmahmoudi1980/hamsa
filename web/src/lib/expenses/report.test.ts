import { describe, expect, it } from 'vitest';
import { monthKey, reportMonthLabel } from './report';

describe('monthKey', () => {
	it('zero-pads the Gregorian year and month', () => {
		expect(monthKey(new Date(2026, 5, 5))).toBe('2026-06');
		expect(monthKey(new Date(2026, 0, 1))).toBe('2026-01');
	});
});

describe('reportMonthLabel', () => {
	it('labels the month in Jalali with Persian digits', () => {
		// 5 June 2026 is 15 Khordad 1405.
		expect(reportMonthLabel(new Date(2026, 5, 5))).toBe('خرداد ۱۴۰۵');
	});

	it('rolls a Gregorian January date into the previous Jalali year', () => {
		// 1 Jan 2026 is 11 Dey 1404.
		expect(reportMonthLabel(new Date(2026, 0, 1))).toBe('دی ۱۴۰۴');
	});
});
