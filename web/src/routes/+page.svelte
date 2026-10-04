<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth, isManager, setGuest } from '#lib/auth/auth.svelte';
	import { me } from '#lib/api/endpoints/auth';
	import { ApiError } from '#lib/api/apiError';
	import { fa } from '#i18n/fa';

	/**
	 * Decides where to land after a sign-in.
	 *
	 * A manager or the superadmin goes to the manager console; a resident to
	 * their panel. The superadmin shares the manager shell because it governs no
	 * buildings but still needs the invite tool (002-multi-manager-support).
	 */
	function homeFor(): string {
		return isManager() ? '/m' : '/r';
	}

	async function restore() {
		// No session at all: nothing to restore.
		if (auth.status === 'guest') {
			await goto('/login', { replace: true });
			return;
		}

		// A refresh token exists, so try to exchange it for a live session.
		try {
			const profile = await me();
			auth.user = { id: profile.id, name: profile.name, role: profile.role };
			auth.status = 'authenticated';
			await goto(homeFor(), { replace: true });
		} catch (err) {
			// An invalid or expired refresh token is the normal case here (the
			// stored token can be days old), so it lands on the login screen with
			// no scary message. Anything else keeps the session and says so.
			if (err instanceof ApiError && (err.isUnauthenticated || err.isForbidden)) {
				setGuest(null);
				await goto('/login', { replace: true });
			} else {
				auth.status = 'unknown';
				auth.error = err instanceof ApiError ? err.message : fa.errorGeneric;
			}
		}
	}

	onMount(() => {
		void restore();
	});
</script>

<main class="grid min-h-dvh place-items-center bg-surface px-6">
	<div class="flex flex-col items-center text-center" role="status" aria-live="polite">
		<div
			class="grid size-16 place-items-center rounded-3xl bg-primary text-on-primary shadow-lg shadow-primary/15"
		>
			<svg viewBox="0 0 32 32" fill="none" class="size-9" aria-hidden="true">
				<path
					d="M5 27V13.5L16 5l11 8.5V27h-8v-8h-6v8H5Z"
					stroke="currentColor"
					stroke-width="2.2"
					stroke-linejoin="round"
				/>
				<path
					d="M2.5 14.5 16 4l13.5 10.5"
					stroke="currentColor"
					stroke-width="2.2"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
		</div>
		<p class="mt-5 text-xl font-bold text-on-surface">{fa.appName}</p>
		{#if auth.error}
			<p class="mt-2 max-w-sm text-sm text-danger" role="alert">{auth.error}</p>
			<button class="mt-5 btn-primary" type="button" onclick={restore}>{fa.retry}</button>
		{:else}
			<p class="mt-1 text-sm text-on-surface-variant">{fa.loading}</p>
			<div class="mt-6 h-1 w-36 overflow-hidden rounded-full bg-surface-high">
				<div class="h-full w-1/2 animate-pulse rounded-full bg-primary"></div>
			</div>
		{/if}
	</div>
</main>
