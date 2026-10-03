<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getUnitBalance } from '#lib/api/endpoints/payments';
	import { getUnit } from '#lib/api/endpoints/units';
	import BalanceBreakdown from '#lib/components/BalanceBreakdown.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const unitId = $derived(page.params.id ?? '');

	const unitQuery = createQuery(() => ({
		queryKey: qk.units.detail(unitId),
		queryFn: () => getUnit(unitId),
		enabled: !!unitId
	}));

	const balanceQuery = createQuery(() => ({
		queryKey: qk.balances.unit(unitId),
		queryFn: () => getUnitBalance(unitId),
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
		<p class="text-sm font-semibold text-primary">{fa.balance}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">
			{fa.unit}
			{#if unitQuery.data}
				{toPersianDigits(unitQuery.data.number)}
			{:else}
				…
			{/if}
		</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.unitBalanceDescription}</p>
	</header>

	{#if balanceQuery.isPending}
		<div class="h-72 animate-pulse rounded-3xl bg-white"></div>
	{:else if balanceQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{balanceQuery.error instanceof ApiError ? balanceQuery.error.message : fa.errorGeneric}
			</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void balanceQuery.refetch()}
				>{fa.retry}</button
			>
		</section>
	{:else if balanceQuery.data}
		<BalanceBreakdown balance={balanceQuery.data} />
	{/if}
</div>
