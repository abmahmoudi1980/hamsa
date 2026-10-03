/**
 * Session token storage.
 *
 * Access token: held in memory only. A page reload drops it and the app
 * silently refreshes, so an XSS payload cannot read it out of storage.
 *
 * Refresh token: kept in localStorage. It survives reloads (the whole point of
 * a refresh token) and is the residual risk of a browser client — see
 * research.md F6. It is mitigated, not eliminated:
 *   - strict CSP with no third-party script sources,
 *   - no `innerHTML`/`dangerouslySetInnerHTML` equivalent anywhere in src/,
 *   - the access token never written to disk.
 * The httpOnly-cookie migration (F6 P1) removes this residue entirely; it needs
 * a contract change to /auth/login + /auth/refresh, so it is deliberately out
 * of scope here.
 */

const REFRESH_KEY = 'hamsa.refreshToken';
const USER_KEY = 'hamsa.user';

export interface StoredUser {
	id: string;
	name: string;
	role: 'superadmin' | 'manager' | 'resident';
}

export interface Session {
	accessToken: string;
	/** Unix seconds, from the server's `expires_in`. */
	expiresIn: number;
	/** Persisted to localStorage so a reload can restore the session. */
	refreshToken: string;
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
 * Persists a session. Only the refresh token and the display user reach
 * storage; the access token stays in the module variable.
 */
export function saveSession(session: Session): void {
	accessToken = session.accessToken;
	expiresAt = Date.now() + session.expiresIn * 1000;
	cachedUser = session.user;
	writeStorage(REFRESH_KEY, session.refreshToken);
	writeJson(USER_KEY, session.user);
}

/** Replaces only the token pair, e.g. after a refresh rotation. */
export function saveRotatedTokens(
	accessTokenValue: string,
	expiresIn: number,
	refreshToken: string
): void {
	accessToken = accessTokenValue;
	expiresAt = Date.now() + expiresIn * 1000;
	writeStorage(REFRESH_KEY, refreshToken);
}

export function getRefreshToken(): string | null {
	if (typeof localStorage === 'undefined') return null;
	try {
		return localStorage.getItem(REFRESH_KEY);
	} catch {
		return null;
	}
}

/** Clears every trace of the session. Called on logout and on refresh failure. */
export function clearSession(): void {
	accessToken = null;
	expiresAt = 0;
	cachedUser = null;
	if (typeof localStorage === 'undefined') return;
	localStorage.removeItem(REFRESH_KEY);
	localStorage.removeItem(USER_KEY);
}

/** True when a refresh token is present, i.e. a reload can restore a session. */
export function hasStoredSession(): boolean {
	return getRefreshToken() !== null;
}

function writeStorage(key: string, value: string): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(key, value);
	} catch {
		// Private-mode / quota errors must not break sign-in; the session simply
		// won't survive a reload.
	}
}

function writeJson(key: string, value: unknown): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(key, JSON.stringify(value));
	} catch {
		// ignored — see writeStorage
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
