/**
 * The single HTTP entry point for the whole client.
 *
 * Responsibilities, in order:
 *   1. attach the bearer token (and refresh it first when it is near expiry),
 *   2. on a 401, refresh once and replay the original request,
 *   3. parse the error envelope into `ApiError`,
 *   4. hand parsed JSON (or null) back.
 *
 * Every feature module talks to the API through this, so refresh and error
 * handling cannot drift per-endpoint.
 */

import { API_BASE } from '#lib/config';
import { apiErrorFromBody, apiErrorFromException } from './apiError';
import { CSRF_HEADER_NAME, readCsrfToken } from './csrf';
import { ensureFreshToken, refreshOnce, reportSessionExpired } from './refreshQueue';
import { getAccessToken, saveRotatedTokens } from './tokenStore';

/** Paths that must never trigger a refresh-and-replay. */
const AUTH_PATHS = [
	'/auth/refresh',
	'/auth/login',
	'/auth/setup',
	'/auth/register',
	// Logout is cookie-authenticated: it must never refresh first (a dead
	// session would abort the sign-out before the server clears the cookies).
	'/auth/logout'
];

export interface RequestOptions {
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	/** JSON body; serialized with UTF-8 content type. */
	body?: unknown;
	/** Query values; undefined/null entries are dropped. */
	query?: Record<string, string | number | boolean | undefined | null>;
	/** Send a FormData body untouched (file upload). */
	formData?: FormData;
	signal?: AbortSignal;
	/** Set for uploads so the browser can report progress. */
	onUploadProgress?: (fraction: number) => void;
}

/** The `/auth/refresh` response, in the wire's snake_case. */
export interface AuthTokens {
	access_token: string;
	/** Still returned for the Android client; the browser uses the cookie. */
	refresh_token?: string;
	/** Seconds until the new access token expires. */
	expires_in?: number;
}

/** Raw JSON GET. Throws ApiError on any non-2xx. */
export async function get<T>(path: string, options: RequestOptions = {}): Promise<T> {
	return request<T>(path, { ...options, method: 'GET' });
}

export async function post<T>(
	path: string,
	body?: unknown,
	options: RequestOptions = {}
): Promise<T> {
	return request<T>(path, { ...options, method: 'POST', body });
}

export async function put<T>(
	path: string,
	body?: unknown,
	options: RequestOptions = {}
): Promise<T> {
	return request<T>(path, { ...options, method: 'PUT', body });
}

export async function patch<T>(
	path: string,
	body?: unknown,
	options: RequestOptions = {}
): Promise<T> {
	return request<T>(path, { ...options, method: 'PATCH', body });
}

export async function del<T>(path: string, options: RequestOptions = {}): Promise<T> {
	return request<T>(path, { ...options, method: 'DELETE' });
}

/** 204 No Content (and empty bodies generally) resolve to null. */
async function request<T>(path: string, options: RequestOptions, retried = false): Promise<T> {
	let response: Response;
	try {
		({ response } = await send(path, options, retried));
	} catch (error) {
		const apiError = apiErrorFromException(error);
		if (apiError.isUnauthenticated && !isAuthPath(path)) reportSessionExpired();
		throw apiError;
	}

	if (response.status === 401 && !isAuthPath(path)) {
		if (retried) {
			reportSessionExpired();
			throw apiErrorFromBody(response.status, await readBody(response));
		}

		// The session may simply have expired. One refresh-and-replay, shared
		// with any other request that 401'd at the same moment.
		try {
			await refreshOnce(refreshAccessToken);
		} catch (error) {
			const apiError = apiErrorFromException(error);
			if (apiError.isUnauthenticated) reportSessionExpired();
			throw apiError;
		}
		return request<T>(path, options, true);
	}

	if (!response.ok) {
		throw apiErrorFromBody(response.status, await readBody(response));
	}

	return (await readBody(response)) as T;
}

interface SendResult {
	response: Response;
	/** True on the replay, so a 401 cannot loop forever. */
	retried: boolean;
}

async function send(path: string, options: RequestOptions, retried = false): Promise<SendResult> {
	const method = options.method ?? 'GET';
	const url = buildUrl(path, options.query);

	const headers = new Headers({ Accept: 'application/json', 'X-Hamsa-Client': 'web' });
	let body: BodyInit | undefined;

	if (options.formData) {
		// No Content-Type: the browser must add the multipart boundary itself.
		body = options.formData;
	} else if (options.body !== undefined) {
		headers.set('Content-Type', 'application/json; charset=utf-8');
		body = JSON.stringify(options.body);
	}

	// Cookie-authenticated writes (refresh, logout) must echo the double-submit
	// token; harmless on every other request, which the server never checks.
	const csrf = readCsrfToken();
	if (csrf && method !== 'GET') {
		headers.set(CSRF_HEADER_NAME, csrf);
	}

	// A pre-flight refresh avoids a wasted round trip on the very first request
	// after a reload, where no request has had the chance to 401 yet.
	const token = isAuthPath(path) ? getAccessToken() : await ensureFreshToken(refreshAccessToken);
	if (token) headers.set('Authorization', `Bearer ${token}`);

	const response = await fetch(url, {
		method,
		headers,
		body,
		signal: options.signal,
		// Sends the httpOnly refresh cookie on same-origin calls; required
		// explicitly for the cross-origin dev setup.
		credentials: 'include',
		...(options.onUploadProgress && options.formData ? { duplex: 'half' } : {})
	} as RequestInit);

	return { response, retried };
}

function isAuthPath(path: string): boolean {
	return AUTH_PATHS.some((p) => path.startsWith(p));
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
	const base = `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`;
	if (!query) return base;

	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(query)) {
		if (value === undefined || value === null || value === '') continue;
		params.set(key, String(value));
	}
	const qs = params.toString();
	return qs ? `${base}?${qs}` : base;
}

/**
 * Reads a response body as JSON, tolerating 204/empty and non-JSON error pages.
 * A body that will not parse becomes a null result rather than throwing, so the
 * caller reports a clean envelope error instead of a JSON syntax error.
 */
async function readBody(response: Response): Promise<unknown> {
	if (response.status === 204) return null;
	const text = await response.text();
	if (text === '') return null;
	try {
		return JSON.parse(text);
	} catch {
		return null;
	}
}

/**
 * Exchanges the refresh cookie for a new access token.
 *
 * Must bypass `request()` entirely: it is the one call that cannot itself be
 * retried, or a failing refresh would recurse. The refresh token travels in the
 * httpOnly cookie (credentials: include) and the matching CSRF token is echoed
 * in the header; the browser never reads the refresh token itself.
 */
export async function refreshAccessToken(): Promise<string> {
	const headers: Record<string, string> = {
		Accept: 'application/json',
		'X-Hamsa-Client': 'web'
	};
	const csrf = readCsrfToken();
	if (csrf) headers[CSRF_HEADER_NAME] = csrf;

	const response = await fetch(`${API_BASE}/auth/refresh`, {
		method: 'POST',
		headers,
		credentials: 'include'
	});

	if (!response.ok) {
		throw apiErrorFromBody(response.status, await readBody(response));
	}

	const data = (await readBody(response)) as AuthTokens | null;
	if (!data?.access_token) {
		throw apiErrorFromException(new Error('malformed refresh response'));
	}

	saveRotatedTokens(data.access_token, data.expires_in ?? 900);
	return data.access_token;
}
