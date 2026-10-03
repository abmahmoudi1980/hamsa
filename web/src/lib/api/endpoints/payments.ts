/**
 * Payment and balance endpoints (contracts/api.md "Payments & Balances").
 *
 * The ledger returns the standard pagination envelope (F4) plus a deprecated
 * `payments` alias the pre-003 Flutter client read; the web reads `items`.
 */

import { get } from '#lib/api/http';
import type { Toman } from '#lib/format/money';
import type { Page } from './shared';

export type PaymentMethod = 'manual' | 'gateway' | 'transfer';
export type PaymentStatus = 'recorded' | 'verified' | 'failed' | 'reversed';

export interface Payment {
	id: string;
	building_id: string;
	unit_id: string;
	invoice_id?: string;
	method: string;
	amount: Toman;
	/** Gregorian ISO `YYYY-MM-DD`. */
	paid_at: string;
	tracking_number?: string;
	gateway?: string;
	authority?: string;
	recorded_by?: string;
	status: string;
	created_at: string;
	updated_at: string;
	/** Denormalized display field from the ledger query. */
	unit_number?: string;
}

export interface PaymentFilters {
	unit_id?: string;
	method?: string;
	/** Gregorian ISO `YYYY-MM-DD`. */
	from?: string;
	to?: string;
	page?: number;
	page_size?: number;
}

/** The unit's financial position (spec §9); all amounts are integer Toman strings. */
export interface UnitBalance {
	unit_id: string;
	prior_debt: Toman;
	current_invoice_amount: Toman;
	late_fee_total: Toman;
	credit: Toman;
	paid_total: Toman;
	/** Surplus money not yet consumed by an invoice — an asset, outside the formula. */
	credit_asset: Toman;
	balance: Toman;
	updated_at: string;
}

/** `GET /buildings/{id}/payments` — manager ledger with unit/method/date filters. */
export function listPayments(
	buildingId: string,
	filters: PaymentFilters = {}
): Promise<Page<Payment>> {
	return get<Page<Payment>>(`/buildings/${buildingId}/payments`, { query: { ...filters } });
}

/** `GET /me/payments` — the resident's own payment history. */
export function listMyPayments(filters: PaymentFilters = {}): Promise<Page<Payment>> {
	return get<Page<Payment>>('/me/payments', { query: { ...filters } });
}

/** `GET /units/{id}/balance` — current balance components for one unit. */
export function getUnitBalance(unitId: string): Promise<UnitBalance> {
	return get<UnitBalance>(`/units/${unitId}/balance`);
}
