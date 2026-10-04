<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listMyMaintenance } from '#lib/api/endpoints/maintenance';
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
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;
	let pageNumber = $state(1);
	const filters = $derived(JSON.stringify({ pageNumber }));
	const requestsQuery = createQuery(() => ({
		queryKey: qk.maintenance.mine(filters),
		queryFn: () => listMyMaintenance(pageNumber, PAGE_SIZE)
	}));
	const requests = $derived(requestsQuery.data?.items ?? []);
	const total = $derived(requestsQuery.data?.total ?? 0);

	function dateLabel(value: string): string {
		const date = fromIsoDate(value);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.navHome}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.myRequests}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.maintenanceSubmitDescription}</p>
		</div>
		<a class="btn-primary min-h-11" href="/r/maintenance/new">{fa.newMaintenanceRequest}</a>
	</header>

	<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
		{#if requestsQuery.isPending}
			<div class="space-y-2 p-5">
				{#each [1, 2, 3] as row (row)}<div
						class="h-20 animate-pulse rounded-xl bg-surface-high"
					></div>{/each}
			</div>
		{:else if requestsQuery.isError}
			<div class="p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{requestsQuery.error instanceof ApiError ? requestsQuery.error.message : fa.errorGeneric}
				</p>
				<button class="mt-5 btn-primary" type="button" onclick={() => void requestsQuery.refetch()}
					>{fa.retry}</button
				>
			</div>
		{:else if requests.length}
			<ul class="divide-y divide-outline-variant/40">
				{#each requests as request (request.id)}
					<li>
						<a
							class="block space-y-3 px-5 py-4 transition-colors hover:bg-surface/70"
							href={`/r/maintenance/${request.id}`}
						>
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									<p class="truncate font-bold">{request.title}</p>
									<p class="mt-1 text-xs text-on-surface-variant">
										{maintenanceCategoryLabel(request.category)} · {dateLabel(request.created_at)}
									</p>
								</div>
								<StatusChip
									tone={maintenanceStatusTone(request.status)}
									label={maintenanceStatusLabel(request.status)}
								/>
							</div>
							<div class="flex items-center gap-2">
								<StatusChip
									tone={priorityTone(request.priority)}
									label={priorityLabel(request.priority)}
								/>
							</div>
						</a>
					</li>
				{/each}
			</ul>
			<Pagination
				page={pageNumber}
				pageSize={PAGE_SIZE}
				{total}
				onChange={(next) => (pageNumber = next)}
			/>
		{:else}
			<div class="p-10 text-center">
				<p class="text-lg font-bold">{fa.noMaintenanceRequests}</p>
				<a class="mt-5 btn-primary" href="/r/maintenance/new">{fa.newMaintenanceRequest}</a>
			</div>
		{/if}
	</section>
</div>
