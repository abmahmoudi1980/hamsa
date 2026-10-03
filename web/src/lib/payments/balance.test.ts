import { describe, expect, it } from 'vitest';
import type { Invoice, InvoiceAdjustment } from '#lib/api/endpoints/invoices';
import type { Toman } from '#lib/format/money';
import {
	adjustmentSums,
	balanceFromComponents,
	balanceReconciles,
	invoiceEffective,
	invoiceOutstanding,
	isPayable,
	type BalanceComponents
} from './balance';

/** Test inputs are plain strings; the branded Toman type is a runtime no-op. */
const comp = (fields: Record<string, string>) => fields as unknown as BalanceComponents;
const recon = (fields: Record<string, string>) =>
	fields as unknown as BalanceComponents & { balance: Toman };

function adjustment(kind: string, amount: string): InvoiceAdjustment {
	return { kind, amount } as unknown as InvoiceAdjustment;
}

function invoice(overrides: Record<string, unknown> = {}): Invoice {
	return {
		final_amount: '1000000',
		paid_amount: '0',
		status: 'unpaid',
		...overrides
	} as unknown as Invoice;
}

describe('balanceFromComponents', () => {
	it('applies the §9 formula exactly', () => {
		expect(
			balanceFromComponents(
				comp({
					prior_debt: '500000',
					current_invoice_amount: '2000000',
					late_fee_total: '100000',
					credit: '0',
					paid_total: '0'
				})
			)
		).toBe('2600000');
	});

	it('subtracts payments and embedded credit', () => {
		expect(
			balanceFromComponents(
				comp({
					prior_debt: '0',
					current_invoice_amount: '1000000',
					late_fee_total: '0',
					credit: '200000',
					paid_total: '300000'
				})
			)
		).toBe('500000');
	});

	it('floors an overpaid unit at zero (surplus lives in credit_asset)', () => {
		expect(
			balanceFromComponents(
				comp({
					prior_debt: '0',
					current_invoice_amount: '100',
					late_fee_total: '0',
					credit: '0',
					paid_total: '500'
				})
			)
		).toBe('0');
	});

	it('stays exact beyond Number.MAX_SAFE_INTEGER', () => {
		expect(
			balanceFromComponents(
				comp({
					prior_debt: '9007199254740993',
					current_invoice_amount: '9007199254740993',
					late_fee_total: '0',
					credit: '0',
					paid_total: '0'
				})
			)
		).toBe('18014398509481986');
	});
});

describe('adjustmentSums', () => {
	it('is zero for an invoice with no notes', () => {
		expect(adjustmentSums([])).toEqual({ debit: '0', credit: '0' });
	});

	it('sums each kind independently', () => {
		expect(
			adjustmentSums([
				adjustment('debit', '100'),
				adjustment('debit', '50'),
				adjustment('credit', '30')
			])
		).toEqual({ debit: '150', credit: '30' });
	});
});

describe('invoiceEffective / invoiceOutstanding', () => {
	it('is the final amount when there are no adjustments', () => {
		expect(invoiceEffective(invoice())).toBe('1000000');
	});

	it('adds debit notes and subtracts credit notes', () => {
		expect(
			invoiceEffective(
				invoice({ adjustments: [adjustment('debit', '100'), adjustment('credit', '40')] })
			)
		).toBe('1000060');
	});

	it('subtracts the paid amount from the effective total', () => {
		expect(invoiceOutstanding(invoice({ paid_amount: '250000' }))).toBe('750000');
	});

	it('accounts for adjustments before the paid amount', () => {
		expect(
			invoiceOutstanding(
				invoice({ paid_amount: '1000000', adjustments: [adjustment('debit', '100')] })
			)
		).toBe('100');
	});

	it('floors at zero when the invoice is overpaid', () => {
		expect(invoiceOutstanding(invoice({ paid_amount: '5000000' }))).toBe('0');
	});
});

describe('isPayable', () => {
	it('is true for an unpaid invoice with an outstanding amount', () => {
		expect(isPayable(invoice())).toBe(true);
	});

	it('is false once settled or cancelled', () => {
		expect(isPayable(invoice({ status: 'paid' }))).toBe(false);
		expect(isPayable(invoice({ status: 'cancelled' }))).toBe(false);
	});

	it('is false for a fully paid invoice even if the status lags', () => {
		expect(isPayable(invoice({ paid_amount: '1000000' }))).toBe(false);
	});
});

describe('balanceReconciles', () => {
	it('is true when the components reproduce the stored balance', () => {
		expect(
			balanceReconciles(
				recon({
					prior_debt: '100',
					current_invoice_amount: '200',
					late_fee_total: '0',
					credit: '0',
					paid_total: '50',
					balance: '250'
				})
			)
		).toBe(true);
	});

	it('is false when the stored balance drifts from the formula', () => {
		expect(
			balanceReconciles(
				recon({
					prior_debt: '100',
					current_invoice_amount: '200',
					late_fee_total: '0',
					credit: '0',
					paid_total: '50',
					balance: '251'
				})
			)
		).toBe(false);
	});
});
