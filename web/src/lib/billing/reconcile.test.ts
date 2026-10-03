import { describe, expect, it } from 'vitest';
import { formatExactShare, reconcileItem, reconcilePreview, sharesSum } from './reconcile';

const share = (rounded: string) => ({ rounded_share: rounded });

describe('sharesSum', () => {
	it('sums Toman strings through BigInt, beyond Number.MAX_SAFE_INTEGER', () => {
		expect(sharesSum([share('9007199254740993'), share('9007199254740993')])).toBe(
			'18014398509481986'
		);
	});

	it('is zero for an item with no participants', () => {
		expect(sharesSum([])).toBe('0');
	});
});

describe('reconcileItem', () => {
	it('reports zero residual when the rounded shares equal total_used', () => {
		// 10,000,000 split evenly across three units — largest-remainder leaves
		// one unit with the extra Toman.
		const result = reconcileItem({
			total_used: '10000000',
			shares: [share('3333334'), share('3333333'), share('3333333')]
		});
		expect(result).toEqual({
			totalUsed: '10000000',
			roundedSum: '10000000',
			residual: '0',
			reconciled: true
		});
	});

	it('surfaces a signed residual when a share is off by one', () => {
		const result = reconcileItem({ total_used: '1000', shares: [share('999')] });
		expect(result.residual).toBe('-1');
		expect(result.reconciled).toBe(false);
	});

	it('reconciles a fixed item against total_used, not its nominal total', () => {
		const result = reconcileItem({
			total_used: '200000',
			shares: [share('100000'), share('100000')]
		});
		expect(result.reconciled).toBe(true);
	});
});

describe('reconcilePreview', () => {
	it('is reconciled only when every item is, and sums the residuals', () => {
		const result = reconcilePreview({
			items: [
				{ total_used: '10000000', shares: [share('5000000'), share('5000000')] },
				{ total_used: '8000000', shares: [share('8000000')] }
			]
		});
		expect(result.reconciled).toBe(true);
		expect(result.residual).toBe('0');
		expect(result.items).toHaveLength(2);
	});

	it('flags the period when a single item is short', () => {
		const result = reconcilePreview({
			items: [
				{ total_used: '1000', shares: [share('1000')] },
				{ total_used: '1000', shares: [share('999')] }
			]
		});
		expect(result.reconciled).toBe(false);
		expect(result.residual).toBe('-1');
	});
});

describe('formatExactShare', () => {
	it('drops trailing zeros and renders Persian digits', () => {
		expect(formatExactShare('1600000.0000')).toBe('۱۶۰۰۰۰۰');
		expect(formatExactShare('333333.3333')).toBe('۳۳۳۳۳۳.۳۳۳۳');
	});
});
