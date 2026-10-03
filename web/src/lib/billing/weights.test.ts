import { describe, expect, it } from 'vitest';
import {
	addWeightRow,
	removeWeightRow,
	weightsProblem,
	weightSum,
	type WeightRow
} from './weights';

const row = (method: WeightRow['method'], weight: string): WeightRow => ({ method, weight });

describe('weightSum', () => {
	it('reads Persian and Latin digits together', () => {
		expect(weightSum([row('equal', '۵۰'), row('per_area', '50')])).toBe(100);
	});

	it('is NaN when a row is blank', () => {
		expect(Number.isNaN(weightSum([row('equal', '')]))).toBe(true);
	});
});

describe('weightsProblem', () => {
	it('rejects an empty editor', () => {
		expect(weightsProblem([])).toBe('empty');
	});

	it('rejects a blank or zero weight', () => {
		expect(weightsProblem([row('equal', ''), row('per_area', '100')])).toBe('non_positive');
		expect(weightsProblem([row('equal', '0'), row('per_area', '100')])).toBe('non_positive');
	});

	it('rejects weights that do not sum to 100', () => {
		expect(weightsProblem([row('equal', '60'), row('per_area', '30')])).toBe('sum');
	});

	it('accepts positive weights summing to 100', () => {
		expect(weightsProblem([row('equal', '60'), row('per_area', '40')])).toBeNull();
	});
});

describe('row editing', () => {
	it('adds a blank row up to the method cap', () => {
		let rows: WeightRow[] = [];
		for (let i = 0; i < 6; i++) rows = addWeightRow(rows);
		expect(rows).toHaveLength(4);
	});

	it('removes the row at the given index', () => {
		const rows = [row('equal', '50'), row('per_area', '50')];
		expect(removeWeightRow(rows, 0)).toEqual([row('per_area', '50')]);
	});
});
