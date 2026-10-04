/**
 * Expense and financial-report endpoints (contracts/api.md "Expenses &
 * Financial Report", FR-027).
 *
 * Money note: unlike invoices/payments (which marshal Toman as JSON strings),
 * the expense and report DTOs carry `int64` fields that Go marshals as JSON
 * **numbers**. `formatTomanWithUnit` accepts a number and validates it is a
 * safe integer, so the UI stays exact; expense amounts never approach
 * `Number.MAX_SAFE_INTEGER`. Input sends `amount` as a JSON number to match the
 * backend's `json.Number` field.
 */

import { del, get, post, put } from '#lib/api/http';
import type { Page } from './shared';

/** Canonical category tokens (backend expense.Categories; migration 0006). */
export const EXPENSE_CATEGORIES = [
	'water',
	'electricity',
	'gas',
	'elevator',
	'cleaning',
	'security',
	'repair',
	'insurance',
	'equipment',
	'other'
] as const;

export type ExpenseCategory = (typeof EXPENSE_CATEGORIES)[number];

export type ApprovalStatus = 'pending' | 'approved' | 'rejected';

export interface Expense {
	id: string;
	building_id: string;
	title: string;
	category: string;
	/** Integer Toman as a JSON number (see the module note). */
	amount: number;
	/** Gregorian ISO `YYYY-MM-DD`. */
	expense_date: string;
	description?: string;
	payer_person_id?: string;
	/**
	 * Storage path of the bound receipt: `<fileId>.<ext>` (see storage.Save), so
	 * the file id is the segment before the first dot. Resolve it through
	 * `#lib/files/fileCache`, never as a bare URL.
	 */
	receipt_file?: string;
	approval_status: string;
	created_by?: string;
	created_at: string;
	updated_at: string;
}

export interface ExpenseInput {
	title: string;
	category: string;
	/** Integer Toman, sent as a JSON number. */
	amount: number;
	expense_date: string;
	description?: string | null;
	payer_person_id?: string | null;
	/** Empty string clears a bound receipt on update; a UUID binds one. */
	receipt_file_id?: string | null;
	approval_status: string;
}

export interface ExpenseFilters {
	category?: string;
	approval?: string;
	/** Gregorian ISO `YYYY-MM-DD`. */
	from?: string;
	/** Gregorian ISO `YYYY-MM-DD`. */
	to?: string;
	page?: number;
	page_size?: number;
}

export interface FinancialReport {
	month: string;
	monthly_income: number;
	monthly_expense: number;
	net: number;
	total_debt: number;
	total_payments: number;
	total_expenses: number;
}

/** `GET /buildings/{id}/expenses` — filtered, paginated list. */
export function listExpenses(
	buildingId: string,
	filters: ExpenseFilters = {}
): Promise<Page<Expense>> {
	return get<Page<Expense>>(`/buildings/${buildingId}/expenses`, { query: { ...filters } });
}

/** `GET /expenses/{id}` — one non-deleted expense. */
export function getExpense(id: string): Promise<Expense> {
	return get<Expense>(`/expenses/${id}`);
}

/** `POST /buildings/{id}/expenses` — record an expense (audited). */
export function createExpense(buildingId: string, input: ExpenseInput): Promise<Expense> {
	return post<Expense>(`/buildings/${buildingId}/expenses`, input);
}

/** `PUT /expenses/{id}` — partial update, including the approval workflow. */
export function updateExpense(id: string, input: Partial<ExpenseInput>): Promise<Expense> {
	return put<Expense>(`/expenses/${id}`, input);
}

/** `DELETE /expenses/{id}` — soft-delete; the row survives for history. */
export function deleteExpense(id: string): Promise<{ deleted: boolean }> {
	return del<{ deleted: boolean }>(`/expenses/${id}`);
}

/** `GET /buildings/{id}/financial-report?month=YYYY-MM` — FR-027. */
export function getFinancialReport(buildingId: string, month: string): Promise<FinancialReport> {
	return get<FinancialReport>(`/buildings/${buildingId}/financial-report`, { query: { month } });
}
