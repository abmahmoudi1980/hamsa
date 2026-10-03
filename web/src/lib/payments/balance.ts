/**
 * Money-movement arithmetic (Phase 6).
 *
 * The server is the authority for every stored amount; these helpers exist so
 * the client can (a) display the §9 balance formula with the exact components
 * the API returned and prove it reconciles, and (b) derive an invoice's
 * outstanding amount from its frozen final amount plus its adjustment notes.
 * All arithmetic goes through BigInt `Toman` helpers — never a float.
 */

import type { Invoice, InvoiceAdjustment } from '#lib/api/endpoints/invoices';
import {
	addToman,
	positiveDifference,
	subtractToman,
	sumToman,
	type Toman
} from '#lib/format/money';

/** The five components that feed the §9 balance formula. */
export interface BalanceComponents {
	prior_debt: Toman;
	current_invoice_amount: Toman;
	late_fee_total: Toman;
	credit: Toman;
	paid_total: Toman;
}

/**
 * §9: `balance = prior_debt + current + late_fee − paid − credit`, floored at
 * zero (a negative result would mean unrecorded surplus, which the server
 * keeps in `credit_asset` instead).
 */
export function balanceFromComponents(components: BalanceComponents): Toman {
	const charges = addToman(
		components.prior_debt,
		components.current_invoice_amount,
		components.late_fee_total
	);
	const reductions = addToman(components.paid_total, components.credit);
	return positiveDifference(charges, reductions);
}

/** Sums the debit and credit notes of an invoice (missing notes are zero). */
export function adjustmentSums(adjustments: readonly InvoiceAdjustment[] = []): {
	debit: Toman;
	credit: Toman;
} {
	return {
		debit: sumToman(adjustments.filter((a) => a.kind === 'debit').map((a) => a.amount)),
		credit: sumToman(adjustments.filter((a) => a.kind === 'credit').map((a) => a.amount))
	};
}

/** The corrected payable amount: frozen final amount + debit notes − credit notes. */
export function invoiceEffective(invoice: Invoice): Toman {
	const { debit, credit } = adjustmentSums(invoice.adjustments);
	return subtractToman(addToman(invoice.final_amount, debit), credit);
}

/** What is still owed on an invoice, floored at zero. */
export function invoiceOutstanding(invoice: Invoice): Toman {
	return positiveDifference(invoiceEffective(invoice), invoice.paid_amount);
}

/** True while the invoice can still receive money (not settled, not cancelled). */
export function isPayable(invoice: Invoice): boolean {
	if (invoice.status === 'paid' || invoice.status === 'cancelled') return false;
	return invoiceOutstanding(invoice) !== '0';
}

/** True when the displayed components reproduce the server's stored balance. */
export function balanceReconciles(balance: BalanceComponents & { balance: Toman }): boolean {
	return balanceFromComponents(balance) === balance.balance;
}
