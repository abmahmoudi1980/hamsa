/**
 * Route guard for the auth section.
 *
 * Runs on the client before the page renders (SPA mode), so an already-signed-in
 * user hitting /login is sent straight to their own shell. This mirrors the
 * mobile router's redirect (app_router.dart `_redirect`).
 */

import { redirect } from '@sveltejs/kit';
import { auth, isManager } from '#lib/auth/auth.svelte';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = () => {
	if (auth.status === 'unknown') return; // startup restore still running
	if (auth.status === 'authenticated') {
		redirect(307, isManager() ? '/m' : '/r');
	}
};
