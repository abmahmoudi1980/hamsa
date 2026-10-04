import { describe, expect, it } from 'vitest';
import {
	GAME_EVENTS,
	canAffordAction,
	canAffordEventChoice,
	chooseAction,
	createCampaign,
	resolveEventChoice,
	scoreCampaign
} from './engine';
import { CAMPAIGN_MONTHS } from './model';

describe('campaign engine', () => {
	it('creates a new campaign with the expected starting resources and metrics', () => {
		const campaign = createCampaign();

		expect(campaign).toMatchObject({
			month: 1,
			credits: 55,
			condition: 60,
			satisfaction: 60,
			efficiency: 60,
			phase: 'action',
			history: []
		});
	});

	it('does not apply an action the campaign cannot afford', () => {
		const campaign = createCampaign();
		const result = chooseAction({ ...campaign, credits: 10 }, 'elevator');

		expect(canAffordAction({ ...campaign, credits: 10 }, 'elevator')).toBe(false);
		expect(result).toEqual({
			ok: false,
			state: { ...campaign, credits: 10 },
			reason: 'unaffordable'
		});
	});

	it('applies a selected action and exposes a deterministic event', () => {
		const result = chooseAction(createCampaign(1), 'maintenance');

		expect(result.ok).toBe(true);
		if (!result.ok) return;
		expect(result.state).toMatchObject({
			credits: 37,
			condition: 72,
			satisfaction: 63,
			phase: 'event',
			selectedActionId: 'maintenance',
			currentEventId: 'waterLeak'
		});
	});

	it('rejects event choices that belong to a different event', () => {
		const action = chooseAction(createCampaign(1), 'maintenance');
		expect(action.ok).toBe(true);
		if (!action.ok) return;

		expect(canAffordEventChoice(action.state, 'urgentLiftRepair')).toBe(false);
		expect(resolveEventChoice(action.state, 'urgentLiftRepair')).toEqual({
			ok: false,
			state: action.state,
			reason: 'invalid-choice'
		});
	});

	it('resolves an event, records the month, and grants income for the next month', () => {
		const action = chooseAction(createCampaign(1), 'maintenance');
		expect(action.ok).toBe(true);
		if (!action.ok) return;

		const result = resolveEventChoice(action.state, 'repairLeak');
		expect(result.ok).toBe(true);
		if (!result.ok) return;

		expect(result.state).toMatchObject({
			month: 2,
			credits: 54,
			condition: 84,
			satisfaction: 64,
			phase: 'action',
			history: [{ month: 1, actionId: 'maintenance', eventId: 'waterLeak', choiceId: 'repairLeak' }]
		});
	});

	it('completes after 12 months and returns a bounded score and outcome', () => {
		let campaign = createCampaign(7);
		while (campaign.phase !== 'complete') {
			if (campaign.phase === 'action') {
				const action = chooseAction(campaign, 'maintenance');
				expect(action.ok).toBe(true);
				if (!action.ok) break;
				campaign = action.state;
				continue;
			}

			const event = GAME_EVENTS.find((item) => item.id === campaign.currentEventId);
			const choice = event?.choices.find((item) => item.cost === 0);
			expect(choice).toBeDefined();
			if (!choice) break;
			const result = resolveEventChoice(campaign, choice.id);
			expect(result.ok).toBe(true);
			if (!result.ok) break;
			campaign = result.state;
		}

		expect(campaign.phase).toBe('complete');
		expect(campaign.history).toHaveLength(CAMPAIGN_MONTHS);
		expect(scoreCampaign(campaign).score).toBeGreaterThanOrEqual(0);
		expect(scoreCampaign(campaign).score).toBeLessThanOrEqual(100);
	});

	it('clamps metrics and returns wrong-phase errors without changing state', () => {
		const campaign = { ...createCampaign(), condition: 98 };
		const action = chooseAction(campaign, 'maintenance');
		expect(action.ok).toBe(true);
		if (!action.ok) return;
		expect(action.state.condition).toBe(100);
		expect(chooseAction(action.state, 'garden')).toEqual({
			ok: false,
			state: action.state,
			reason: 'wrong-phase'
		});
	});
});
