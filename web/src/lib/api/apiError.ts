/**
 * The API error envelope (contracts/api.md):
 *
 *   { "error": { "code", "message", "details": [{ "field", "rule" }] } }
 *
 * Every non-2xx response uses it, so this module is the single place that
 * knows how to read one. `message` is always Persian (the product is
 * Persian-only) and is rendered verbatim — never translated client-side.
 */

export type ApiErrorCode =
	| 'VALIDATION_ERROR'
	| 'UNAUTHENTICATED'
	| 'FORBIDDEN'
	| 'NOT_FOUND'
	| 'CONFLICT'
	| 'RATE_LIMITED'
	| 'INTERNAL';

/** One field-level validation failure from the server. */
export interface ApiErrorDetail {
	field?: string;
	rule?: string;
}

/** A parsed error envelope, also used for network/abort failures. */
export class ApiError extends Error {
	readonly code: ApiErrorCode;
	readonly status: number;
	readonly details: ApiErrorDetail[];
	/** Per-field messages, ready to merge into a form state. */
	readonly fieldErrors: Readonly<Record<string, string>>;

	constructor(code: ApiErrorCode, message: string, status = 0, details: ApiErrorDetail[] = []) {
		super(message);
		this.name = 'ApiError';
		this.code = code;
		this.status = status;
		this.details = details;

		const perField: Record<string, string> = {};
		for (const d of details) {
			// A field-less detail (e.g. a cross-field rule) has no place to land;
			// the top-level message already carries it.
			if (d.field) perField[d.field] = d.rule ?? message;
		}
		this.fieldErrors = perField;
	}

	/** True when the session is gone and the user must sign in again. */
	get isUnauthenticated(): boolean {
		return this.code === 'UNAUTHENTICATED' || this.status === 401;
	}

	get isForbidden(): boolean {
		return this.code === 'FORBIDDEN' || this.status === 403;
	}

	/** A 404 that came from our own router rather than the API. */
	get isNotFound(): boolean {
		return this.code === 'NOT_FOUND' || this.status === 404;
	}

	/** A rejected state transition or a duplicate (409). */
	get isConflict(): boolean {
		return this.code === 'CONFLICT' || this.status === 409;
	}

	get isValidation(): boolean {
		return this.code === 'VALIDATION_ERROR' || this.status === 400;
	}

	/** Retryable only for transport/server faults, never for a 4xx. */
	get isTransient(): boolean {
		return this.status === 0 || this.status === 429 || this.status >= 500;
	}
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null;
}

/** Narrows an unknown `details` entry without trusting its shape. */
function toDetail(value: unknown): ApiErrorDetail | null {
	if (!isRecord(value)) return null;
	const field = typeof value.field === 'string' ? value.field : undefined;
	const rule = typeof value.rule === 'string' ? value.rule : undefined;
	return field || rule ? { field, rule } : null;
}

const KNOWN_CODES: ReadonlySet<string> = new Set([
	'VALIDATION_ERROR',
	'UNAUTHENTICATED',
	'FORBIDDEN',
	'NOT_FOUND',
	'CONFLICT',
	'RATE_LIMITED',
	'INTERNAL'
]);

/**
 * Builds an ApiError from a non-2xx response body.
 *
 * Falls back to a generic Persian message when the body is not the documented
 * envelope (a proxy error page, an empty 502), so the UI always has something
 * safe to show instead of leaking raw HTML or JSON to the user.
 */
export function apiErrorFromBody(status: number, body: unknown): ApiError {
	if (isRecord(body) && isRecord(body.error)) {
		const err = body.error;
		const code =
			typeof err.code === 'string' && KNOWN_CODES.has(err.code)
				? (err.code as ApiErrorCode)
				: codeForStatus(status);
		const message =
			typeof err.message === 'string' && err.message.trim() !== ''
				? err.message
				: fallbackMessage(status);
		const details = Array.isArray(err.details)
			? err.details.map(toDetail).filter((d): d is ApiErrorDetail => d !== null)
			: [];
		return new ApiError(code, message, status, details);
	}
	return new ApiError(codeForStatus(status), fallbackMessage(status), status);
}

/** Wraps a thrown transport error (offline, DNS, CORS, abort). */
export function apiErrorFromException(err: unknown): ApiError {
	if (err instanceof ApiError) return err;
	if (err instanceof DOMException && err.name === 'AbortError') {
		return new ApiError('INTERNAL', 'درخواست لغو شد.', 0);
	}
	// Iranian mobile networks drop frequently; this is the common case and needs
	// wording that suggests retrying rather than blaming the user.
	return new ApiError('INTERNAL', 'ارتباط با سرور برقرار نشد. اتصال اینترنت را بررسی کنید.', 0);
}

function codeForStatus(status: number): ApiErrorCode {
	switch (status) {
		case 400:
			return 'VALIDATION_ERROR';
		case 401:
			return 'UNAUTHENTICATED';
		case 403:
			return 'FORBIDDEN';
		case 404:
			return 'NOT_FOUND';
		case 409:
			return 'CONFLICT';
		case 429:
			return 'RATE_LIMITED';
		default:
			return status >= 500 ? 'INTERNAL' : 'VALIDATION_ERROR';
	}
}

function fallbackMessage(status: number): string {
	switch (status) {
		case 400:
			return 'داده‌های ارسالی نامعتبر است.';
		case 401:
			return 'برای این عملیات باید وارد شوید.';
		case 403:
			return 'دسترسی غیرمجاز است.';
		case 404:
			return 'موردی یافت نشد.';
		case 409:
			return 'این عملیات با وضعیت فعلی امکان‌پذیر نیست.';
		case 429:
			return 'تعداد درخواست‌ها زیاد است. کمی بعد دوباره تلاش کنید.';
		default:
			return status >= 500 ? 'خطای داخلی سرور. لطفاً دوباره تلاش کنید.' : 'درخواست ناموفق بود.';
	}
}
