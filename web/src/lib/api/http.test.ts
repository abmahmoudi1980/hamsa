import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from './apiError';
import { get, post } from './http';
import { clearSession, getAccessToken, saveSession } from './tokenStore';
import { __resetRefreshQueue } from './refreshQueue';

function memoryStorage(): Storage {
	const values = new Map<string, string>();
	return {
		get length() {
			return values.size;
		},
		clear: () => values.clear(),
		getItem: (key) => values.get(key) ?? null,
		key: (index) => [...values.keys()][index] ?? null,
		removeItem: (key) => values.delete(key),
		setItem: (key, value) => values.set(key, value)
	};
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(() => {
	vi.stubGlobal('localStorage', memoryStorage());
	__resetRefreshQueue();
	clearSession();
});

afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
});

describe('HTTP authentication recovery', () => {
	it('refreshes once and replays a request after a 401', async () => {
		saveSession({
			accessToken: 'expired-access',
			expiresIn: 900,
			refreshToken: 'refresh-1',
			user: { id: 'user-1', name: 'مدیر', role: 'manager' }
		});
		const fetchMock = vi
			.fn<typeof fetch>()
			.mockResolvedValueOnce(
				jsonResponse(401, { error: { code: 'UNAUTHENTICATED', message: 'منقضی' } })
			)
			.mockResolvedValueOnce(
				jsonResponse(200, {
					access_token: 'fresh-access',
					refresh_token: 'refresh-2',
					expires_in: 900
				})
			)
			.mockResolvedValueOnce(jsonResponse(200, { value: 'ok' }));
		vi.stubGlobal('fetch', fetchMock);

		const result = await get<{ value: string }>('/test');

		expect(result.value).toBe('ok');
		expect(fetchMock).toHaveBeenCalledTimes(3);
		expect(getAccessToken()).toBe('fresh-access');
	});

	it('does not retry forever when the replay also returns 401', async () => {
		saveSession({
			accessToken: 'expired-access',
			expiresIn: 900,
			refreshToken: 'refresh-1',
			user: { id: 'user-1', name: 'مدیر', role: 'manager' }
		});
		const unauthorized = () =>
			jsonResponse(401, { error: { code: 'UNAUTHENTICATED', message: 'نشست منقضی است' } });
		const fetchMock = vi
			.fn<typeof fetch>()
			.mockResolvedValueOnce(unauthorized())
			.mockResolvedValueOnce(
				jsonResponse(200, {
					access_token: 'fresh-access',
					refresh_token: 'refresh-2',
					expires_in: 900
				})
			)
			.mockResolvedValueOnce(unauthorized());
		vi.stubGlobal('fetch', fetchMock);

		await expect(get('/test')).rejects.toMatchObject({ status: 401 });
		expect(fetchMock).toHaveBeenCalledTimes(3);
		expect(getAccessToken()).toBeNull();
	});

	it('refreshes before sending a protected POST after a page reload', async () => {
		saveSession({
			accessToken: 'old-access',
			expiresIn: 900,
			refreshToken: 'refresh-1',
			user: { id: 'user-1', name: 'مدیر', role: 'manager' }
		});
		clearSession();
		localStorage.setItem('hamsa.refreshToken', 'refresh-1');
		localStorage.setItem(
			'hamsa.user',
			JSON.stringify({ id: 'user-1', name: 'مدیر', role: 'manager' })
		);
		const fetchMock = vi
			.fn<typeof fetch>()
			.mockResolvedValueOnce(
				jsonResponse(200, {
					access_token: 'fresh-access',
					refresh_token: 'refresh-2',
					expires_in: 900
				})
			)
			.mockResolvedValueOnce(jsonResponse(201, { id: 'building-1' }));
		vi.stubGlobal('fetch', fetchMock);

		const result = await post<{ id: string }>('/buildings', { name: 'ساختمان' });

		expect(result.id).toBe('building-1');
		expect(fetchMock).toHaveBeenCalledTimes(2);
		const calls = fetchMock.mock.calls;
		const refreshCall = calls.at(0);
		const mutationCall = calls.at(1);
		expect(refreshCall?.[0]).toBe('/api/v1/auth/refresh');
		expect((mutationCall?.[1]?.headers as Headers).get('Authorization')).toBe(
			'Bearer fresh-access'
		);
	});

	it('does not pre-refresh a login request', async () => {
		const fetchMock = vi
			.fn<typeof fetch>()
			.mockResolvedValueOnce(
				jsonResponse(401, { error: { code: 'UNAUTHENTICATED', message: 'ورود ناموفق' } })
			);
		vi.stubGlobal('fetch', fetchMock);

		await expect(
			post('/auth/login', { phone: '09120000000', password: 'wrong' })
		).rejects.toBeInstanceOf(ApiError);
		expect(fetchMock).toHaveBeenCalledOnce();
	});
});
