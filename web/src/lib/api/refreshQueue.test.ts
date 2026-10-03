import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	__resetRefreshQueue,
	ensureFreshToken,
	isRefreshing,
	refreshOnce,
	reportSessionExpired,
	setSessionExpiredHandler
} from './refreshQueue';
import {
	clearSession,
	getAccessToken,
	needsRefresh,
	saveRotatedTokens,
	saveSession
} from './tokenStore';

/**
 * The mandatory gate from the plan: a burst of 401s must trigger exactly ONE
 * refresh exchange.
 *
 * The failure this protects against is not theoretical — the server ROTATES the
 * refresh token, so N concurrent refreshes mean N-1 of them present an
 * already-consumed token, fail, clear the session and log the user out
 * mid-session. Mobile has the same guard in its QueuedInterceptor.
 */

beforeEach(() => {
	__resetRefreshQueue();
	clearSession();
});

afterEach(() => {
	vi.restoreAllMocks();
});

describe('single-flight refresh', () => {
	it('collapses concurrent refreshes into one exchange', async () => {
		let calls = 0;
		let release!: (value: string) => void;
		const gate = new Promise<string>((resolve) => {
			release = resolve;
		});

		const ensure = () => {
			calls += 1;
			return gate;
		};

		// Five requests all discover the token is missing at the same moment.
		const waiters = [
			refreshOnce(ensure),
			refreshOnce(ensure),
			refreshOnce(ensure),
			refreshOnce(ensure),
			refreshOnce(ensure)
		];

		expect(calls).toBe(1);
		expect(isRefreshing()).toBe(true);

		release('new-access-token');
		const tokens = await Promise.all(waiters);

		expect(calls).toBe(1);
		expect(tokens).toEqual(Array(5).fill('new-access-token'));
		expect(isRefreshing()).toBe(false);
	});

	it('lets a later refresh start once the previous one settled', async () => {
		let calls = 0;
		const ensure = async () => {
			calls += 1;
			return `token-${calls}`;
		};

		expect(await refreshOnce(ensure)).toBe('token-1');
		expect(await refreshOnce(ensure)).toBe('token-2');
		expect(calls).toBe(2);
	});

	it('starts a fresh attempt after a failed one instead of reusing it', async () => {
		let calls = 0;
		const ensure = async () => {
			calls += 1;
			if (calls === 1) throw new Error('refresh token already used');
			return 'token-after-retry';
		};

		await expect(refreshOnce(ensure)).rejects.toThrow('refresh token already used');
		// The rejected attempt must not poison the next one.
		expect(await refreshOnce(ensure)).toBe('token-after-retry');
		expect(calls).toBe(2);
	});

	it('propagates a rejection to every waiter, not just the first', async () => {
		const ensure = () => Promise.reject(new Error('revoked'));

		const results = await Promise.allSettled([refreshOnce(ensure), refreshOnce(ensure)]);

		expect(results.every((r) => r.status === 'rejected')).toBe(true);
	});
});

describe('ensureFreshToken', () => {
	it('refreshes when there is no access token', async () => {
		const ensure = vi.fn(async () => 'fresh');
		expect(await ensureFreshToken(ensure)).toBe('fresh');
		expect(ensure).toHaveBeenCalledOnce();
	});

	it('reuses a still-valid access token without refreshing', async () => {
		saveRotatedTokens('valid-token', 900);
		const ensure = vi.fn(async () => 'should-not-be-used');

		expect(await ensureFreshToken(ensure)).toBe('valid-token');
		expect(ensure).not.toHaveBeenCalled();
	});

	it('refreshes a token that is inside the expiry skew', async () => {
		// 60s TTL with the default 30s skew: not yet expired, but about to be.
		saveRotatedTokens('nearly-expired', 20);
		expect(needsRefresh()).toBe(true);

		const ensure = vi.fn(async () => 'refreshed');
		expect(await ensureFreshToken(ensure)).toBe('refreshed');
		expect(ensure).toHaveBeenCalledOnce();
	});
});

describe('reportSessionExpired', () => {
	it('clears the session and notifies the app', () => {
		saveSession({
			accessToken: 'a',
			expiresIn: 900,
			user: { id: 'u1', name: 'مدیر', role: 'manager' }
		});
		expect(getAccessToken()).toBe('a');

		const onExpired = vi.fn();
		setSessionExpiredHandler(onExpired);

		reportSessionExpired();

		expect(getAccessToken()).toBeNull();
		expect(onExpired).toHaveBeenCalledOnce();
	});
});
