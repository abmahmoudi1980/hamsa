export const GAME_SAVE_VERSION = 1;
export const CAMPAIGN_MONTHS = 12;
export const STARTING_CREDITS = 55;
export const MONTHLY_INCOME = 35;
export const METRIC_MIN = 0;
export const METRIC_MAX = 100;

export type GameActionId = 'maintenance' | 'insulation' | 'garden' | 'elevator';

export type GameEventId = 'waterLeak' | 'energySpike' | 'liftFailure' | 'gardenRequest';

export type GameEventChoiceId =
	| 'repairLeak'
	| 'deferLeak'
	| 'installLighting'
	| 'reduceLighting'
	| 'urgentLiftRepair'
	| 'delayLiftRepair'
	| 'buildGarden'
	| 'postponeGarden';

export type CampaignPhase = 'action' | 'event' | 'complete';

export interface CampaignState {
	month: number;
	credits: number;
	condition: number;
	satisfaction: number;
	efficiency: number;
	seed: number;
	phase: CampaignPhase;
	selectedActionId: GameActionId | null;
	currentEventId: GameEventId | null;
	history: CampaignHistoryEntry[];
}

export interface CampaignHistoryEntry {
	month: number;
	actionId: GameActionId;
	eventId: GameEventId;
	choiceId: GameEventChoiceId;
}
