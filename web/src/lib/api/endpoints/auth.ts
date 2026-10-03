/** Typed session DTOs and auth endpoints (contracts/api.md "Auth"). */

import { get, post } from '#lib/api/http';
import { saveSession, type StoredUser } from '#lib/api/tokenStore';

export type UserRole = StoredUser['role'];

export interface SessionResponse {
	access_token: string;
	/** Still returned for the Android client; the browser relies on the cookie. */
	refresh_token?: string;
	/** Seconds until the access token expires. */
	expires_in: number;
	user: { id: string; name: string; role: UserRole };
}

/**
 * `GET /auth/me` — the authoritative role/scope context.
 *
 * superadmin returns an empty buildings list (it governs none); a manager gets
 * the buildings it may manage; a resident gets the units it occupies.
 */
export interface MeResponse {
	id: string;
	name: string;
	role: UserRole;
	buildings: ManagedBuilding[];
	units: OwnedUnit[];
}

export interface ManagedBuilding {
	id: string;
	name: string;
	address: string | null;
}

export interface OwnedUnit {
	building_id: string;
	unit_id: string;
	number: string;
}

export interface LoginInput {
	phone: string;
	password: string;
}

export interface RegisterInput {
	phone: string;
	/** One-time invite code issued by a manager (or superadmin). */
	code: string;
	password: string;
	name?: string;
}

export interface SetupInput {
	phone: string;
	password: string;
	name?: string;
}

export interface InviteInput {
	phone: string;
	/** Defaults to `resident`; `manager` issues a co-manager. */
	role?: Exclude<UserRole, 'superadmin'>;
}

export interface InviteResult {
	code: string;
	role: Exclude<UserRole, 'superadmin'>;
	expires_in_days: number;
}

/** Persists the session from a login/register/setup response. */
function adoptSession(data: SessionResponse): StoredUser {
	const user: StoredUser = { id: data.user.id, name: data.user.name, role: data.user.role };
	saveSession({
		accessToken: data.access_token,
		expiresIn: data.expires_in,
		user
	});
	return user;
}

/**
 * `POST /auth/login`. The server answers 401 with the same code for an unknown
 * phone and a wrong password, so this never reveals whether an account exists.
 */
export async function login(input: LoginInput): Promise<StoredUser> {
	const data = await post<SessionResponse>('/auth/login', input);
	return adoptSession(data);
}

/** `POST /auth/register` — redeems a one-time invite code. */
export async function register(input: RegisterInput): Promise<StoredUser> {
	const data = await post<SessionResponse>('/auth/register', input);
	return adoptSession(data);
}

/**
 * `POST /auth/setup` — first-run bootstrap, creating the superadmin.
 * The server answers 409 once any active user exists, which the caller turns
 * into a "setup is not available" message rather than a generic error.
 */
export async function setup(input: SetupInput): Promise<StoredUser> {
	const data = await post<SessionResponse>('/auth/setup', input);
	return adoptSession(data);
}

/** `GET /auth/me` — refreshes the cached role/scope after a reload. */
export async function me(): Promise<MeResponse> {
	return get<MeResponse>('/auth/me');
}

/**
 * `POST /auth/logout` — revokes the refresh token family server-side.
 *
 * No body needed: the refresh token is in the httpOnly cookie and the request
 * carries the double-submit CSRF token. The server clears both cookies.
 */
export async function logout(): Promise<void> {
	await post<null>('/auth/logout');
}

/** `POST /auth/invites` — manager/superadmin only. */
export async function createInvite(input: InviteInput): Promise<InviteResult> {
	return post<InviteResult>('/auth/invites', input);
}

/** `POST /auth/password` — change password, 204 on success. */
export async function changePassword(input: {
	current_password: string;
	new_password: string;
}): Promise<void> {
	await post<null>('/auth/password', input);
}
