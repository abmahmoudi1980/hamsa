import { describe, expect, it } from 'vitest';
import { ApiError, apiErrorFromBody, apiErrorFromException } from './apiError';

describe('apiErrorFromBody', () => {
	it('reads the documented envelope', () => {
		const err = apiErrorFromBody(400, {
			error: {
				code: 'VALIDATION_ERROR',
				message: 'شماره واحد در این ساختمان تکراری است',
				details: [{ field: 'number', rule: 'unique_per_building' }]
			}
		});

		expect(err.code).toBe('VALIDATION_ERROR');
		expect(err.status).toBe(400);
		expect(err.message).toBe('شماره واحد در این ساختمان تکراری است');
		expect(err.details).toHaveLength(1);
	});

	it('maps details to per-field errors for form state', () => {
		const err = apiErrorFromBody(400, {
			error: {
				code: 'VALIDATION_ERROR',
				message: 'داده نامعتبر',
				details: [
					{ field: 'number', rule: 'unique_per_building' },
					{ field: 'area_m2', rule: 'positive' }
				]
			}
		});

		expect(err.fieldErrors).toEqual({
			number: 'unique_per_building',
			area_m2: 'positive'
		});
	});

	it('drops a field-less detail instead of inventing a field', () => {
		const err = apiErrorFromBody(400, {
			error: { code: 'VALIDATION_ERROR', message: 'مهم', details: [{ rule: 'sum' }] }
		});
		expect(err.fieldErrors).toEqual({});
		expect(err.details).toEqual([{ rule: 'sum' }]);
	});

	it('falls back to the message when a detail has no rule', () => {
		const err = apiErrorFromBody(400, {
			error: { code: 'VALIDATION_ERROR', message: 'مهم', details: [{ field: 'title' }] }
		});
		expect(err.fieldErrors.title).toBe('مهم');
	});

	it('infers the code from the status when the body omits it', () => {
		expect(apiErrorFromBody(403, {}).code).toBe('FORBIDDEN');
		expect(apiErrorFromBody(404, {}).code).toBe('NOT_FOUND');
		expect(apiErrorFromBody(409, {}).code).toBe('CONFLICT');
		expect(apiErrorFromBody(429, {}).code).toBe('RATE_LIMITED');
		expect(apiErrorFromBody(500, {}).code).toBe('INTERNAL');
	});

	it('ignores an unknown code rather than trusting it', () => {
		expect(apiErrorFromBody(400, { error: { code: 'WAT', message: 'x' } }).code).toBe(
			'VALIDATION_ERROR'
		);
	});

	it('produces a Persian fallback for a non-envelope body', () => {
		// A proxy's HTML error page must never reach the UI as raw markup.
		const err = apiErrorFromBody(502, '<html>Bad Gateway</html>');
		expect(err.code).toBe('INTERNAL');
		expect(err.message).toBe('خطای داخلی سرور. لطفاً دوباره تلاش کنید.');
	});

	it('handles a null or empty body', () => {
		expect(apiErrorFromBody(404, null).message).toBe('موردی یافت نشد.');
		expect(apiErrorFromBody(401, undefined).isUnauthenticated).toBe(true);
	});

	it('tolerates a non-array details field', () => {
		const err = apiErrorFromBody(400, {
			error: { code: 'VALIDATION_ERROR', message: 'x', details: 'oops' }
		});
		expect(err.details).toEqual([]);
	});
});

describe('classification helpers', () => {
	it('recognises authentication failures', () => {
		expect(apiErrorFromBody(401, {}).isUnauthenticated).toBe(true);
		expect(new ApiError('UNAUTHENTICATED', 'x', 403).isUnauthenticated).toBe(true);
	});

	it('recognises forbidden, not-found and conflict', () => {
		expect(apiErrorFromBody(403, {}).isForbidden).toBe(true);
		expect(apiErrorFromBody(404, {}).isNotFound).toBe(true);
		expect(apiErrorFromBody(409, {}).isConflict).toBe(true);
		expect(apiErrorFromBody(400, {}).isValidation).toBe(true);
	});

	it('marks only transport faults and 5xx as transient', () => {
		// A 4xx must never be retried — it will fail identically.
		expect(apiErrorFromBody(500, {}).isTransient).toBe(true);
		expect(apiErrorFromBody(429, {}).isTransient).toBe(true);
		expect(new ApiError('INTERNAL', 'x', 0).isTransient).toBe(true);
		expect(apiErrorFromBody(400, {}).isTransient).toBe(false);
		expect(apiErrorFromBody(404, {}).isTransient).toBe(false);
	});
});

describe('apiErrorFromException', () => {
	it('passes an ApiError through unchanged', () => {
		const original = new ApiError('CONFLICT', 'تکراری', 409);
		expect(apiErrorFromException(original)).toBe(original);
	});

	it('reports a transport failure in Persian with a retry hint', () => {
		const err = apiErrorFromException(new TypeError('Failed to fetch'));
		expect(err.status).toBe(0);
		expect(err.isTransient).toBe(true);
		expect(err.message).toBe('ارتباط با سرور برقرار نشد. اتصال اینترنت را بررسی کنید.');
	});

	it('reports an abort distinctly', () => {
		const abort = new DOMException('aborted', 'AbortError');
		expect(apiErrorFromException(abort).message).toBe('درخواست لغو شد.');
	});
});
