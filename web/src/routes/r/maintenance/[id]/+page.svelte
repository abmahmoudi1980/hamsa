<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getMyMaintenanceRequest } from '#lib/api/endpoints/maintenance';
	import AttachmentPreview from '#lib/components/AttachmentPreview.svelte';
	import MaintenanceTimeline from '#lib/components/MaintenanceTimeline.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import {
		maintenanceCategoryLabel,
		maintenanceStatusLabel,
		maintenanceStatusTone,
		priorityLabel,
		priorityTone
	} from '#lib/format/labels';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const requestId = $derived(page.params.id ?? '');
	const requestQuery = createQuery(() => ({
		queryKey: qk.maintenance.detail(requestId),
		queryFn: () => getMyMaintenanceRequest(requestId),
		enabled: !!requestId
	}));
</script>

{#if requestQuery.isPending}
	<div class="mx-auto max-w-3xl space-y-4" aria-label={fa.loading}>
		<div class="h-10 animate-pulse rounded-xl bg-surface-high"></div>
		<div class="h-64 animate-pulse rounded-2xl bg-white"></div>
	</div>
{:else if requestQuery.isError || !requestQuery.data}
	<section class="mx-auto max-w-3xl card p-8 text-center">
		<p class="text-lg font-bold">{fa.errorGeneric}</p>
		<p class="mt-2 text-sm text-on-surface-variant">
			{requestQuery.error instanceof ApiError ? requestQuery.error.message : fa.errorGeneric}
		</p>
		<a class="mt-5 btn-secondary" href="/r/maintenance">{fa.maintenanceBackToList}</a>
	</section>
{:else}
	{@const request = requestQuery.data}
	<div class="mx-auto max-w-3xl space-y-6">
		<a
			class="text-sm font-semibold text-on-surface-variant hover:text-primary"
			href="/r/maintenance">{fa.maintenanceBackToList}</a
		>
		<header class="flex flex-wrap items-start justify-between gap-4">
			<div class="min-w-0">
				<p class="text-sm font-semibold text-primary">
					{maintenanceCategoryLabel(request.category)}
				</p>
				<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{request.title}</h1>
				<p class="mt-2 text-sm text-on-surface-variant">{request.location || fa.location}</p>
			</div>
			<div class="flex gap-2">
				<StatusChip
					tone={priorityTone(request.priority)}
					label={priorityLabel(request.priority)}
				/><StatusChip
					tone={maintenanceStatusTone(request.status)}
					label={maintenanceStatusLabel(request.status)}
				/>
			</div>
		</header>
		<section class="space-y-4 card p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.maintenanceDetails}</h2>
			{#if request.description}<p class="text-sm leading-7 whitespace-pre-wrap">
					{request.description}
				</p>{/if}
			{#if request.photo_file}<div class="space-y-2">
					<p class="text-sm font-semibold">{fa.photo}</p>
					<AttachmentPreview fileRef={request.photo_file} label={fa.photo} />
				</div>{/if}
			{#if request.notes}<div class="rounded-xl bg-surface p-4">
					<p class="text-xs font-semibold text-on-surface-variant">{fa.note}</p>
					<p class="mt-2 text-sm leading-7 whitespace-pre-wrap">{request.notes}</p>
				</div>{/if}
		</section>
		<MaintenanceTimeline {request} />
	</div>
{/if}
