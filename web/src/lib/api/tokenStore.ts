/**
 * Session token storage.
 *
 * Access token: held in memory only. A page reload drops it and the app
 * silently refreshes, so an XSS payload cannot read it out of storage.
 *
 * Refresh token: NOT in JavaScript-reachable storage at all. It lives in an
 * httpOnly cookie the browser attaches to `/api/v1/auth/refresh` (F6,
 * 003-web-frontend), so a script injected into the page cannot exfiltrate it.
 * The cookie is `SameSite=Lax`, which also blocks the cross-site POST a CSRF
 * attack needs; cookie-authenticated writes additionally carry the double-submit
 * `X-CSRF-Token` (see csrf.ts).
 *
 * The display user stays in `localStorage` so a reload can render the shell
 * before `/auth/me` answers; it is not a credential.
 */

const USER_KEY = 'hamsa.user';

export interface StoredUser {
	id: string;
	name: string;
	role: 'superadmin' | 'manager' | 'resident';
}

export interface Session {
	accessToken: string;
	/** Seconds, from the server's `expires_in`. */
	expiresIn: number;
	user: StoredUser;
}

/** In-memory only: cleared on reload by design. */
let accessToken: string | null = null;
let expiresAt = 0;
/** The user is persisted so a reload can render a shell before /auth/me returns. */
let cachedUser: StoredUser | null = readJson<StoredUser>(USER_KEY);

export function getAccessToken(): string | null {
	return accessToken;
}

export function getCachedUser(): StoredUser | null {
	return cachedUser;
}

export function isManagerRole(role: StoredUser['role'] | undefined): boolean {
	return role === 'manager' || role === 'superadmin';
}

/** True when the access token is absent or within `skewMs` of expiry. */
export function needsRefresh(skewMs = 30_000): boolean {
	if (!accessToken) return true;
	return Date.now() >= expiresAt - skewMs;
}

/**
 * Persists a session. Only the display user reaches storage; both tokens stay
 * out of JavaScript-reachable persistence (access in memory, refresh in the
 * httpOnly cookie).
 */
export function saveSession(session: Session): void {
	accessToken = session.accessToken;
	expiresAt = Date.now() + session.expiresIn * 1000;
	cachedUser = session.user;
	writeJson(USER_KEY, session.user);
}

/** Replaces the in-memory access token after a refresh rotation. */
export function saveRotatedTokens(accessTokenValue: string, expiresIn: number): void {
	accessToken = accessTokenValue;
	expiresAt = Date.now() + expiresIn * 1000;
}

/** Clears every trace of the session. Called on logout and on refresh failure. */
export function clearSession(): void {
	accessToken = null;
	expiresAt = 0;
	cachedUser = null;
	if (typeof localStorage === 'undefined') return;
	localStorage.removeItem(USER_KEY);
}

/**
 * True when a reload may have a restorable session.
 *
 * The refresh cookie is httpOnly, so the client cannot detect it directly; the
 * persisted display user is the cheap proxy. A false negative only means an
 * unnecessary trip to the login screen.
 */
export function hasPersistedUser(): boolean {
	return cachedUser !== null;
}

function writeJson(key: string, value: unknown): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(key, JSON.stringify(value));
	} catch {
		// Private-mode / quota errors must not break sign-in; the session simply
		// won't survive a reload.
	}
}

function readJson<T>(key: string): T | null {
	if (typeof localStorage === 'undefined') return null;
	try {
		const raw = localStorage.getItem(key);
		if (!raw) return null;
		const parsed: unknown = JSON.parse(raw);
		return isStoredUser(parsed) ? (parsed as T) : null;
	} catch {
		return null;
	}
}

/** Validates the persisted user so a tampered entry can't fake a role. */
function isStoredUser(value: unknown): boolean {
	if (typeof value !== 'object' || value === null) return false;
	const u = value as Record<string, unknown>;
	return (
		typeof u.id === 'string' &&
		typeof u.name === 'string' &&
		(u.role === 'superadmin' || u.role === 'manager' || u.role === 'resident')
	);
}
