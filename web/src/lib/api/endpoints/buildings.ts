import { get, post } from '#lib/api/http';

export interface Building {
	id: string;
	name: string;
	address: string | null;
	block_count: number;
	floor_count: number;
	unit_count: number;
	built_year: number | null;
	manager_phone: string | null;
	emergency_phone: string | null;
	notes: string | null;
	created_at?: string;
}

export interface BuildingInput {
	name: string;
	address?: string;
	block_count: number;
	floor_count: number;
	unit_count: number;
	built_year?: number;
	manager_phone?: string;
	emergency_phone?: string;
	notes?: string;
}

interface BuildingListResponse {
	items: Building[];
}

export async function listBuildings(): Promise<Building[]> {
	const response = await get<BuildingListResponse>('/buildings');
	return response.items;
}

export function createBuilding(input: BuildingInput): Promise<Building> {
	return post<Building>('/buildings', input);
}
