<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { z } from 'zod';
	import { ApiError } from '#lib/api/apiError';
	import { getBuilding, updateBuilding, type BuildingInput } from '#lib/api/endpoints/buildings';
	import { latinDigitsOnly } from '#lib/format/digits';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const buildingId = $derived(page.params.id ?? '');

	const buildingQuery = createQuery(() => ({
		queryKey: qk.buildings.detail(buildingId),
		queryFn: () => getBuilding(buildingId),
		enabled: !!buildingId
	}));

	const formSchema = z.object({
		name: z.string().trim().min(1).max(120),
		address: z.string().trim().max(500),
		block_count: z.number().int().min(0).max(1000),
		floor_count: z.number().int().min(0).max(1000),
		unit_count: z.number().int().min(1).max(10000)
	});

	let name = $state('');
	let address = $state('');
	let blockCount = $state('1');
	let floorCount = $state('1');
	let unitCount = $state('1');
	let builtYear = $state('');
	let managerPhone = $state('');
	let emergencyPhone = $state('');
	let notes = $state('');
	let nameError = $state('');
	let formError = $state('');

	let seeded = false;
	$effect(() => {
		const building = buildingQuery.data;
		if (building && !seeded) {
			name = building.name;
			address = building.address ?? '';
			blockCount = String(building.block_count);
			floorCount = String(building.floor_count);
			unitCount = String(building.unit_count);
			builtYear = building.built_year ? String(building.built_year) : '';
			managerPhone = building.manager_phone ?? '';
			emergencyPhone = building.emergency_phone ?? '';
			notes = building.notes ?? '';
			seeded = true;
		}
	});

	function countValue(value: string): number {
		const digits = latinDigitsOnly(value);
		return digits ? Number(digits) : Number.NaN;
	}

	const mutation = createMutation(() => ({
		mutationFn: (input: BuildingInput) => updateBuilding(buildingId, input),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: qk.buildings.list() });
			await queryClient.invalidateQueries({ queryKey: qk.buildings.detail(buildingId) });
			await goto('/m/buildings', { replace: true });
		}
	}));

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		nameError = '';
		formError = '';

		const parsed = formSchema.safeParse({
			name,
			address,
			block_count: countValue(blockCount),
			floor_count: countValue(floorCount),
			unit_count: countValue(unitCount)
		});

		if (!parsed.success) {
			nameError = parsed.error.issues.some((issue) => issue.path[0] === 'name')
				? fa.buildingNameRequired
				: '';
			formError = parsed.error.issues.some((issue) => issue.path[0] !== 'name')
				? fa.buildingCountsInvalid
				: '';
			return;
		}

		const input: BuildingInput = {
			name: parsed.data.name,
			address: parsed.data.address || undefined,
			block_count: parsed.data.block_count,
			floor_count: parsed.data.floor_count,
			unit_count: parsed.data.unit_count,
			built_year: builtYear ? countValue(builtYear) : undefined,
			manager_phone: managerPhone || undefined,
			emergency_phone: emergencyPhone || undefined,
			notes: notes || undefined
		};

		try {
			await mutation.mutateAsync(input);
		} catch (error) {
			formError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
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
		<p class="text-sm font-semibold text-primary">{fa.navBuildings}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.editBuilding}</h1>
	</header>

	{#if buildingQuery.isPending}
		<div class="h-96 animate-pulse rounded-3xl bg-white"></div>
	{:else if buildingQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{buildingQuery.error instanceof ApiError ? buildingQuery.error.message : fa.errorGeneric}
			</p>
		</section>
	{:else}
		<form
			class="overflow-hidden rounded-3xl border border-outline-variant/60 bg-white shadow-sm"
			onsubmit={submit}
			novalidate
		>
			<div class="grid gap-5 p-5 sm:grid-cols-2 sm:p-8">
				<div class="space-y-2 sm:col-span-2">
					<label class="block text-sm font-semibold" for="building-name"
						>{fa.buildingName}<span class="text-danger" aria-hidden="true">*</span></label
					>
					<input
						id="building-name"
						class={`input-base bg-white ${nameError ? 'border-danger' : ''}`}
						bind:value={name}
						maxlength="120"
						placeholder={fa.buildingNamePlaceholder}
						aria-invalid={nameError ? 'true' : undefined}
					/>
					{#if nameError}<p class="text-xs text-danger">{nameError}</p>{/if}
				</div>

				<div class="space-y-2 sm:col-span-2">
					<label class="block text-sm font-semibold" for="building-address"
						>{fa.buildingAddress}</label
					>
					<input
						id="building-address"
						class="input-base bg-white"
						bind:value={address}
						maxlength="500"
					/>
				</div>

				<div class="space-y-2">
					<label class="block text-sm font-semibold" for="block-count">{fa.blockCount}</label>
					<input
						id="block-count"
						class="input-base bg-white"
						bind:value={blockCount}
						inputmode="numeric"
						dir="ltr"
					/>
				</div>
				<div class="space-y-2">
					<label class="block text-sm font-semibold" for="floor-count">{fa.floorCount}</label>
					<input
						id="floor-count"
						class="input-base bg-white"
						bind:value={floorCount}
						inputmode="numeric"
						dir="ltr"
					/>
				</div>
				<div class="space-y-2">
					<label class="block text-sm font-semibold" for="unit-count">{fa.unitCount}</label>
					<input
						id="unit-count"
						class="input-base bg-white"
						bind:value={unitCount}
						inputmode="numeric"
						dir="ltr"
					/>
				</div>
				<div class="space-y-2">
					<label class="block text-sm font-semibold" for="built-year">{fa.yearBuilt}</label>
					<input
						id="built-year"
						class="input-base bg-white"
						bind:value={builtYear}
						inputmode="numeric"
						dir="ltr"
					/>
				</div>
				<div class="space-y-2">
					<label class="block text-sm font-semibold" for="manager-phone">{fa.contactPhone}</label>
					<input
						id="manager-phone"
						class="input-base bg-white"
						bind:value={managerPhone}
						inputmode="tel"
						dir="ltr"
					/>
				</div>
				<div class="space-y-2">
					<label class="block text-sm font-semibold" for="emergency-phone"
						>{fa.emergencyPhone}</label
					>
					<input
						id="emergency-phone"
						class="input-base bg-white"
						bind:value={emergencyPhone}
						inputmode="tel"
						dir="ltr"
					/>
				</div>
				<div class="space-y-2 sm:col-span-2">
					<label class="block text-sm font-semibold" for="building-notes">{fa.description}</label>
					<textarea id="building-notes" class="min-h-24 input-base bg-white" bind:value={notes}
					></textarea>
				</div>

				{#if formError}
					<p
						class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger sm:col-span-2"
						role="alert"
					>
						{formError}
					</p>
				{/if}
			</div>

			<div
				class="flex flex-col-reverse gap-3 border-t border-outline-variant/50 bg-surface/70 p-5 sm:flex-row sm:justify-end sm:px-8"
			>
				<a class="btn-secondary" href="/m/buildings">{fa.cancel}</a>
				<button class="btn-primary" type="submit" disabled={mutation.isPending}>
					{mutation.isPending ? fa.saving : fa.save}
				</button>
			</div>
		</form>
	{/if}
</div>
