import { del, get, post, put } from '#lib/api/http';
import type { Page } from './shared';

export type UnitStatus = 'active' | 'vacant' | 'occupied' | 'inactive';

export interface Unit {
	id: string;
	building_id: string;
	number: string;
	block: string | null;
	floor: number;
	area_m2: number;
	parking_count: number;
	parking_numbers: string | null;
	storage_count: number;
	storage_numbers: string | null;
	status: UnitStatus | string;
	notes: string | null;
	created_at: string;
	updated_at: string;
}

export interface UnitInput {
	number: string;
	block?: string;
	floor?: number;
	area_m2: number;
	parking_count?: number;
	parking_numbers?: string;
	storage_count?: number;
	storage_numbers?: string;
	status?: UnitStatus;
	notes?: string;
}

export interface UnitFilters {
	q?: string;
	block?: string;
	floor?: string | number;
	status?: string;
	page?: number;
	page_size?: number;
}

/** `GET /buildings/{id}/units` — server-side `?q=&block=&floor=&status=` + paging. */
export function listUnits(buildingId: string, filters: UnitFilters = {}): Promise<Page<Unit>> {
	return get<Page<Unit>>(`/buildings/${buildingId}/units`, { query: { ...filters } });
}

export function getUnit(id: string): Promise<Unit> {
	return get<Unit>(`/units/${id}`);
}

export function createUnit(buildingId: string, input: UnitInput): Promise<Unit> {
	return post<Unit>(`/buildings/${buildingId}/units`, input);
}

export function updateUnit(id: string, input: UnitInput): Promise<Unit> {
	return put<Unit>(`/units/${id}`, input);
}

/** Soft-delete (archive); financial history is retained. */
export function archiveUnit(id: string): Promise<null> {
	return del<null>(`/units/${id}`);
}

export interface UnitHistoryEntry {
	id: number;
	user_id: string | null;
	action: string;
	before_value: unknown;
	after_value: unknown;
	created_at: string;
}

/** `GET /units/{id}/history` — append-only audit trail, newest first. */
export function getUnitHistory(id: string): Promise<{ items: UnitHistoryEntry[] }> {
	return get<{ items: UnitHistoryEntry[] }>(`/units/${id}/history`);
}
