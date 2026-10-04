import { z } from 'zod';
import {
	CAMPAIGN_MONTHS,
	GAME_SAVE_VERSION,
	METRIC_MAX,
	METRIC_MIN,
	type CampaignState
} from './model';

export interface GameStorage {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
	removeItem(key: string): void;
}

const actionIds = ['maintenance', 'insulation', 'garden', 'elevator'] as const;
const eventIds = ['waterLeak', 'energySpike', 'liftFailure', 'gardenRequest'] as const;
const choiceIds = [
	'repairLeak',
	'deferLeak',
	'installLighting',
	'reduceLighting',
	'urgentLiftRepair',
	'delayLiftRepair',
	'buildGarden',
	'postponeGarden'
] as const;

const campaignSchema = z
	.object({
		month: z.number().int().min(1).max(CAMPAIGN_MONTHS),
		credits: z.number().int().nonnegative(),
		condition: z.number().int().min(METRIC_MIN).max(METRIC_MAX),
		satisfaction: z.number().int().min(METRIC_MIN).max(METRIC_MAX),
		efficiency: z.number().int().min(METRIC_MIN).max(METRIC_MAX),
		seed: z.number().int().min(0).max(0xffffffff),
		phase: z.enum(['action', 'event', 'complete']),
		selectedActionId: z.enum(actionIds).nullable(),
		currentEventId: z.enum(eventIds).nullable(),
		history: z
			.array(
				z.object({
					month: z.number().int().min(1).max(CAMPAIGN_MONTHS),
					actionId: z.enum(actionIds),
					eventId: z.enum(eventIds),
					choiceId: z.enum(choiceIds)
				})
			)
			.max(CAMPAIGN_MONTHS)
	})
	.superRefine((campaign, context) => {
		const validActionPhase =
			campaign.phase !== 'action' ||
			(campaign.selectedActionId === null && campaign.currentEventId === null);
		const validEventPhase =
			campaign.phase !== 'event' ||
			(campaign.selectedActionId !== null && campaign.currentEventId !== null);
		const validCompletePhase =
			campaign.phase !== 'complete' ||
			(campaign.month === CAMPAIGN_MONTHS && campaign.history.length === CAMPAIGN_MONTHS);
		const expectedHistoryLength =
			campaign.phase === 'complete' ? CAMPAIGN_MONTHS : campaign.month - 1;

		if (!validActionPhase || !validEventPhase || !validCompletePhase) {
			context.addIssue({ code: 'custom', message: 'Invalid campaign phase state' });
		}
		if (campaign.history.length !== expectedHistoryLength) {
			context.addIssue({ code: 'custom', message: 'Invalid campaign history length' });
		}
	});

const envelopeSchema = z.object({
	version: z.number().int(),
	campaign: campaignSchema
});

export function gameSaveKey(userId: string): string {
	return `hamsa.strategy-game.v${GAME_SAVE_VERSION}:${userId}`;
}

export function loadCampaign(storage: GameStorage, key: string): CampaignState | null {
	let raw: string | null;
	try {
		raw = storage.getItem(key);
	} catch {
		return null;
	}
	if (raw === null) return null;

	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch {
		discardSave(storage, key);
		return null;
	}

	const envelope = envelopeSchema.safeParse(parsed);
	if (!envelope.success || envelope.data.version !== GAME_SAVE_VERSION) {
		discardSave(storage, key);
		return null;
	}
	return envelope.data.campaign;
}

export function saveCampaign(storage: GameStorage, key: string, campaign: CampaignState): boolean {
	const validated = campaignSchema.safeParse(campaign);
	if (!validated.success) return false;
	try {
		storage.setItem(key, JSON.stringify({ version: GAME_SAVE_VERSION, campaign: validated.data }));
		return true;
	} catch {
		return false;
	}
}

export function clearCampaign(storage: GameStorage, key: string): boolean {
	try {
		storage.removeItem(key);
		return true;
	} catch {
		return false;
	}
}

function discardSave(storage: GameStorage, key: string): void {
	try {
		storage.removeItem(key);
	} catch {
		return;
	}
}
