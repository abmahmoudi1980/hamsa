<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		listBuildingMaintenance,
		MAINTENANCE_CATEGORIES,
		MAINTENANCE_PRIORITIES,
		MAINTENANCE_STATUSES
	} from '#lib/api/endpoints/maintenance';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { formatJalaliDate, fromIsoDate } from '#lib/format/jalali';
	import {
		maintenanceCategoryLabel,
		maintenanceStatusLabel,
		maintenanceStatusTone,
		priorityLabel,
		priorityTone
	} from '#lib/format/labels';
	import { qk } from '#lib/query/keys';
	import { toPersianDigits } from '#lib/format/digits';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;
	const buildingId = $derived(buildingSelection.id ?? '');
	let status = $state('');
	let priority = $state('');
	let category = $state('');
	let page = $state(1);
	const filterKey = $derived(JSON.stringify({ status, priority, category, page }));

	const requestsQuery = createQuery(() => ({
		queryKey: qk.maintenance.list(buildingId, filterKey),
		queryFn: () =>
			listBuildingMaintenance(buildingId, {
				status: status || undefined,
				priority: priority || undefined,
				category: category || undefined,
				page,
				page_size: PAGE_SIZE
			}),
		enabled: !!buildingId
	}));

	const requests = $derived(requestsQuery.data?.items ?? []);
	const total = $derived(requestsQuery.data?.total ?? 0);

	function dateLabel(value: string): string {
		const date = fromIsoDate(value);
		return date ? formatJalaliDate(date) : '—';
	}

	function resetFilters() {
		status = '';
		priority = '';
		category = '';
		page = 1;
	}
</script>

<div class="space-y-6">
	<header>
		<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navMaintenance}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.maintenanceListDescription}</p>
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
				<label class="block text-xs font-semibold" for="maintenance-status">{fa.changeStatus}</label
				>
				<select
					id="maintenance-status"
					class="input-base bg-white"
					bind:value={status}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allMaintenanceStatuses}</option>
					{#each MAINTENANCE_STATUSES as value (value)}<option {value}
							>{maintenanceStatusLabel(value)}</option
						>{/each}
				</select>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="maintenance-priority">{fa.priority}</label>
				<select
					id="maintenance-priority"
					class="input-base bg-white"
					bind:value={priority}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allPriorities}</option>
					{#each MAINTENANCE_PRIORITIES as value (value)}<option {value}
							>{priorityLabel(value)}</option
						>{/each}
				</select>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="maintenance-category">{fa.category}</label>
				<select
					id="maintenance-category"
					class="input-base bg-white"
					bind:value={category}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allCategories}</option>
					{#each MAINTENANCE_CATEGORIES as value (value)}<option {value}
							>{maintenanceCategoryLabel(value)}</option
						>{/each}
				</select>
			</div>
			<div class="flex items-end justify-end">
				<button class="btn-secondary min-h-10" type="button" onclick={resetFilters}
					>{fa.reset}</button
				>
			</div>
		</form>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if requestsQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3, 4, 5] as row (row)}<div
							class="h-12 animate-pulse rounded-xl bg-surface-high"
						></div>{/each}
				</div>
			{:else if requestsQuery.isError}
				<div class="p-8 text-center">
					<p class="text-lg font-bold">{fa.errorGeneric}</p>
					<p class="mt-2 text-sm text-on-surface-variant">
						{requestsQuery.error instanceof ApiError
							? requestsQuery.error.message
							: fa.errorGeneric}
					</p>
					<button
						class="mt-5 btn-primary"
						type="button"
						onclick={() => void requestsQuery.refetch()}>{fa.retry}</button
					>
				</div>
			{:else if requests.length}
				<div
					class="flex items-center gap-2 border-b border-outline-variant/50 px-5 py-3 text-sm text-on-surface-variant"
				>
					<span class="rounded-lg bg-primary/10 px-2 py-0.5 font-bold text-primary"
						>{toPersianDigits(String(total))}</span
					>{fa.resultsCount}
				</div>
				<ul class="divide-y divide-outline-variant/40">
					{#each requests as request (request.id)}
						<li
							class="flex flex-wrap items-center justify-between gap-4 px-5 py-4 hover:bg-surface/60"
						>
							<div class="min-w-0 flex-1">
								<a
									class="font-semibold text-primary hover:underline"
									href={`/m/maintenance/${request.id}`}>{request.title}</a
								>
								<p class="mt-1 text-xs text-on-surface-variant">
									{maintenanceCategoryLabel(request.category)} · {dateLabel(request.created_at)} · {priorityLabel(
										request.priority
									)}
								</p>
							</div>
							<div class="flex items-center gap-3">
								<StatusChip
									tone={priorityTone(request.priority)}
									label={priorityLabel(request.priority)}
								/>
								<StatusChip
									tone={maintenanceStatusTone(request.status)}
									label={maintenanceStatusLabel(request.status)}
								/>
							</div>
						</li>
					{/each}
				</ul>
				<Pagination {page} pageSize={PAGE_SIZE} {total} onChange={(next) => (page = next)} />
			{:else}
				<p class="p-10 text-center text-sm text-on-surface-variant">{fa.noMaintenanceRequests}</p>
			{/if}
		</section>
	{/if}
</div>
