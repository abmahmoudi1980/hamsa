/**
 * Single-flight token refresh (research R4).
 *
 * The failure this prevents: a screen that fires four parallel queries gets
 * four 401s at once. Naively each would call `POST /auth/refresh`, but the
 * server ROTATES the refresh token — the first call invalidates the token the
 * other three are holding, so two of them fail, the session is cleared, and the
 * user is logged out mid-session.
 *
 * So: the first 401 starts a refresh; every request that arrives while it is in
 * flight awaits that same promise. When it resolves, all waiters replay with the
 * new access token and exactly one network refresh happened.
 *
 * This mirrors the mobile client's `QueuedInterceptor`
 * (mobile/lib/core/network/refresh_interceptor.dart) so both clients behave the
 * same way.
 */

import { clearSession, getAccessToken, needsRefresh } from './tokenStore';

/** Resolves with a fresh access token, or rejects when the session is gone. */
type RefreshFn = () => Promise<string>;

let inFlight: Promise<string> | null = null;

/** Notified when the session is unrecoverable, so the app routes to /login. */
let onSessionExpired: (() => void) | null = null;

export function setSessionExpiredHandler(handler: (() => void) | null): void {
	onSessionExpired = handler;
}

/**
 * Returns a valid access token, refreshing first when the current one is
 * missing or within its expiry skew. Used before a non-401-triggered request
 * (e.g. the first request after a page reload, where no request has 401'd yet).
 */
export async function ensureFreshToken(ensureFresh: RefreshFn): Promise<string> {
	if (!needsRefresh()) {
		const current = getAccessToken();
		if (current) return current;
	}
	return refreshOnce(ensureFresh);
}

/**
 * Refreshes at most once concurrently: the first caller performs the exchange,
 * the rest await its result.
 */
export function refreshOnce(ensureFresh: RefreshFn): Promise<string> {
	if (inFlight) return inFlight;

	const attempt = ensureFresh().finally(() => {
		// Cleared before the awaiting callers resume, so a 401 from a replayed
		// request starts a new attempt instead of reusing a settled one.
		inFlight = null;
	});

	inFlight = attempt;
	return attempt;
}

/** Tears the session down and notifies the app exactly once. */
export function reportSessionExpired(): void {
	clearSession();
	onSessionExpired?.();
}

/** Test seam: reset module state between cases. */
export function __resetRefreshQueue(): void {
	inFlight = null;
	onSessionExpired = null;
}

/** True while a refresh exchange is running. For assertions in tests. */
export function isRefreshing(): boolean {
	return inFlight !== null;
}
