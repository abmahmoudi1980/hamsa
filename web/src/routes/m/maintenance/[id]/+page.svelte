<script lang="ts">
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		getMaintenanceRequest,
		MAINTENANCE_STATUSES,
		updateMaintenanceRequest
	} from '#lib/api/endpoints/maintenance';
	import { listPersons } from '#lib/api/endpoints/persons';
	import AttachmentPreview from '#lib/components/AttachmentPreview.svelte';
	import MaintenanceTimeline from '#lib/components/MaintenanceTimeline.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { parseTomanInput, tomanToNumber } from '#lib/format/money';
	import {
		maintenanceCategoryLabel,
		maintenanceStatusLabel,
		maintenanceStatusTone,
		priorityLabel,
		priorityTone
	} from '#lib/format/labels';
	import { nextMaintenanceStatuses } from '#lib/maintenance/workflow';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const requestId = $derived(page.params.id ?? '');
	const requestQuery = createQuery(() => ({
		queryKey: qk.maintenance.detail(requestId),
		queryFn: () => getMaintenanceRequest(requestId),
		enabled: !!requestId
	}));
	const buildingId = $derived(requestQuery.data?.building_id ?? '');
	const personsQuery = createQuery(() => ({
		queryKey: ['persons', buildingId, 'maintenance-assignees'],
		queryFn: () => listPersons(buildingId, { page: 1, page_size: 100 }),
		enabled: !!buildingId
	}));

	let statusChoice = $state('');
	let assigneePersonId = $state('');
	let recordedCost = $state('');
	let notes = $state('');
	let loadedRequestId = '';
	let error = $state('');

	$effect(() => {
		const request = requestQuery.data;
		if (!request || loadedRequestId === request.id) return;
		loadedRequestId = request.id;
		assigneePersonId = request.assignee_person_id ?? '';
		recordedCost = request.recorded_cost === undefined ? '' : String(request.recorded_cost);
		notes = request.notes ?? '';
	});

	const choices = $derived(
		requestQuery.data ? nextMaintenanceStatuses(requestQuery.data.status) : MAINTENANCE_STATUSES
	);

	const updateMutation = createMutation(() => ({
		mutationFn: () => {
			const cost = recordedCost.trim() === '' ? null : parseTomanInput(recordedCost);
			if (cost !== null && tomanToNumber(cost) < 0) throw new RangeError(fa.invalidAmount);
			return updateMaintenanceRequest(requestId, {
				status: statusChoice || undefined,
				assignee_person_id: assigneePersonId || '00000000-0000-0000-0000-000000000000',
				recorded_cost:
					cost === null
						? requestQuery.data?.recorded_cost === undefined
							? undefined
							: ''
						: tomanToNumber(cost),
				notes: notes.trim()
			});
		},
		onSuccess: async (updated) => {
			statusChoice = '';
			assigneePersonId = updated.assignee_person_id ?? '';
			recordedCost = updated.recorded_cost === undefined ? '' : String(updated.recorded_cost);
			notes = updated.notes ?? '';
			error = '';
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.maintenance.detail(requestId) }),
				queryClient.invalidateQueries({ queryKey: ['maintenance', updated.building_id] }),
				queryClient.invalidateQueries({ queryKey: ['maintenance', 'me'] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(updated.building_id) }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.resident() }),
				queryClient.invalidateQueries({ queryKey: ['notifications'] })
			]);
		}
	}));

	async function save(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		try {
			await updateMutation.mutateAsync();
		} catch (cause) {
			error =
				cause instanceof ApiError
					? cause.message
					: cause instanceof Error
						? cause.message
						: fa.errorGeneric;
		}
	}
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
		<a class="mt-5 btn-secondary" href="/m/maintenance">{fa.back}</a>
	</section>
{:else}
	{@const request = requestQuery.data}
	<div class="mx-auto max-w-3xl space-y-6">
		<a
			class="text-sm font-semibold text-on-surface-variant hover:text-primary"
			href="/m/maintenance">{fa.back}</a
		>
		<header class="flex flex-wrap items-start justify-between gap-4">
			<div class="min-w-0">
				<p class="text-sm font-semibold text-primary">
					{maintenanceCategoryLabel(request.category)}
				</p>
				<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{request.title}</h1>
				<p class="mt-2 text-sm text-on-surface-variant">
					{request.location || fa.location} · {priorityLabel(request.priority)}
				</p>
			</div>
			<div class="flex gap-2">
				<StatusChip tone={priorityTone(request.priority)} label={priorityLabel(request.priority)} />
				<StatusChip
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
		</section>

		<MaintenanceTimeline {request} />

		<section class="card p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.changeStatus}</h2>
			<form class="mt-4 space-y-4" onsubmit={save}>
				<div class="grid gap-4 sm:grid-cols-2">
					<div class="space-y-2">
						<label class="block text-sm font-semibold" for="request-status">{fa.changeStatus}</label
						>
						<select id="request-status" class="input-base bg-white" bind:value={statusChoice}>
							<option value="">{maintenanceStatusLabel(request.status)}</option>
							{#each choices as value (value)}<option {value}
									>{maintenanceStatusLabel(value)}</option
								>{/each}
						</select>
					</div>
					<div class="space-y-2">
						<label class="block text-sm font-semibold" for="request-assignee">{fa.assignee}</label>
						<select id="request-assignee" class="input-base bg-white" bind:value={assigneePersonId}>
							<option value="">{fa.noAssignee}</option>
							{#each personsQuery.data?.items ?? [] as person (person.id)}<option value={person.id}
									>{person.full_name}</option
								>{/each}
						</select>
					</div>
					<div class="space-y-2">
						<label class="block text-sm font-semibold" for="request-cost">{fa.recordedCost}</label>
						<input
							id="request-cost"
							class="input-base bg-white"
							bind:value={recordedCost}
							inputmode="numeric"
							dir="ltr"
							placeholder={fa.maintenanceCostHint}
						/>
						<p class="text-xs text-on-surface-variant">{fa.maintenanceCostHint}</p>
					</div>
					<div class="space-y-2">
						<label class="block text-sm font-semibold" for="request-notes">{fa.note}</label>
						<textarea id="request-notes" class="min-h-24 input-base bg-white" bind:value={notes}
						></textarea>
					</div>
				</div>
				{#if error}<p class="text-sm text-danger" role="alert">{error}</p>{/if}
				<button class="btn-primary min-h-10" type="submit" disabled={updateMutation.isPending}
					>{updateMutation.isPending ? fa.saving : fa.save}</button
				>
			</form>
		</section>
	</div>
{/if}
