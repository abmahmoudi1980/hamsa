/**
 * Invoice endpoints (contracts/api.md "Invoices").
 *
 * An issued invoice is immutable: only `cancel`, `adjustments` and the
 * payment-driven `status`/`paid_amount` fields may change afterwards. Amounts
 * are `Toman` strings on the wire.
 */

import { get, post } from '#lib/api/http';
import type { Toman } from '#lib/format/money';
import type { Page } from './shared';
import type { Payment } from './payments';

export type InvoiceStatus = 'unpaid' | 'partial' | 'paid' | 'cancelled' | 'expired';
export type InvoiceItemKind = 'charge' | 'late_fee' | 'adjustment';
export type AdjustmentKind = 'debit' | 'credit';

/** One append-only displayable row of an invoice. */
export interface InvoiceItem {
	id: string;
	invoice_id: string;
	kind: InvoiceItemKind | string;
	title: string;
	cost_item_id?: string;
	amount: Toman;
	method?: string;
	adjustment_id?: string;
	created_at: string;
}

/** An explicit correction note (FR-017); original amounts stay untouched. */
export interface InvoiceAdjustment {
	id: string;
	invoice_id: string;
	kind: AdjustmentKind | string;
	amount: Toman;
	reason: string;
	created_by?: string;
	created_at: string;
}

export interface Invoice {
	id: string;
	invoice_number: string;
	building_id: string;
	period_id: string;
	unit_id: string;
	base_amount: Toman;
	prior_debt: Toman;
	late_fee_amount: Toman;
	credit_amount: Toman;
	final_amount: Toman;
	issue_date?: string;
	due_date?: string;
	status: InvoiceStatus | string;
	paid_amount: Toman;
	inputs_frozen_at?: string;
	created_at: string;
	updated_at: string;
	/** Populated only by the detail read. */
	items?: InvoiceItem[];
	adjustments?: InvoiceAdjustment[];
	/** Denormalized display fields from the list/detail queries. */
	unit_number?: string;
	period_title?: string;
}

export interface InvoiceFilters {
	period_id?: string;
	unit_id?: string;
	status?: string;
	page?: number;
	page_size?: number;
}

/** `GET /buildings/{id}/invoices` — manager list, filter + server-side paging. */
export function listInvoices(
	buildingId: string,
	filters: InvoiceFilters = {}
): Promise<Page<Invoice>> {
	return get<Page<Invoice>>(`/buildings/${buildingId}/invoices`, { query: { ...filters } });
}

/** `GET /invoices/{id}` — detail with items + adjustments (manager or owning resident). */
export function getInvoice(id: string): Promise<Invoice> {
	return get<Invoice>(`/invoices/${id}`);
}

/** `POST /invoices/{id}/cancel` — row preserved, audited (BR-10). */
export function cancelInvoice(id: string, reason: string): Promise<Invoice> {
	return post<Invoice>(`/invoices/${id}/cancel`, { reason });
}

export interface AdjustmentInput {
	kind: AdjustmentKind;
	amount: Toman;
	reason: string;
}

/** `POST /invoices/{id}/adjustments` — append a debit/credit note. */
export function addAdjustment(id: string, input: AdjustmentInput): Promise<InvoiceAdjustment> {
	return post<InvoiceAdjustment>(`/invoices/${id}/adjustments`, input);
}

export interface ManualPaymentInput {
	amount: Toman;
	/** Gregorian ISO `YYYY-MM-DD`; the server defaults to today when omitted. */
	paid_at?: string;
	tracking_number?: string;
}

/** `POST /invoices/{id}/payments` — manager records a manual (cash/transfer) payment. */
export function recordManualPayment(
	invoiceId: string,
	input: ManualPaymentInput
): Promise<Payment> {
	return post<Payment>(`/invoices/${invoiceId}/payments`, input);
}

export interface GatewayStartInput {
	/** Omitted → the invoice's full outstanding amount. */
	amount?: Toman;
	/** Same-origin result path the gateway callback 302s back to (F3). */
	return_path?: string;
}

export interface GatewayStart {
	payment_id: string;
	payment_url: string;
	amount: Toman;
	status: string;
}

/** `POST /invoices/{id}/pay` — resident/manager starts an online gateway payment. */
export function startGatewayPayment(
	invoiceId: string,
	input: GatewayStartInput = {}
): Promise<GatewayStart> {
	return post<GatewayStart>(`/invoices/${invoiceId}/pay`, input);
}

// --- resident scope ---------------------------------------------------------

/** `GET /me/invoices` — the resident's own units' invoices. */
export function listMyInvoices(filters: InvoiceFilters = {}): Promise<Page<Invoice>> {
	return get<Page<Invoice>>('/me/invoices', { query: { ...filters } });
}

/** `GET /me/invoices/{id}` — own invoice detail; a foreign id is a server-side 403. */
export function getMyInvoice(id: string): Promise<Invoice> {
	return get<Invoice>(`/me/invoices/${id}`);
}
