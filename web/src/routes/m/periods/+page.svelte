<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listPeriods } from '#lib/api/endpoints/billing';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import { periodStatusLabel, periodStatusTone } from '#lib/format/labels';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const buildingId = $derived(buildingSelection.id ?? '');

	const periodsQuery = createQuery(() => ({
		queryKey: qk.periods.list(buildingId),
		queryFn: () => listPeriods(buildingId),
		enabled: !!buildingId
	}));

	const periods = $derived(periodsQuery.data ?? []);

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navPeriods}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.periodListDescription}</p>
		</div>
		{#if buildingId}
			<a href="/m/periods/new" class="btn-primary min-h-11 gap-2">
				<span class="text-xl leading-none" aria-hidden="true">＋</span>
				{fa.newPeriod}
			</a>
		{/if}
	</header>

	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.noBuildingsHint}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else}
		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if periodsQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3, 4] as row (row)}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{/each}
				</div>
			{:else if periodsQuery.isError}
				<div class="p-8 text-center">
					<p class="text-lg font-bold">{fa.errorGeneric}</p>
					<p class="mt-2 text-sm text-on-surface-variant">
						{periodsQuery.error instanceof ApiError ? periodsQuery.error.message : fa.errorGeneric}
					</p>
					<button class="mt-5 btn-primary" type="button" onclick={() => void periodsQuery.refetch()}
						>{fa.retry}</button
					>
				</div>
			{:else if periods.length}
				<div class="overflow-x-auto">
					<table class="w-full min-w-[46rem] text-sm">
						<thead class="bg-surface/70 text-xs text-on-surface-variant">
							<tr>
								<th class="px-5 py-3 text-start font-semibold">{fa.periodTitle}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.startDatePeriod}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.endDatePeriod}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.dueDate}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.status}</th>
								<th class="px-5 py-3 text-start font-semibold">
									<span class="sr-only">{fa.periodDetail}</span>
								</th>
							</tr>
						</thead>
						<tbody>
							{#each periods as period (period.id)}
								<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
									<td class="px-5 py-3 font-bold">
										<a class="text-primary hover:underline" href={`/m/periods/${period.id}`}
											>{period.title}</a
										>
									</td>
									<td class="px-5 py-3">{dateLabel(period.start_date)}</td>
									<td class="px-5 py-3">{dateLabel(period.end_date)}</td>
									<td class="px-5 py-3">{dateLabel(period.due_date)}</td>
									<td class="px-5 py-3">
										<StatusChip
											tone={periodStatusTone(period.status)}
											label={periodStatusLabel(period.status)}
										/>
									</td>
									<td class="px-5 py-3 text-end">
										<a
											class="text-xs font-semibold text-primary hover:underline"
											href={`/m/periods/${period.id}`}>{fa.periodDetail}</a
										>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="p-10 text-center">
					<p class="text-lg font-bold">{fa.noPeriods}</p>
					<p class="mt-2 text-sm text-on-surface-variant">{fa.noPeriodsHint}</p>
					<a class="mt-5 btn-primary" href="/m/periods/new">{fa.newPeriod}</a>
				</div>
			{/if}
		</section>
	{/if}
</div>
