import { del, get, post, put } from '#lib/api/http';

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

export function getBuilding(id: string): Promise<Building> {
	return get<Building>(`/buildings/${id}`);
}

export function updateBuilding(id: string, input: BuildingInput): Promise<Building> {
	return put<Building>(`/buildings/${id}`, input);
}

export function deleteBuilding(id: string): Promise<null> {
	return del<null>(`/buildings/${id}`);
}

/** One manager's grant on a building (002-multi-manager-support). */
export interface BuildingManager {
	user_id: string;
	phone: string;
	name: string;
	role: string;
	granted_at: string;
}

export async function listManagers(buildingId: string): Promise<BuildingManager[]> {
	const response = await get<{ items: BuildingManager[] }>(`/buildings/${buildingId}/managers`);
	return response.items;
}

export function grantManager(buildingId: string, phone: string): Promise<BuildingManager> {
	return post<BuildingManager>(`/buildings/${buildingId}/managers`, { phone });
}

export function revokeManager(buildingId: string, userId: string): Promise<null> {
	return del<null>(`/buildings/${buildingId}/managers/${userId}`);
}
