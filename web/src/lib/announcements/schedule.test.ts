import { describe, expect, it } from 'vitest';
import { apiScheduleFromLocal, localScheduleFromApi } from './schedule';

describe('announcement schedules', () => {
	it('converts a local schedule to an RFC3339 instant', () => {
		const value = apiScheduleFromLocal('2026-10-04', '14:35');
		expect(value).toMatch(/^2026-10-04T/);
		expect(new Date(value ?? '').getMinutes()).toBe(35);
	});

	it('round trips a schedule through local date and time fields', () => {
		const source = new Date(2026, 9, 4, 14, 35).toISOString();
		expect(localScheduleFromApi(source)).toEqual({ date: '2026-10-04', time: '14:35' });
	});

	it('clears optional schedules and rejects invalid values', () => {
		expect(apiScheduleFromLocal('', '')).toBeNull();
		expect(() => apiScheduleFromLocal('2026-02-31', '12:00')).toThrow(RangeError);
		expect(() => apiScheduleFromLocal('2026-10-04', '25:00')).toThrow(RangeError);
	});
});
