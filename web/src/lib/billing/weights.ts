/**
 * Combined-method weight editor rules.
 *
 * The backend rejects a `combined` cost item unless every weight is a positive
 * integer and the weights sum to exactly 100 (`ErrBadWeights`). Surfacing the
 * same rule live keeps the manager from submitting a request that returns 400,
 * and the `specific_units` method is intentionally excluded from a combo: the
 * engine does not carry sub-selections into `calcCombined`.
 */

import { latinDigitsOnly } from '#lib/format/digits';

export const COMBO_METHODS = ['equal', 'per_occupant', 'per_area', 'fixed'] as const;
export type ComboMethod = (typeof COMBO_METHODS)[number];

export interface WeightRow {
	method: ComboMethod;
	weight: string;
}

export type WeightsProblem = 'empty' | 'non_positive' | 'sum';

/** Sum of the weights, or NaN when a row is blank. */
export function weightSum(rows: readonly WeightRow[]): number {
	let sum = 0;
	for (const row of rows) {
		const digits = latinDigitsOnly(row.weight);
		if (digits === '') return Number.NaN;
		sum += Number(digits);
	}
	return sum;
}

/** `null` when the rows are valid; otherwise the reason they are not. */
export function weightsProblem(rows: readonly WeightRow[]): WeightsProblem | null {
	if (rows.length === 0) return 'empty';
	let sum = 0;
	for (const row of rows) {
		const digits = latinDigitsOnly(row.weight);
		if (digits === '') return 'non_positive';
		const value = Number(digits);
		if (value <= 0) return 'non_positive';
		sum += value;
	}
	return sum === 100 ? null : 'sum';
}

/** Returns a fresh copy with a new blank row, or the same rows at the cap. */
export function addWeightRow(rows: readonly WeightRow[]): WeightRow[] {
	if (rows.length >= COMBO_METHODS.length) return [...rows];
	return [...rows, { method: 'equal', weight: '' }];
}

export function removeWeightRow(rows: readonly WeightRow[], index: number): WeightRow[] {
	return rows.filter((_, i) => i !== index);
}
