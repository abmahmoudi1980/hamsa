import { del, get, post, put } from '#lib/api/http';
import type { Page } from './shared';

export interface Person {
	id: string;
	building_id: string;
	full_name: string;
	phone: string | null;
	national_id: string | null;
	created_at: string;
	updated_at: string;
}

export interface PersonInput {
	full_name: string;
	phone?: string;
	national_id?: string;
}

export interface PersonFilters {
	q?: string;
	page?: number;
	page_size?: number;
}

export function listPersons(
	buildingId: string,
	filters: PersonFilters = {}
): Promise<Page<Person>> {
	return get<Page<Person>>(`/buildings/${buildingId}/persons`, { query: { ...filters } });
}

export function createPerson(buildingId: string, input: PersonInput): Promise<Person> {
	return post<Person>(`/buildings/${buildingId}/persons`, input);
}

export function updatePerson(id: string, input: PersonInput): Promise<Person> {
	return put<Person>(`/persons/${id}`, input);
}

/** Soft-archive; occupancy history is retained (spec §21). */
export function archivePerson(id: string): Promise<null> {
	return del<null>(`/persons/${id}`);
}
