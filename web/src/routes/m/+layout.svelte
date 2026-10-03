<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { auth, isManager, restoreSession, setGuest } from '#lib/auth/auth.svelte';
	import { listBuildings } from '#lib/api/endpoints/buildings';
	import { logout } from '#lib/api/endpoints/auth';
	import { clearSession } from '#lib/api/tokenStore';
	import { buildingSelection, selectBuilding } from '#lib/building/selection.svelte';
	import { fa } from '#i18n/fa';
	import { roleLabel } from '#lib/format/labels';
	import { qk } from '#lib/query/keys';

	let { children } = $props();
	const queryClient = useQueryClient();

	const buildingsQuery = createQuery(() => ({
		queryKey: qk.buildings.list(),
		queryFn: listBuildings,
		enabled: auth.status === 'authenticated' && auth.user?.role === 'manager'
	}));

	const buildings = $derived(buildingsQuery.data ?? []);
	const selectedBuilding = $derived(
		buildings.find((building) => building.id === buildingSelection.id) ?? buildings[0] ?? null
	);

	$effect(() => {
		const current = buildingSelection.id;
		const items = buildingsQuery.data;
		const first = items?.at(0);
		if (first && !items?.some((building) => building.id === current)) {
			selectBuilding(first.id);
		}
	});

	$effect(() => {
		const status = auth.status;
		const role = auth.user?.role;
		if (status === 'guest') void goto('/login', { replaceState: true });
		if (status === 'authenticated' && role === 'resident') {
			void goto('/r', { replaceState: true });
		}
	});

	onMount(async () => {
		if (auth.status === 'unknown') await restoreSession();
		if (auth.status === 'authenticated' && !isManager()) {
			await goto('/r', { replaceState: true });
		}
	});

	async function signOut() {
		try {
			await logout();
		} catch {
			clearSession();
		} finally {
			queryClient.clear();
			setGuest();
		}
	}

	function changeBuilding(event: Event) {
		const id = (event.currentTarget as HTMLSelectElement).value;
		selectBuilding(id || null);
		void goto('/m', { replaceState: false });
	}
</script>

<svelte:head>
	<meta name="theme-color" content="#00695c" />
</svelte:head>

