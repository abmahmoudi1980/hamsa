<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth, restoreSession, setGuest } from '#lib/auth/auth.svelte';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { logout } from '#lib/api/endpoints/auth';
	import { clearSession, isManagerRole } from '#lib/api/tokenStore';
	import { clearFileObjectUrls } from '#lib/files/fileCache';
	import { fa } from '#i18n/fa';
	import { roleLabel } from '#lib/format/labels';
	import NotificationBell from '#lib/components/NotificationBell.svelte';

	let { children } = $props();
	const queryClient = useQueryClient();

	$effect(() => {
		const status = auth.status;
		const role = auth.user?.role;
		if (status === 'guest') void goto('/login', { replace: true });
		if (status === 'authenticated' && role && isManagerRole(role)) {
			void goto('/m', { replace: true });
		}
	});

	onMount(async () => {
		if (auth.status === 'unknown') await restoreSession();
		if (auth.status === 'authenticated' && auth.user && isManagerRole(auth.user.role)) {
			await goto('/m', { replace: true });
		}
	});

	async function signOut() {
		try {
			await logout();
		} catch {
			clearSession();
		} finally {
			queryClient.clear();
			clearFileObjectUrls();
			setGuest();
		}
	}
</script>

{#if auth.status === 'authenticated' && auth.user?.role === 'resident'}
	<div class="min-h-dvh bg-surface">
		<header
			class="no-print sticky top-0 z-20 flex min-h-16 items-center justify-between border-b border-outline-variant/50 bg-white/90 px-4 backdrop-blur sm:px-7"
		>
			<a href="/r" class="flex items-center gap-3">
				<span class="grid size-10 place-items-center rounded-2xl bg-primary text-on-primary">
					<svg viewBox="0 0 32 32" fill="none" class="size-6" aria-hidden="true">
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
						/>
					</svg>
				</span>
				<span>
					<span class="block text-lg leading-tight font-bold">{fa.appName}</span>
					<span class="text-xs text-on-surface-variant">{roleLabel(auth.user.role)}</span>
				</span>
			</a>
			<div class="flex items-center gap-1">
				<NotificationBell href="/r/notifications" />
				<button class="icon-button" type="button" aria-label={fa.logout} onclick={signOut}>
					<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
						<path
							d="M10 17l5-5-5-5m5 5H3m9-9h6a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-6"
							stroke="currentColor"
							stroke-width="1.8"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
				</button>
			</div>
		</header>

		<main
			class="mx-auto min-h-[calc(100dvh-4rem)] w-full max-w-5xl px-4 py-6 pb-24 sm:px-7 sm:py-8"
		>
			{@render children()}
		</main>

		<nav
			class="no-print fixed inset-x-0 bottom-0 z-30 flex gap-1 overflow-x-auto border-t border-outline-variant/60 bg-white/95 px-3 py-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] backdrop-blur"
			aria-label={fa.mainNavigation}
		>
			<a class="mobile-nav-link" href="/r" aria-current="page">
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
					><path
						d="m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-6v-7h-4v7H4a1 1 0 0 1-1-1V10Z"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linejoin="round"
					/></svg
				>
				{fa.navHome}
			</a>
			<a class="mobile-nav-link" href="/r/invoices">
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
					><path
						d="M7 3h10a1 1 0 0 1 1 1v16l-3-2-3 2-3-2-3 2V4a1 1 0 0 1 1-1ZM9 8h6M9 12h6"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linecap="round"
						stroke-linejoin="round"
					/></svg
				>
				{fa.navInvoices}
			</a>
			<a class="mobile-nav-link" href="/r/payments">
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
					><path
						d="M2 8h20v10a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V8Zm0-3h20v3H2V5Zm4 9h4"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linecap="round"
						stroke-linejoin="round"
					/></svg
				>
				{fa.navPayments}
			</a>
			<a class="mobile-nav-link" href="/r/maintenance">
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
					><path
						d="M14.7 6.3a4 4 0 0 0-5.7 5.7l-5.5 5.5a2 2 0 0 0 2.8 2.8l5.5-5.5a4 4 0 0 0 5.7-5.7l-2.3 2.3-3-3 2.5-2.1Z"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linecap="round"
						stroke-linejoin="round"
					/></svg
				>
				{fa.navMaintenance}
			</a>
			<a class="mobile-nav-link" href="/r/announcements">
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
					><path
						d="M4 14V8a2 2 0 0 1 2-2h2l9-3v18l-9-3H6a2 2 0 0 1-2-2Zm4 2 2 5h4l-2.2-4.3M20 8a5 5 0 0 1 0 6"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linecap="round"
						stroke-linejoin="round"
					/></svg
				>
				{fa.navAnnouncements}
			</a>
			<a class="mobile-nav-link" href="/r/notifications">
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
					><path
						d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9ZM10 21h4"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linecap="round"
						stroke-linejoin="round"
					/></svg
				>
				{fa.navNotifications}
			</a>
		</nav>
	</div>
{:else if auth.error}
	<main class="grid min-h-dvh place-items-center px-6">
		<div class="max-w-md text-center">
			<p class="text-sm text-danger" role="alert">{auth.error}</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void restoreSession()}
				>{fa.retry}</button
			>
		</div>
	</main>
{:else}
	<main class="grid min-h-dvh place-items-center" role="status" aria-live="polite">
		<div class="flex flex-col items-center gap-4">
			<span class="size-6 animate-spin rounded-full border-[3px] border-primary/25 border-t-primary"
			></span>
			<p class="text-sm text-on-surface-variant">{fa.loading}</p>
		</div>
	</main>
{/if}
