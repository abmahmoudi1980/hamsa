import { describe, expect, it } from 'vitest';
import { latinDecimalOnly } from './digits';

describe('latinDecimalOnly', () => {
	it('normalizes Persian and Arabic-Indic digits to Latin', () => {
		expect(latinDecimalOnly('۲.۵')).toBe('2.5');
		expect(latinDecimalOnly('٣,٤')).toBe('34');
	});

	it('keeps at most one decimal point', () => {
		expect(latinDecimalOnly('2.5.3')).toBe('2.53');
	});

	it('drops letters, signs and separators', () => {
		expect(latinDecimalOnly('-12abc3')).toBe('123');
	});

	it('returns an empty string when there is no digit', () => {
		expect(latinDecimalOnly('abc')).toBe('');
	});
});