{#if auth.status === 'authenticated' && isManager()}
	<div class="min-h-dvh bg-surface lg:grid lg:grid-cols-[17rem_minmax(0,1fr)]">
		<aside class="no-print hidden border-e border-outline-variant/60 bg-white lg:flex lg:flex-col">
			<a href="/m" class="flex items-center gap-3 px-6 py-7" aria-label={fa.appName}>
				<span class="grid size-11 place-items-center rounded-2xl bg-primary text-on-primary">
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
					<span class="text-xs text-on-surface-variant">{fa.managerConsole}</span>
				</span>
			</a>

			{#if auth.user?.role === 'manager'}
				<div class="px-4 pb-5">
					<label
						class="mb-2 block px-2 text-xs font-semibold text-on-surface-variant"
						for="building-switcher"
					>
						{fa.switchBuilding}
					</label>
					{#if buildingsQuery.isPending}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{:else if selectedBuilding}
						<select
							id="building-switcher"
							class="input-base bg-surface text-sm font-semibold"
							value={selectedBuilding.id}
							onchange={changeBuilding}
						>
							{#each buildings as building (building.id)}
								<option value={building.id}>{building.name}</option>
							{/each}
						</select>
					{:else}
						<a class="btn-secondary w-full text-sm" href="/m/buildings">{fa.createFirstBuilding}</a>
					{/if}
				</div>
			{/if}

			<nav class="flex-1 space-y-1 px-4" aria-label={fa.mainNavigation}>
				<a class="nav-link" href="/m">
					<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
						<path
							d="m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-6v-7h-4v7H4a1 1 0 0 1-1-1V10Z"
							stroke="currentColor"
							stroke-width="1.7"
							stroke-linejoin="round"
						/>
					</svg>
					{fa.navDashboard}
				</a>
				{#if auth.user?.role === 'manager'}
					<a class="nav-link" href="/m/buildings">
						<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
							<path
								d="M4 21V5l8-3v19m0-12h8v12M2 21h20M8 7h1m-1 4h1m-1 4h1m7-3h1m-1 4h1"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
						{fa.navBuildings}
					</a>
					<a class="nav-link" href="/m/units">
						<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
							<path
								d="M3 21h18M5 21V7l7-4 7 4v14M9 21v-5h6v5M9 10h.01M15 10h.01M9 13h.01M15 13h.01"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
						{fa.navUnits}
					</a>
					<a class="nav-link" href="/m/persons">
						<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
							<path
								d="M16 20v-1.5a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4V20M9.5 10.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7ZM21 20v-1.5a4 4 0 0 0-3-3.87M16.5 3.63a4 4 0 0 1 0 7.75"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
						{fa.navPeople}
					</a>
					<a class="nav-link" href="/m/periods">
						<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
							<path
								d="M8 3v3m8-3v3M4 8h16M5 5h14a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1Z"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
						{fa.navPeriods}
					</a>
					<a class="nav-link" href="/m/invoices">
						<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
							<path
								d="M7 3h10a1 1 0 0 1 1 1v16l-3-2-3 2-3-2-3 2V4a1 1 0 0 1 1-1ZM9 8h6M9 12h6"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
						{fa.navInvoices}
					</a>
					<a class="nav-link" href="/m/payments">
						<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
							<path
								d="M2 8h20v10a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V8Zm0-3h20v3H2V5Zm4 9h4"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
						{fa.navPayments}
					</a>
				{:else}
					<div
						class="mx-2 rounded-xl bg-surface px-4 py-3 text-sm leading-6 text-on-surface-variant"
					>
						{fa.superadminHomeHint}
					</div>
				{/if}
			</nav>

			<div class="border-t border-outline-variant/50 p-4">
				<div class="flex items-center gap-3 rounded-2xl bg-surface p-3">
					<span
						class="grid size-10 shrink-0 place-items-center rounded-full bg-primary/10 font-bold text-primary"
					>
						{auth.user?.name.slice(0, 1) ?? 'ه'}
					</span>
					<span class="min-w-0 flex-1">
						<span class="block truncate text-sm font-semibold">{auth.user?.name}</span>
						<span class="block text-xs text-on-surface-variant">{roleLabel(auth.user?.role)}</span>
					</span>
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
			</div>
		</aside>

		<div class="min-w-0">
			<header
				class="no-print sticky top-0 z-20 flex min-h-16 items-center justify-between border-b border-outline-variant/50 bg-white/90 px-4 backdrop-blur sm:px-7 lg:px-10"
			>
				<div class="flex min-w-0 items-center gap-3">
					<span
						class="grid size-9 shrink-0 place-items-center rounded-xl bg-primary text-on-primary lg:hidden"
					>
						<svg viewBox="0 0 32 32" fill="none" class="size-5" aria-hidden="true">
							<path
								d="M5 27V13.5L16 5l11 8.5V27h-8v-8h-6v8H5Z"
								stroke="currentColor"
								stroke-width="2.2"
								stroke-linejoin="round"
							/>
						</svg>
					</span>
					<div class="min-w-0">
						<p class="truncate text-sm font-bold">{selectedBuilding?.name ?? fa.appName}</p>
						<p class="text-xs text-on-surface-variant">{roleLabel(auth.user?.role)}</p>
					</div>
				</div>
				<div class="flex items-center gap-2">
					{#if auth.user?.role === 'manager' && buildings.length > 1}
						<select
							class="input-base max-w-48 bg-white py-2 text-xs lg:hidden"
							value={selectedBuilding?.id ?? ''}
							onchange={changeBuilding}
							aria-label={fa.switchBuilding}
						>
							{#each buildings as building (building.id)}
								<option value={building.id}>{building.name}</option>
							{/each}
						</select>
					{/if}
					<button
						class="icon-button lg:hidden"
						type="button"
						aria-label={fa.logout}
						onclick={signOut}
					>
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
				class="mx-auto min-h-[calc(100dvh-4rem)] w-full max-w-[90rem] px-4 py-6 pb-24 sm:px-7 sm:py-8 lg:px-10 lg:pb-10"
			>
				{@render children()}
			</main>
		</div>

		<nav
			class="no-print fixed inset-x-0 bottom-0 z-30 flex gap-1 overflow-x-auto border-t border-outline-variant/60 bg-white/95 px-3 py-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] backdrop-blur lg:hidden"
			aria-label={fa.mainNavigation}
		>
			<a class="mobile-nav-link" href="/m" aria-current="page">
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
					><path
						d="m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-6v-7h-4v7H4a1 1 0 0 1-1-1V10Z"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linejoin="round"
					/></svg
				>
				{fa.navDashboard}
			</a>
			{#if auth.user?.role === 'manager'}
				<a class="mobile-nav-link" href="/m/buildings">
					<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
						><path
							d="M4 21V5l8-3v19m0-12h8v12M2 21h20M8 7h1m-1 4h1m-1 4h1m7-3h1m-1 4h1"
							stroke="currentColor"
							stroke-width="1.7"
							stroke-linecap="round"
							stroke-linejoin="round"
						/></svg
					>
					{fa.navBuildings}
				</a>
				<a class="mobile-nav-link" href="/m/units">
					<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
						><path
							d="M3 21h18M5 21V7l7-4 7 4v14M9 21v-5h6v5M9 10h.01M15 10h.01"
							stroke="currentColor"
							stroke-width="1.7"
							stroke-linecap="round"
							stroke-linejoin="round"
						/></svg
					>
					{fa.navUnits}
				</a>
				<a class="mobile-nav-link" href="/m/persons">
					<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
						><path
							d="M16 20v-1.5a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4V20M9.5 10.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z"
							stroke="currentColor"
							stroke-width="1.7"
							stroke-linecap="round"
							stroke-linejoin="round"
						/></svg
					>
					{fa.navPeople}
				</a>
				<a class="mobile-nav-link" href="/m/periods">
					<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
						><path
							d="M8 3v3m8-3v3M4 8h16M5 5h14a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1Z"
							stroke="currentColor"
							stroke-width="1.7"
							stroke-linecap="round"
							stroke-linejoin="round"
						/></svg
					>
					{fa.navPeriods}
				</a>
				<a class="mobile-nav-link" href="/m/invoices">
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
				<a class="mobile-nav-link" href="/m/payments">
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
			{/if}
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
			<span class="grid size-14 place-items-center rounded-2xl bg-primary text-on-primary">
				<span class="size-6 animate-spin rounded-full border-[3px] border-white/35 border-t-white"
				></span>
			</span>
			<p class="text-sm text-on-surface-variant">{fa.loading}</p>
		</div>
	</main>
{/if}
