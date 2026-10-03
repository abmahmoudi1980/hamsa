<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getPeriod } from '#lib/api/endpoints/billing';
	import PeriodForm from '#lib/components/PeriodForm.svelte';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const periodId = $derived(page.params.id ?? '');

	const periodQuery = createQuery(() => ({
		queryKey: qk.periods.detail(periodId),
		queryFn: () => getPeriod(periodId),
		enabled: !!periodId
	}));

	const period = $derived(periodQuery.data);
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		href={`/m/periods/${periodId}`}
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
		{fa.backToPeriodsList}
	</a>

	<header>
		<p class="text-sm font-semibold text-primary">{fa.navPeriods}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.editPeriod}</h1>
	</header>

	{#if periodQuery.isPending}
		<div class="h-72 animate-pulse rounded-3xl bg-white"></div>
	{:else if periodQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{periodQuery.error instanceof ApiError ? periodQuery.error.message : fa.errorGeneric}
			</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void periodQuery.refetch()}
				>{fa.retry}</button
			>
		</section>
	{:else if period && period.status !== 'draft'}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.onlyDraftEditable}</p>
			<a class="mt-5 btn-primary" href={`/m/periods/${periodId}`}>{fa.backToPeriodsList}</a>
		</section>
	{:else if period}
		<PeriodForm buildingId={period.building_id} periodId={period.id} initial={period} />
	{/if}
</div>
