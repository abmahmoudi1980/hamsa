import {
	CAMPAIGN_MONTHS,
	METRIC_MAX,
	METRIC_MIN,
	MONTHLY_INCOME,
	STARTING_CREDITS,
	type CampaignState,
	type GameActionId,
	type GameEventChoiceId,
	type GameEventId
} from './model';

export interface GameAction {
	id: GameActionId;
	cost: number;
	condition: number;
	satisfaction: number;
	efficiency: number;
}

export interface GameEventChoice {
	id: GameEventChoiceId;
	cost: number;
	credits: number;
	condition: number;
	satisfaction: number;
	efficiency: number;
}

export interface GameEvent {
	id: GameEventId;
	choices: readonly GameEventChoice[];
}

export type CampaignScore = {
	score: number;
	outcome: 'excellent' | 'stable' | 'struggling';
};

export type TransitionResult =
	| { ok: true; state: CampaignState }
	| { ok: false; state: CampaignState; reason: 'wrong-phase' | 'unaffordable' | 'invalid-choice' };

export const GAME_ACTIONS: readonly GameAction[] = [
	{ id: 'maintenance', cost: 18, condition: 12, satisfaction: 3, efficiency: 0 },
	{ id: 'insulation', cost: 24, condition: 2, satisfaction: 0, efficiency: 14 },
	{ id: 'garden', cost: 20, condition: 0, satisfaction: 13, efficiency: -2 },
	{ id: 'elevator', cost: 32, condition: 15, satisfaction: 4, efficiency: 0 }
];

export const GAME_EVENTS: readonly GameEvent[] = [
	{
		id: 'waterLeak',
		choices: [
			{ id: 'repairLeak', cost: 18, credits: 0, condition: 12, satisfaction: 1, efficiency: 0 },
			{ id: 'deferLeak', cost: 0, credits: 0, condition: -10, satisfaction: -6, efficiency: 0 }
		]
	},
	{
		id: 'energySpike',
		choices: [
			{
				id: 'installLighting',
				cost: 15,
				credits: 0,
				condition: 0,
				satisfaction: 0,
				efficiency: 12
			},
			{ id: 'reduceLighting', cost: 0, credits: 0, condition: 0, satisfaction: -7, efficiency: 4 }
		]
	},
	{
		id: 'liftFailure',
		choices: [
			{
				id: 'urgentLiftRepair',
				cost: 24,
				credits: 0,
				condition: 14,
				satisfaction: 4,
				efficiency: 0
			},
			{
				id: 'delayLiftRepair',
				cost: 0,
				credits: 0,
				condition: -12,
				satisfaction: -9,
				efficiency: 0
			}
		]
	},
	{
		id: 'gardenRequest',
		choices: [
			{ id: 'buildGarden', cost: 20, credits: 0, condition: 0, satisfaction: 12, efficiency: -1 },
			{ id: 'postponeGarden', cost: 0, credits: 4, condition: 0, satisfaction: -3, efficiency: 0 }
		]
	}
];

const actionById = new Map(GAME_ACTIONS.map((action) => [action.id, action]));
const eventById = new Map(GAME_EVENTS.map((event) => [event.id, event]));

export function createCampaign(seed = 1): CampaignState {
	return {
		month: 1,
		credits: STARTING_CREDITS,
		condition: 60,
		satisfaction: 60,
		efficiency: 60,
		seed: normalizeSeed(seed),
		phase: 'action',
		selectedActionId: null,
		currentEventId: null,
		history: []
	};
}

export function canAffordAction(state: CampaignState, actionId: GameActionId): boolean {
	const action = actionById.get(actionId);
	return state.phase === 'action' && action !== undefined && action.cost <= state.credits;
}

export function canAffordEventChoice(state: CampaignState, choiceId: GameEventChoiceId): boolean {
	if (state.phase !== 'event' || state.currentEventId === null) return false;
	const event = eventById.get(state.currentEventId);
	const choice = event?.choices.find((item) => item.id === choiceId);
	return choice !== undefined && choice.cost <= state.credits;
}

export function chooseAction(state: CampaignState, actionId: GameActionId): TransitionResult {
	if (state.phase !== 'action') return { ok: false, state, reason: 'wrong-phase' };
	const action = actionById.get(actionId);
	if (!action) return { ok: false, state, reason: 'invalid-choice' };
	if (action.cost > state.credits) return { ok: false, state, reason: 'unaffordable' };

	const next = advanceSeed(state.seed);
	const eventIndex = next.seed % GAME_EVENTS.length;
	const event = GAME_EVENTS[eventIndex];
	if (!event) return { ok: false, state, reason: 'invalid-choice' };

	return {
		ok: true,
		state: {
			...applyMetrics(state, action),
			credits: state.credits - action.cost,
			seed: next.seed,
			phase: 'event',
			selectedActionId: action.id,
			currentEventId: event.id
		}
	};
}

export function resolveEventChoice(
	state: CampaignState,
	choiceId: GameEventChoiceId
): TransitionResult {
	if (state.phase !== 'event' || state.currentEventId === null || state.selectedActionId === null) {
		return { ok: false, state, reason: 'wrong-phase' };
	}

	const event = eventById.get(state.currentEventId);
	const choice = event?.choices.find((item) => item.id === choiceId);
	if (!choice) return { ok: false, state, reason: 'invalid-choice' };
	if (choice.cost > state.credits) return { ok: false, state, reason: 'unaffordable' };

	const history = [
		...state.history,
		{
			month: state.month,
			actionId: state.selectedActionId,
			eventId: state.currentEventId,
			choiceId: choice.id
		}
	];
	const afterChoice = applyMetrics(state, choice);
	const baseState: CampaignState = {
		...afterChoice,
		credits: state.credits - choice.cost + choice.credits,
		history,
		selectedActionId: null,
		currentEventId: null
	};

	if (state.month === CAMPAIGN_MONTHS) {
		return { ok: true, state: { ...baseState, phase: 'complete' } };
	}

	return {
		ok: true,
		state: {
			...baseState,
			month: state.month + 1,
			credits: baseState.credits + MONTHLY_INCOME,
			phase: 'action'
		}
	};
}

export function scoreCampaign(state: CampaignState): CampaignScore {
	const score = Math.round(
		state.condition * 0.4 + state.satisfaction * 0.35 + state.efficiency * 0.25
	);
	const outcome = score >= 80 ? 'excellent' : score >= 60 ? 'stable' : 'struggling';
	return { score, outcome };
}

function applyMetrics(
	state: CampaignState,
	effect: Pick<GameAction, 'condition' | 'satisfaction' | 'efficiency'>
): CampaignState {
	return {
		...state,
		condition: clampMetric(state.condition + effect.condition),
		satisfaction: clampMetric(state.satisfaction + effect.satisfaction),
		efficiency: clampMetric(state.efficiency + effect.efficiency)
	};
}

function clampMetric(value: number): number {
	return Math.max(METRIC_MIN, Math.min(METRIC_MAX, value));
}

function normalizeSeed(seed: number): number {
	return Number.isSafeInteger(seed) ? seed >>> 0 : 1;
}

function advanceSeed(seed: number): { seed: number } {
	return { seed: (Math.imul(seed, 1664525) + 1013904223) >>> 0 };
}
