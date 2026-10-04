<script lang="ts">
	import { goto } from '$app/navigation';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		createMaintenanceRequest,
		MAINTENANCE_CATEGORIES,
		MAINTENANCE_PRIORITIES
	} from '#lib/api/endpoints/maintenance';
	import FileUploadField from '#lib/components/FileUploadField.svelte';
	import { fileIdFromRef } from '#lib/files/fileCache';
	import { maintenanceCategoryLabel, priorityLabel } from '#lib/format/labels';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	let title = $state('');
	let category = $state<(typeof MAINTENANCE_CATEGORIES)[number]>('other');
	let description = $state('');
	let location = $state('');
	let priority = $state<(typeof MAINTENANCE_PRIORITIES)[number]>('normal');
	let photoFile = $state('');
	let error = $state('');

	const submitMutation = createMutation(() => ({
		mutationFn: () =>
			createMaintenanceRequest({
				title: title.trim(),
				category,
				description: description.trim() || null,
				location: location.trim() || null,
				photo_file_id: photoFile ? fileIdFromRef(photoFile) : undefined,
				priority
			}),
		onSuccess: async (request) => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ['maintenance', 'me'] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.resident() }),
				queryClient.invalidateQueries({ queryKey: ['notifications'] })
			]);
			await goto(`/r/maintenance/${request.id}`, { replace: true });
		}
	}));

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		if (!title.trim()) {
			error = fa.required;
			return;
		}
		try {
			await submitMutation.mutateAsync();
		} catch (cause) {
			error = cause instanceof ApiError ? cause.message : fa.errorGeneric;
		}
	}
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a class="text-sm font-semibold text-on-surface-variant hover:text-primary" href="/r/maintenance"
		>{fa.maintenanceBackToList}</a
	>
	<header>
		<p class="text-sm font-semibold text-primary">{fa.myRequests}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.newMaintenanceRequest}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.maintenanceRequestDescription}</p>
	</header>

	<form class="space-y-5 card p-5 sm:p-7" onsubmit={submit}>
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="maintenance-title"
				>{fa.maintenanceRequestTitle}</label
			>
			<input
				id="maintenance-title"
				class="input-base bg-white"
				bind:value={title}
				maxlength="150"
				required
			/>
		</div>
		<div class="grid gap-4 sm:grid-cols-2">
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="maintenance-category">{fa.category}</label>
				<select id="maintenance-category" class="input-base bg-white" bind:value={category}>
					{#each MAINTENANCE_CATEGORIES as value (value)}<option {value}
							>{maintenanceCategoryLabel(value)}</option
						>{/each}
				</select>
			</div>
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="maintenance-priority">{fa.priority}</label>
				<select id="maintenance-priority" class="input-base bg-white" bind:value={priority}>
					{#each MAINTENANCE_PRIORITIES as value (value)}<option {value}
							>{priorityLabel(value)}</option
						>{/each}
				</select>
			</div>
			<div class="space-y-2 sm:col-span-2">
				<label class="block text-sm font-semibold" for="maintenance-location">{fa.location}</label>
				<input id="maintenance-location" class="input-base bg-white" bind:value={location} />
			</div>
			<div class="space-y-2 sm:col-span-2">
				<label class="block text-sm font-semibold" for="maintenance-description">{fa.detail}</label>
				<textarea
					id="maintenance-description"
					class="min-h-32 input-base bg-white"
					bind:value={description}
					maxlength="2000"></textarea>
			</div>
		</div>
		<FileUploadField bind:value={photoFile} label={fa.photo} imageOnly previewLabel={fa.photo} />
		<p class="-mt-3 text-xs text-on-surface-variant">{fa.maintenancePhotoHint}</p>
		{#if error}<p class="text-sm text-danger" role="alert">{error}</p>{/if}
		<div class="flex flex-wrap gap-3">
			<button class="btn-primary min-h-11" type="submit" disabled={submitMutation.isPending}
				>{submitMutation.isPending ? fa.saving : fa.newMaintenanceRequest}</button
			>
			<a class="btn-secondary min-h-11" href="/r/maintenance">{fa.cancel}</a>
		</div>
	</form>
</div>
