import { get } from '#lib/api/http';
import type { Toman } from '#lib/format/money';

export interface DashboardAlert {
	kind: 'debtor_unit' | 'past_due_invoice' | 'open_request' | 'pending_expense' | string;
	severity: number;
	unit_id?: string;
	unit_number?: string;
	amount?: Toman;
	title?: string;
	ref_type?: string;
	ref_id?: string;
	created_at?: string;
}

export interface ManagerDashboard {
	building_id: string;
	unit_count: number;
	occupied_unit_count: number;
	debtor_unit_count: number;
	total_debt: Toman;
	month_income: Toman;
	month_expense: Toman;
	open_requests: number;
	pending_expenses: number;
	month: string;
	alerts: DashboardAlert[];
	quick_actions: Array<{ key: string; title: string; path: string }>;
}

export interface ResidentHome {
	payable_amount: Toman;
	unit_count: number;
	latest_invoice?: {
		id: string;
		invoice_number: string;
		unit_number?: string;
		period_title?: string;
		final_amount: Toman;
		due_date?: string;
		status: string;
	};
	open_request_count: number;
	latest_request?: {
		id: string;
		title: string;
		priority: string;
		status: string;
		created_at?: string;
	};
	latest_announcements: Array<{
		id: string;
		title: string;
		created_at?: string;
		is_read: boolean;
	}>;
	unread_announcement_count: number;
}

export function getManagerDashboard(buildingId: string): Promise<ManagerDashboard> {
	return get<ManagerDashboard>(`/buildings/${buildingId}/dashboard`);
}

export function getResidentHome(): Promise<ResidentHome> {
	return get<ResidentHome>('/me/home');
}
