import { get, patch, post } from '#lib/api/http';
import type { Page } from './shared';

export const MAINTENANCE_CATEGORIES = [
	'elevator',
	'utilities',
	'electrical',
	'water',
	'cleaning',
	'common_area',
	'parking',
	'other'
] as const;

export const MAINTENANCE_PRIORITIES = ['normal', 'important', 'urgent'] as const;

export const MAINTENANCE_STATUSES = [
	'new',
	'under_review',
	'in_progress',
	'done',
	'closed'
] as const;

export interface MaintenanceRequest {
	id: string;
	building_id: string;
	unit_id?: string;
	submitted_by: string;
	title: string;
	category: string;
	description?: string;
	location?: string;
	photo_file?: string;
	priority: string;
	status: string;
	assignee_person_id?: string;
	recorded_cost?: number;
	notes?: string;
	created_at: string;
	updated_at: string;
	closed_at?: string;
}

export interface MaintenanceInput {
	title: string;
	category: string;
	description?: string | null;
	location?: string | null;
	photo_file_id?: string | null;
	priority: string;
}

export interface MaintenanceUpdate {
	status?: string;
	assignee_person_id?: string | null;
	recorded_cost?: number | string | null;
	notes?: string | null;
}

export interface MaintenanceFilters {
	status?: string;
	priority?: string;
	category?: string;
	page?: number;
	page_size?: number;
}

export function listBuildingMaintenance(
	buildingId: string,
	filters: MaintenanceFilters = {}
): Promise<Page<MaintenanceRequest>> {
	return get<Page<MaintenanceRequest>>(`/buildings/${buildingId}/maintenance-requests`, {
		query: { ...filters }
	});
}

export function listMyMaintenance(page = 1, pageSize = 20): Promise<Page<MaintenanceRequest>> {
	return get<Page<MaintenanceRequest>>('/me/maintenance-requests', {
		query: { page, page_size: pageSize }
	});
}

export function getMaintenanceRequest(id: string): Promise<MaintenanceRequest> {
	return get<MaintenanceRequest>(`/maintenance-requests/${id}`);
}

export function getMyMaintenanceRequest(id: string): Promise<MaintenanceRequest> {
	return get<MaintenanceRequest>(`/me/maintenance-requests/${id}`);
}

export function createMaintenanceRequest(input: MaintenanceInput): Promise<MaintenanceRequest> {
	return post<MaintenanceRequest>('/me/maintenance-requests', input);
}

export function updateMaintenanceRequest(
	id: string,
	input: MaintenanceUpdate
): Promise<MaintenanceRequest> {
	return patch<MaintenanceRequest>(`/maintenance-requests/${id}`, input);
}
