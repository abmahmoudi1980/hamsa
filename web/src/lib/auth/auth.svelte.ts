/**
 * Session state for the whole app.
 *
 * Svelte 5 runes hold the state, and a small set of derived helpers exposes it.
 * Deliberately NOT a store: runes keep the value readable at its declaration
 * and avoid the `$`-prefix shadowing that makes store values easy to pass to a
 * function by accident.
 *
 * The flow mirrors the mobile app's auth controller:
 *   unknown → (restore from the httpOnly refresh cookie) → authenticated | guest
 *
 * `unknown` exists because the access token lives in memory only, so after a
 * page load we cannot know whether the session is valid until /auth/me answers.
 * Routes must wait on this state rather than assuming a session.
 */

import { clearSession, getCachedUser, hasPersistedUser, type StoredUser } from '../api/tokenStore';
import { setSessionExpiredHandler } from '../api/refreshQueue';
import { me } from '../api/endpoints/auth';
import { apiErrorFromException } from '../api/apiError';

export type AuthStatus = 'unknown' | 'authenticated' | 'guest';

export interface AuthState {
	status: AuthStatus;
	user: StoredUser | null;
	/** Set when sign-in or session restore failed, for a banner. */
	error: string | null;
	/** True while a sign-in request is in flight. */
	pending: boolean;
}

export const auth = $state<AuthState>({
	status: hasPersistedUser() ? 'unknown' : 'guest',
	user: getCachedUser(),
	error: null,
	pending: false
});

let restorePromise: Promise<StoredUser | null> | null = null;

export function restoreSession(): Promise<StoredUser | null> {
	if (auth.status === 'authenticated' && auth.user) return Promise.resolve(auth.user);
	if (restorePromise) return restorePromise;

	restorePromise = (async () => {
		if (!hasPersistedUser()) {
			setGuest();
			return null;
		}

		auth.status = 'unknown';
		auth.error = null;

		try {
			const profile = await me();
			const user: StoredUser = { id: profile.id, name: profile.name, role: profile.role };
			setAuthenticated(user);
			return user;
		} catch (error) {
			const apiError = apiErrorFromException(error);
			if (apiError.isUnauthenticated || apiError.isForbidden) {
				setGuest();
				return null;
			}
			auth.status = 'unknown';
			auth.error = apiError.message;
			return null;
		}
	})().finally(() => {
		restorePromise = null;
	});

	return restorePromise;
}

/**
 * Applies a restored or freshly-issued session.
 * Called by the login/register flows and by the startup restore.
 */
export function setAuthenticated(user: StoredUser): void {
	auth.status = 'authenticated';
	auth.user = user;
	auth.error = null;
	auth.pending = false;
}

/** Drops back to the guest state (logout, or an unrecoverable session). */
export function setGuest(error: string | null = null): void {
	clearSession();
	auth.status = 'guest';
	auth.user = null;
	auth.error = error;
	auth.pending = false;
}

/** Records a failed attempt without changing the signed-in state. */
export function setAuthError(message: string): void {
	auth.error = message;
	auth.pending = false;
}

export function setAuthPending(pending: boolean): void {
	auth.pending = pending;
}

/**
 * True for manager and superadmin — the roles that reach the manager shell.
 *
 * Reads `auth` directly: it is a `$state` proxy, so the read is tracked and
 * callers inside a `$derived` stay reactive without an explicit `get()`.
 */
export function isManager(): boolean {
	const role = auth.user?.role;
	return role === 'manager' || role === 'superadmin';
}

/**
 * The superadmin governs no buildings but shares the manager shell for the
 * invite tool, matching the mobile router's redirect (002-multi-manager-support).
 */
export function isSuperAdmin(): boolean {
	return auth.user?.role === 'superadmin';
}

/**
 * Wires the refresh queue's failure callback to the guest state. Called once
 * from the root layout so an expired session anywhere in the app lands on the
 * login screen.
 */
export function installSessionExpiryHandler(clearPrivateCache?: () => void): void {
	setSessionExpiredHandler(() => {
		clearPrivateCache?.();
		setGuest('نشست شما منقضی شده است. دوباره وارد شوید.');
	});
}
