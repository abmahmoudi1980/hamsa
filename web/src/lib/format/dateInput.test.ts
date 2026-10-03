import { describe, expect, it } from 'vitest';
import { formatJalaliInput, parseJalaliInput } from './dateInput';

describe('parseJalaliInput', () => {
	it('parses a Persian-digit, slash-separated date to ISO', () => {
		expect(parseJalaliInput('۱۴۰۳/۰۵/۱۲')).toBe('2024-08-02');
	});

	it('accepts Latin digits and other separators', () => {
		expect(parseJalaliInput('1403/05/12')).toBe('2024-08-02');
		expect(parseJalaliInput('1403-5-12')).toBe('2024-08-02');
		expect(parseJalaliInput('1403.05.12')).toBe('2024-08-02');
	});

	it('accepts the compact eight-digit form', () => {
		expect(parseJalaliInput('14030512')).toBe('2024-08-02');
		expect(parseJalaliInput('۱۴۰۳۰۵۱۲')).toBe('2024-08-02');
	});

	it('rejects an impossible month or day', () => {
		expect(parseJalaliInput('1403/13/01')).toBeNull();
		expect(parseJalaliInput('1403/00/10')).toBeNull();
		expect(parseJalaliInput('1403/05/32')).toBeNull();
	});

	it('returns null for empty, partial or out-of-range input', () => {
		expect(parseJalaliInput('')).toBeNull();
		expect(parseJalaliInput('1403/05')).toBeNull();
		expect(parseJalaliInput('4000/01/01')).toBeNull();
	});
});

describe('formatJalaliInput', () => {
	it('renders an ISO date with Persian digits', () => {
		expect(formatJalaliInput('2024-08-02')).toBe('۱۴۰۳/۰۵/۱۲');
	});

	it('returns an empty string for a missing or malformed date', () => {
		expect(formatJalaliInput('')).toBe('');
		expect(formatJalaliInput('not-a-date')).toBe('');
	});
});
