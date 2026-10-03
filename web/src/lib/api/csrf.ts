/**
 * Double-submit CSRF token (003-web-frontend, F6).
 *
 * The refresh token lives in an httpOnly cookie the browser attaches
 * automatically, so a request to /auth/refresh or /auth/logout is not proof
 * that our JavaScript made it. The server therefore also sets a readable
 * `hamsa_csrf` cookie and requires its value echoed in `X-CSRF-Token` on any
 * cookie-authenticated state change.
 *
 * Reading it is all the token needs: it is not a secret (the cookie is
 * readable by design), only proof of same-origin script access.
 */

export const CSRF_COOKIE_NAME = 'hamsa_csrf';
export const CSRF_HEADER_NAME = 'X-CSRF-Token';

function escapeForRegExp(value: string): string {
	return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** The current CSRF token, or null when no cookie is present (guest). */
export function readCsrfToken(): string | null {
	if (typeof document === 'undefined' || !document.cookie) return null;
	const match = document.cookie.match(
		new RegExp(`(?:^|;\\s*)${escapeForRegExp(CSRF_COOKIE_NAME)}=([^;]*)`)
	);
	return match?.[1] !== undefined ? decodeURIComponent(match[1]) : null;
}
