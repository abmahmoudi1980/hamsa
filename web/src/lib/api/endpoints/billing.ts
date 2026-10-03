/**
 * Billing endpoints (contracts/api.md "Billing — periods, cost items,
 * calculation"). Money is a `Toman` string on the wire; a `combined` cost item
 * carries `combo_weights` summing to 100, and `specific_units` carries `unit_ids`.
 */

import { del, get, patch, post, put } from '#lib/api/http';
import type { Toman } from '#lib/format/money';

export type PeriodStatus = 'draft' | 'calculated' | 'issued' | 'closed';

export interface BillingPeriod {
	id: string;
	building_id: string;
	title: string;
	/** Gregorian ISO `YYYY-MM-DD`. */
	start_date: string;
	end_date: string;
	due_date: string;
	late_fee_type: string;
	late_fee_value: number;
	status: PeriodStatus | string;
	calculated_at?: string;
	issued_at?: string;
	closed_at?: string;
	created_at: string;
	updated_at: string;
}

export interface PeriodInput {
	title: string;
	start_date: string;
	end_date: string;
	due_date: string;
	late_fee_type?: string;
	late_fee_value?: number;
}

export type CalcMethod =
	'equal' | 'per_occupant' | 'per_area' | 'fixed' | 'specific_units' | 'combined';

export interface ComboWeight {
	method: string;
	weight: number;
	fixed_amount_per_unit?: Toman;
}

export interface CostItem {
	id: string;
	period_id: string;
	title: string;
	total_amount: Toman;
	method: string;
	fixed_amount_per_unit?: Toman;
	combo_weights?: ComboWeight[];
	include_vacant: boolean;
	unit_ids?: string[];
	created_at: string;
	updated_at: string;
}

export interface CostItemInput {
	title: string;
	total_amount?: Toman;
	method: CalcMethod;
	fixed_amount_per_unit?: Toman;
	combo_weights?: ComboWeight[];
	include_vacant?: boolean;
	unit_ids?: string[];
}

export interface PreviewShare {
	unit_id: string;
	unit_number?: string;
	/** Exact rational share, `NUMERIC(20,4)` on the wire (e.g. `"1000000.0000"`). */
	exact_share: string;
	rounded_share: Toman;
	inputs_snapshot: Record<string, unknown>;
}

export interface PreviewItem extends CostItem {
	/** For `fixed`, Σ per-unit amount rather than the nominal item total. */
	total_used: Toman;
	shares: PreviewShare[];
	reconciled: boolean;
}

export interface PreviewInvoice {
	id: string;
	invoice_number: string;
	unit_id: string;
	unit_number?: string;
	base_amount: Toman;
	prior_debt: Toman;
	late_fee_amount: Toman;
	credit_amount: Toman;
	final_amount: Toman;
	status: string;
	due_date?: string;
}

export interface Preview {
	period: BillingPeriod;
	items: PreviewItem[];
	invoices: PreviewInvoice[];
	reconciled: boolean;
}

export interface IssueResult {
	items: PreviewInvoice[];
	total: number;
}

async function items<T>(path: string): Promise<T[]> {
	const response = await get<{ items: T[] }>(path);
	return response.items;
}

// --- periods -------------------------------------------------------------------

export function listPeriods(buildingId: string): Promise<BillingPeriod[]> {
	return items<BillingPeriod>(`/buildings/${buildingId}/periods`);
}

export function createPeriod(buildingId: string, input: PeriodInput): Promise<BillingPeriod> {
	return post<BillingPeriod>(`/buildings/${buildingId}/periods`, input);
}

export function getPeriod(id: string): Promise<BillingPeriod> {
	return get<BillingPeriod>(`/periods/${id}`);
}

/** Only a `draft` period is editable; the server answers `409` otherwise. */
export function updatePeriod(id: string, input: PeriodInput): Promise<BillingPeriod> {
	return patch<BillingPeriod>(`/periods/${id}`, input);
}

// --- cost items ------------------------------------------------------------------

export function listCostItems(periodId: string): Promise<CostItem[]> {
	return items<CostItem>(`/periods/${periodId}/cost-items`);
}

export function createCostItem(periodId: string, input: CostItemInput): Promise<CostItem> {
	return post<CostItem>(`/periods/${periodId}/cost-items`, input);
}

export function updateCostItem(id: string, input: CostItemInput): Promise<CostItem> {
	return put<CostItem>(`/cost-items/${id}`, input);
}

export function deleteCostItem(id: string): Promise<null> {
	return del<null>(`/cost-items/${id}`);
}

// --- calculation and lifecycle ---------------------------------------------------

/** Runs the charge engine and refreshes drafts; returns the preview payload. */
export function calculatePeriod(periodId: string): Promise<Preview> {
	return post<Preview>(`/periods/${periodId}/calculate`);
}

export function getPreview(periodId: string): Promise<Preview> {
	return get<Preview>(`/periods/${periodId}/preview`);
}

export function reopenPeriod(periodId: string): Promise<BillingPeriod> {
	return post<BillingPeriod>(`/periods/${periodId}/reopen`);
}

export function issuePeriod(periodId: string): Promise<IssueResult> {
	return post<IssueResult>(`/periods/${periodId}/issue`);
}

export function closePeriod(periodId: string): Promise<BillingPeriod> {
	return post<BillingPeriod>(`/periods/${periodId}/close`);
}
