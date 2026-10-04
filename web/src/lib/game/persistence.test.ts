import { describe, expect, it } from 'vitest';
import { createCampaign } from './engine';
import {
	clearCampaign,
	gameSaveKey,
	loadCampaign,
	saveCampaign,
	type GameStorage
} from './persistence';

class MemoryStorage implements GameStorage {
	readonly values = new Map<string, string>();

	getItem(key: string): string | null {
		return this.values.get(key) ?? null;
	}

	setItem(key: string, value: string): void {
		this.values.set(key, value);
	}

	removeItem(key: string): void {
		this.values.delete(key);
	}
}

describe('campaign persistence', () => {
	it('uses a save key scoped to the user and game version', () => {
		expect(gameSaveKey('manager-1')).toBe('hamsa.strategy-game.v1:manager-1');
	});

	it('round-trips a valid campaign through a versioned save', () => {
		const storage = new MemoryStorage();
		const campaign = createCampaign(23);

		expect(saveCampaign(storage, 'game-save', campaign)).toBe(true);
		expect(loadCampaign(storage, 'game-save')).toEqual(campaign);
		expect(JSON.parse(storage.getItem('game-save') ?? '{}').version).toBe(1);
	});

	it('returns null when no saved campaign exists', () => {
		expect(loadCampaign(new MemoryStorage(), 'game-save')).toBeNull();
	});

	it('discards malformed JSON without throwing', () => {
		const storage = new MemoryStorage();
		storage.setItem('game-save', '{');

		expect(loadCampaign(storage, 'game-save')).toBeNull();
		expect(storage.getItem('game-save')).toBeNull();
	});

	it('discards unsupported save versions', () => {
		const storage = new MemoryStorage();
		storage.setItem('game-save', JSON.stringify({ version: 2, campaign: createCampaign() }));

		expect(loadCampaign(storage, 'game-save')).toBeNull();
		expect(storage.getItem('game-save')).toBeNull();
	});

	it('rejects invalid campaign data and leaves existing saves untouched', () => {
		const storage = new MemoryStorage();
		storage.setItem('game-save', 'existing');

		expect(saveCampaign(storage, 'game-save', { ...createCampaign(), credits: -1 })).toBe(false);
		expect(storage.getItem('game-save')).toBe('existing');
	});

	it('removes only the requested game save', () => {
		const storage = new MemoryStorage();
		storage.setItem('game-save', 'campaign');
		storage.setItem('session-data', 'session');

		expect(clearCampaign(storage, 'game-save')).toBe(true);
		expect(storage.getItem('game-save')).toBeNull();
		expect(storage.getItem('session-data')).toBe('session');
	});

	it('returns null when browser storage throws while reading', () => {
		const storage: GameStorage = {
			getItem: () => {
				throw new Error('unavailable');
			},
			setItem: () => undefined,
			removeItem: () => undefined
		};

		expect(loadCampaign(storage, 'game-save')).toBeNull();
	});
});
