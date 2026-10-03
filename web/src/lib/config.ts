/**
 * Runtime configuration.
 *
 * The API base is resolved so the SAME build works in development and
 * production with no rebuild:
 *
 * - Production: the SPA is served from the origin that also reverse-proxies
 *   `/api/` (nginx `app.hamsa-home.ir`), so a relative `/api/v1` base is
 *   same-origin. No CORS preflight, no third-party cookies.
 * - Development: Vite proxies `/api` to the Go server (see vite.config.ts), so
 *   a relative base is again same-origin and the browser needs no CORS grant at
 *   all. `VITE_API_BASE_URL` exists only for the case where you run the API on
 *   a different host/port and deliberately want a cross-origin request.
 */

const DEFAULT_API_BASE = '/api/v1';

/** Absolute API base, or a relative path when served same-origin. */
export const API_BASE: string =
	(import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/+$/, '') ||
	DEFAULT_API_BASE;

/**
 * Path the payment gateway callback should send the browser back to.
 *
 * Sent as `return_path` on `POST /invoices/{id}/pay`; the server 302s here
 * with `?payment_id=&status=&invoice_id=` once it has verified the payment.
 * The server validates this as same-origin, so only a path is accepted.
 */
export const PAYMENT_RESULT_PATH = '/payment/result';

/** localStorage key for the active-building preference (research R8). */
const ACTIVE_BUILDING_KEY = 'hamsa.activeBuildingId';

/**
 * Reads the remembered building id.
 *
 * Manager-only (a resident never picks a building); returns null when unset or
 * when the value is not a UUID, so a hand-edited localStorage entry cannot
 * inject junk into an API path.
 */
export function readActiveBuildingId(): string | null {
	if (typeof localStorage === 'undefined') return null;
	const raw = localStorage.getItem(ACTIVE_BUILDING_KEY);
	return isUuid(raw) ? raw : null;
}

export function writeActiveBuildingId(id: string | null): void {
	if (typeof localStorage === 'undefined') return;
	if (id === null) localStorage.removeItem(ACTIVE_BUILDING_KEY);
	else localStorage.setItem(ACTIVE_BUILDING_KEY, id);
}

/** True for a canonical lowercase-or-uppercase UUID. */
export function isUuid(value: string | null | undefined): value is string {
	if (!value) return false;
	return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
}
