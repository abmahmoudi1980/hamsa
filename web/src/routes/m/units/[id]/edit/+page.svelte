<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getUnit } from '#lib/api/endpoints/units';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import UnitForm from '#lib/components/UnitForm.svelte';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const unitId = $derived(page.params.id ?? '');
	const buildingId = $derived(buildingSelection.id ?? '');

	const unitQuery = createQuery(() => ({
		queryKey: qk.units.detail(unitId),
		queryFn: () => getUnit(unitId),
		enabled: !!unitId
	}));
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		href={`/m/units/${unitId}`}
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
		{fa.backToUnit}
	</a>

	<header>
		<p class="text-sm font-semibold text-primary">{fa.navUnits}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.editUnit}</h1>
	</header>

	{#if unitQuery.isPending}
		<div class="h-96 animate-pulse rounded-3xl bg-white"></div>
	{:else if unitQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{unitQuery.error instanceof ApiError ? unitQuery.error.message : fa.errorGeneric}
			</p>
		</section>
	{:else if unitQuery.data}
		<UnitForm {buildingId} {unitId} initial={unitQuery.data} />
	{/if}
</div>
