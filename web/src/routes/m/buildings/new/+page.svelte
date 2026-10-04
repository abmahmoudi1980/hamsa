<script lang="ts">
	import { goto } from '$app/navigation';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { z } from 'zod';
	import { ApiError } from '#lib/api/apiError';
	import { createBuilding, type BuildingInput } from '#lib/api/endpoints/buildings';
	import { fa } from '#i18n/fa';
	import { latinDigitsOnly } from '#lib/format/digits';
	import { qk } from '#lib/query/keys';
	import { selectBuilding } from '#lib/building/selection.svelte';

	const queryClient = useQueryClient();
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
	let nameError = $state('');
	let formError = $state('');

	const mutation = createMutation(() => ({
		mutationFn: createBuilding,
		onSuccess: async (building) => {
			await queryClient.invalidateQueries({ queryKey: qk.buildings.list() });
			selectBuilding(building.id);
			await goto('/m', { replace: true });
		}
	}));

	function countValue(value: string): number {
		const digits = latinDigitsOnly(value);
		return digits ? Number(digits) : Number.NaN;
	}

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
			unit_count: parsed.data.unit_count
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
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.createBuildingTitle}</h1>
		<p class="mt-2 text-sm leading-6 text-on-surface-variant">{fa.createBuildingDescription}</p>
	</header>

	<form
		class="overflow-hidden rounded-3xl border border-outline-variant/60 bg-white shadow-sm"
		onsubmit={submit}
		novalidate
	>
		<div class="space-y-6 p-5 sm:p-8">
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="building-name">{fa.buildingName}</label>
				<input
					id="building-name"
					class={`input-base bg-white ${nameError ? 'border-danger' : ''}`}
					bind:value={name}
					maxlength="120"
					placeholder={fa.buildingNamePlaceholder}
					aria-invalid={nameError ? 'true' : undefined}
					aria-describedby={nameError ? 'name-error' : undefined}
				/>
				{#if nameError}
					<p id="name-error" class="text-xs text-danger" aria-live="polite">{nameError}</p>
				{/if}
			</div>

			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="building-address"
					>{fa.buildingAddress}</label
				>
				<input
					id="building-address"
					class="input-base bg-white"
					bind:value={address}
					maxlength="500"
					placeholder={fa.addressPlaceholder}
				/>
			</div>

			<div class="grid gap-4 sm:grid-cols-3">
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
			</div>

			{#if formError}
				<p
					class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
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
</div>
