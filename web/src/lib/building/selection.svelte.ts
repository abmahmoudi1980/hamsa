import { readActiveBuildingId, writeActiveBuildingId } from '#lib/config';

export const buildingSelection = $state({ id: readActiveBuildingId() });

export function selectBuilding(id: string | null): void {
	buildingSelection.id = id;
	writeActiveBuildingId(id);
}
