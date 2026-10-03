<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listUnits } from '#lib/api/endpoints/units';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { unitStatusLabel, unitStatusTone } from '#lib/format/labels';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;

	const buildingId = $derived(buildingSelection.id ?? '');

	let q = $state('');
	let block = $state('');
	let floor = $state('');
	let status = $state('');
	let page = $state(1);

	/** Key inputs that change the result (research R5). */
	const filterKey = $derived(JSON.stringify({ q, block, floor, status, page }));

	const unitsQuery = createQuery(() => ({
		queryKey: qk.units.list(buildingId, filterKey),
		queryFn: () => listUnits(buildingId, { q, block, floor, status, page, page_size: PAGE_SIZE }),
		enabled: !!buildingId
	}));

	const units = $derived(unitsQuery.data?.items ?? []);
	const total = $derived(unitsQuery.data?.total ?? 0);

	function resetFilters() {
		q = '';
		block = '';
		floor = '';
		status = '';
		page = 1;
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navUnits}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.filterUnits}</p>
		</div>
		{#if buildingId}
			<a href="/m/units/new" class="btn-primary min-h-11 gap-2">
				<span class="text-xl leading-none" aria-hidden="true">＋</span>
				{fa.newUnit}
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
		<form
			class="grid gap-3 rounded-2xl border border-outline-variant/60 bg-white p-4 sm:grid-cols-2 lg:grid-cols-4"
			onsubmit={(event) => event.preventDefault()}
		>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="unit-q">{fa.search}</label>
				<input
					id="unit-q"
					class="input-base bg-white"
					bind:value={q}
					oninput={() => (page = 1)}
					placeholder={fa.searchPlaceholder}
				/>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="unit-block">{fa.block}</label>
				<input
					id="unit-block"
					class="input-base bg-white"
					bind:value={block}
					oninput={() => (page = 1)}
					placeholder={fa.blockPlaceholder}
				/>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="unit-floor">{fa.floor}</label>
				<input
					id="unit-floor"
					class="input-base bg-white"
					bind:value={floor}
					oninput={() => (page = 1)}
					inputmode="numeric"
					dir="ltr"
				/>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="unit-status">{fa.status}</label>
				<select
					id="unit-status"
					class="input-base bg-white"
					bind:value={status}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allStatuses}</option>
					<option value="active">{fa.statusActive}</option>
					<option value="vacant">{fa.statusVacant}</option>
					<option value="occupied">{fa.statusOccupied}</option>
					<option value="inactive">{fa.statusInactive}</option>
				</select>
			</div>
			<div class="flex justify-end sm:col-span-2 lg:col-span-4">
				<button class="btn-secondary min-h-10" type="button" onclick={resetFilters}>
					{fa.reset}
				</button>
			</div>
		</form>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if unitsQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3, 4, 5] as row (row)}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{/each}
				</div>
			{:else if unitsQuery.isError}
				<div class="p-8 text-center">
					<p class="text-lg font-bold">{fa.errorGeneric}</p>
					<p class="mt-2 text-sm text-on-surface-variant">
						{unitsQuery.error instanceof ApiError ? unitsQuery.error.message : fa.errorGeneric}
					</p>
					<button class="mt-5 btn-primary" type="button" onclick={() => void unitsQuery.refetch()}
						>{fa.retry}</button
					>
				</div>
			{:else if units.length}
				<div
					class="flex items-center gap-2 border-b border-outline-variant/50 px-5 py-3 text-sm text-on-surface-variant"
				>
					<span class="rounded-lg bg-primary/10 px-2 py-0.5 font-bold text-primary"
						>{toPersianDigits(String(total))}</span
					>
					{fa.resultsCount}
				</div>
				<div class="overflow-x-auto">
					<table class="w-full min-w-[44rem] text-sm">
						<thead class="bg-surface/70 text-xs text-on-surface-variant">
							<tr>
								<th class="px-5 py-3 text-start font-semibold">{fa.unitNumber}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.block}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.floor}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.area}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.status}</th>
								<th class="px-5 py-3 text-start font-semibold"
									><span class="sr-only">{fa.edit}</span></th
								>
							</tr>
						</thead>
						<tbody>
							{#each units as unit (unit.id)}
								<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
									<td class="px-5 py-3 font-bold">
										<a class="text-primary hover:underline" href={`/m/units/${unit.id}`}
											>{toPersianDigits(unit.number)}</a
										>
									</td>
									<td class="px-5 py-3">{unit.block ?? '—'}</td>
									<td class="px-5 py-3">{toPersianDigits(String(unit.floor))}</td>
									<td class="px-5 py-3">{toPersianDigits(String(unit.area_m2))}</td>
									<td class="px-5 py-3">
										<StatusChip
											tone={unitStatusTone(unit.status)}
											label={unitStatusLabel(unit.status)}
										/>
									</td>
									<td class="px-5 py-3 text-end">
										<a
											class="text-xs font-semibold text-primary hover:underline"
											href={`/m/units/${unit.id}`}>{fa.edit}</a
										>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<Pagination {page} pageSize={PAGE_SIZE} {total} onChange={(next) => (page = next)} />
			{:else}
				<div class="p-10 text-center">
					<p class="text-lg font-bold">{fa.emptyUnitsTitle}</p>
					<p class="mt-2 text-sm text-on-surface-variant">{fa.noUnitsHint}</p>
					<a class="mt-5 btn-primary" href="/m/units/new">{fa.newUnit}</a>
				</div>
			{/if}
		</section>
	{/if}
</div>
