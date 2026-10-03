<script lang="ts">
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		getBuilding,
		grantManager,
		listManagers,
		revokeManager
	} from '#lib/api/endpoints/buildings';
	import { auth } from '#lib/auth/auth.svelte';
	import { formatJalaliDate, fromIsoDate } from '#lib/format/jalali';
	import { roleLabel } from '#lib/format/labels';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const buildingId = $derived(page.params.id ?? '');

	const buildingQuery = createQuery(() => ({
		queryKey: qk.buildings.detail(buildingId),
		queryFn: () => getBuilding(buildingId),
		enabled: !!buildingId
	}));

	const managersQuery = createQuery(() => ({
		queryKey: qk.buildings.managers(buildingId),
		queryFn: () => listManagers(buildingId),
		enabled: !!buildingId
	}));

	const managers = $derived(managersQuery.data ?? []);

	let phone = $state('');
	let addError = $state('');
	let removeError = $state('');

	const grantMutation = createMutation(() => ({
		mutationFn: (value: string) => grantManager(buildingId, value),
		onSuccess: async () => {
			phone = '';
			await queryClient.invalidateQueries({ queryKey: qk.buildings.managers(buildingId) });
		}
	}));

	async function submitAdd(event: SubmitEvent) {
		event.preventDefault();
		addError = '';
		if (phone.trim() === '') {
			addError = fa.required;
			return;
		}
		try {
			await grantMutation.mutateAsync(phone.trim());
		} catch (error) {
			addError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
	}

	const revokeMutation = createMutation(() => ({
		mutationFn: (userId: string) => revokeManager(buildingId, userId),
		onSuccess: async () => {
			removeError = '';
			await queryClient.invalidateQueries({ queryKey: qk.buildings.managers(buildingId) });
		}
	}));

	async function remove(userId: string) {
		if (!confirm(fa.removeManagerConfirm)) return;
		removeError = '';
		try {
			await revokeMutation.mutateAsync(userId);
		} catch (error) {
			// The last-manager and self-removal guards surface verbatim.
			removeError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
	}

	function grantedLabel(iso: string): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		href="/m/buildings"
		class="inline-flex items-center gap-2 text-sm font-semibold text-on-surface-variant transition-colors hover:text-primary"
	>
		<svg viewBox="0 0 20 20" fill="none" class="size-4 -scale-x-100" aria-hidden="true"
			><path
				d="M16 10H4m0 0 4.5-4.5M4 10l4.5 4.5"
				stroke="currentColor"
				stroke-width="1.8"
				stroke-linecap="round"
				stroke-linejoin="round"
			/></svg
		>
		{fa.backToBuildings}
	</a>

	<header>
		<p class="text-sm font-semibold text-primary">{buildingQuery.data?.name ?? fa.navBuildings}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.managers}</h1>
		<p class="mt-2 text-sm leading-6 text-on-surface-variant">{fa.managersDescription}</p>
	</header>

	<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
		{#if managersQuery.isPending}
			<div class="space-y-2 p-5">
				{#each [1, 2] as row (row)}
					<div class="h-14 animate-pulse rounded-xl bg-surface-high"></div>
				{/each}
			</div>
		{:else if managersQuery.isError}
			<div class="p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{managersQuery.error instanceof ApiError ? managersQuery.error.message : fa.errorGeneric}
				</p>
			</div>
		{:else if managers.length}
			<ul class="divide-y divide-outline-variant/40">
				{#each managers as manager (manager.user_id)}
					<li class="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
						<div class="flex items-center gap-3">
							<span
								class="grid size-10 place-items-center rounded-full bg-primary/10 font-bold text-primary"
							>
								{manager.name.slice(0, 1) || 'م'}
							</span>
							<div>
								<p class="font-semibold">
									{manager.name}
									{#if manager.user_id === auth.user?.id}
										<span class="text-xs font-normal text-primary">(شما)</span>
									{/if}
								</p>
								<p class="text-xs text-on-surface-variant" dir="ltr">{manager.phone}</p>
							</div>
						</div>
						<div class="flex items-center gap-3 text-xs text-on-surface-variant">
							<span>{roleLabel(manager.role)}</span>
							<span>{grantedLabel(manager.granted_at)}</span>
							<button
								class="font-semibold text-danger hover:underline"
								type="button"
								disabled={revokeMutation.isPending}
								onclick={() => void remove(manager.user_id)}>{fa.removeManager}</button
							>
						</div>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="p-8 text-center text-sm text-on-surface-variant">{fa.noManagers}</p>
		{/if}
	</section>

	{#if removeError}
		<p
			class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm text-danger"
			role="alert"
		>
			{removeError}
		</p>
	{/if}

	<form
		class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white"
		onsubmit={submitAdd}
		novalidate
	>
		<div class="space-y-4 p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.addManager}</h2>
			<p class="text-sm text-on-surface-variant">{fa.addManagerHint}</p>
			<div class="max-w-sm space-y-2">
				<label class="block text-sm font-semibold" for="manager-phone">{fa.managerPhone}</label>
				<input
					id="manager-phone"
					class={`input-base bg-white ${addError ? 'border-danger' : ''}`}
					bind:value={phone}
					inputmode="tel"
					dir="ltr"
					placeholder={fa.phonePlaceholder}
				/>
				{#if addError}<p class="text-xs text-danger" aria-live="polite">{addError}</p>{/if}
			</div>
			<button class="btn-primary min-h-10" type="submit" disabled={grantMutation.isPending}>
				{grantMutation.isPending ? fa.saving : fa.add}
			</button>
		</div>
	</form>
</div>
