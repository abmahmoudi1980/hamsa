/** Shared validators (mirrors mobile/lib/shared/validation/validators.dart). */

/**
 * Iranian mobile number.
 *
 * Accepts the forms a user actually types: 0912…, +98912…, 0098912…, with
 * Persian/Arabic-Indic digits and spaces/dashes. Returns the canonical
 * `09xxxxxxxxx` form, or null when invalid.
 */
export function normalizePhone(input: string): string | null {
	const digits = normalizeDigits(input).replace(/[\s-]/g, '');
	if (!/^\d+$/.test(digits)) return null;

	if (digits.startsWith('0098')) return toLocal(digits.slice(4));
	if (digits.startsWith('98') && digits.length === 12) return toLocal(digits.slice(2));
	if (digits.startsWith('0') && digits.length === 11) return digits;
	// Bare national number without the leading zero.
	if (digits.length === 10 && digits.startsWith('9')) return `0${digits}`;
	return null;
}

/** True when the input is a valid Iranian mobile number in any accepted form. */
export function isValidPhone(input: string): boolean {
	return normalizePhone(input) !== null;
}

/** The server's policy: ≥ 8 characters with at least one letter and one digit. */
export function isValidPassword(password: string): boolean {
	return password.length >= 8 && /[A-Za-zء-ی]/.test(password) && /[0-9۰-۹٠-٩]/.test(password);
}

/** Invite codes are short, human-transcribed strings. */
export function isValidInviteCode(code: string): boolean {
	return /^[A-Za-z0-9]{4,32}$/.test(normalizeDigits(code).trim());
}

/** A required free-text field. */
export function isRequired(value: string): boolean {
	return value.trim().length > 0;
}

/** Optional text, capped to a sane maximum. */
export function maxLength(value: string, limit: number): boolean {
	return value.length <= limit;
}

/** Non-negative integer as typed (used for areas, counts, floors). */
export function isNonNegativeInteger(input: string): boolean {
	const digits = normalizeDigits(input).trim();
	return /^\d+$/.test(digits);
}

function toLocal(withoutCountryCode: string): string | null {
	if (withoutCountryCode.length === 10 && withoutCountryCode.startsWith('9')) {
		return `0${withoutCountryCode}`;
	}
	return null;
}

/** Persian/Arabic-Indic digits → Latin, so validation sees one script. */
function normalizeDigits(input: string): string {
	let out = '';
	for (const ch of input) {
		const code = ch.codePointAt(0) ?? 0;
		if (code >= 0x30 && code <= 0x39) out += ch;
		else if (code >= 0x06f0 && code <= 0x06f9) out += String.fromCharCode(code - 0x06f0 + 0x30);
		else if (code >= 0x0660 && code <= 0x0669) out += String.fromCharCode(code - 0x0660 + 0x30);
		else out += ch;
	}
	return out;
}
