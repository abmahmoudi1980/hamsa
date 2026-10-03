import { get, patch, post } from '#lib/api/http';
import type { Person } from './persons';

export type Relationship = 'owner' | 'tenant' | 'non_resident_owner';

/** Dated person↔unit link; rows are end-dated, never deleted (FR-007). */
export interface Occupancy {
	id: string;
	unit_id: string;
	person_id: string;
	relationship: Relationship | string;
	start_date: string;
	end_date: string | null;
	is_active: boolean;
	created_at: string;
	person?: Person;
}

export interface OccupancyInput {
	person_id: string;
	relationship: Relationship;
	start_date: string;
	end_date?: string;
}

export function listOccupancies(unitId: string): Promise<{ items: Occupancy[] }> {
	return get<{ items: Occupancy[] }>(`/units/${unitId}/occupancies`);
}

export function addOccupancy(unitId: string, input: OccupancyInput): Promise<Occupancy> {
	return post<Occupancy>(`/units/${unitId}/occupancies`, input);
}

/** End-date an occupancy (preserves history). */
export function endOccupancy(id: string, endDate: string): Promise<Occupancy> {
	return patch<Occupancy>(`/occupancies/${id}`, { end_date: endDate });
}

/** One independent occupant-count data point (FR-008/BR-04). */
export interface OccupantCount {
	id: string;
	unit_id: string;
	occupant_count: number;
	effective_from: string;
	created_at: string;
}

export function listOccupantCounts(unitId: string): Promise<{ items: OccupantCount[] }> {
	return get<{ items: OccupantCount[] }>(`/units/${unitId}/occupant-count`);
}

export function recordOccupantCount(
	unitId: string,
	input: { occupant_count: number; effective_from: string }
): Promise<OccupantCount> {
	return post<OccupantCount>(`/units/${unitId}/occupant-count`, input);
}
