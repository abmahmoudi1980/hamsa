<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		ANNOUNCEMENT_AUDIENCES,
		createAnnouncement,
		updateAnnouncement,
		type Announcement,
		type AnnouncementInput
	} from '#lib/api/endpoints/announcements';
	import { listUnits } from '#lib/api/endpoints/units';
	import { fileIdFromRef } from '#lib/files/fileCache';
	import { apiScheduleFromLocal, localScheduleFromApi } from '#lib/announcements/schedule';
	import FileUploadField from '#lib/components/FileUploadField.svelte';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import { audienceTypeLabel } from '#lib/format/labels';
	import { toPersianDigits } from '#lib/format/digits';
	import { getUnit } from '#lib/api/endpoints/units';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	interface Props {
		buildingId: string;
		announcement?: Announcement;
		onDone: (announcement: Announcement) => void;
		onCancel: () => void;
	}

	let { buildingId, announcement, onDone, onCancel }: Props = $props();
	const queryClient = useQueryClient();
	let title = $state('');
	let body = $state('');
	let audienceType = $state<(typeof ANNOUNCEMENT_AUDIENCES)[number]>('all');
	let audienceValue = $state('');
	let unitSearch = $state('');
	let publishDate = $state('');
	let publishTime = $state('');
	let expireDate = $state('');
	let expireTime = $state('');
	let attachment = $state('');
	let loadedId = '';
	let error = $state('');

	$effect(() => {
		const current = announcement;
		if (!current || loadedId === current.id) return;
		loadedId = current.id;
		title = current.title;
		body = current.body;
		audienceType = current.audience_type as (typeof ANNOUNCEMENT_AUDIENCES)[number];
		audienceValue = current.audience_value ?? '';
		const publish = localScheduleFromApi(current.publish_at);
		publishDate = publish.date;
		publishTime = publish.time;
		const expire = localScheduleFromApi(current.expire_at);
		expireDate = expire.date;
		expireTime = expire.time;
		attachment = current.attachment_file ?? '';
	});

	const unitsQuery = createQuery(() => ({
		queryKey: ['announcement-units', buildingId, unitSearch.trim()],
		queryFn: () =>
			listUnits(buildingId, { q: unitSearch.trim() || undefined, page: 1, page_size: 20 }),
		enabled: !!buildingId && audienceType === 'unit' && unitSearch.trim().length > 0
	}));
	const selectedUnitQuery = createQuery(() => ({
		queryKey: ['units', 'detail', audienceValue],
		queryFn: () => getUnit(audienceValue),
		enabled:
			audienceType === 'unit' &&
			!!audienceValue &&
			!(unitsQuery.data?.items.some((unit) => unit.id === audienceValue) ?? false)
	}));
	const unitOptions = $derived([
		...(unitsQuery.data?.items ?? []),
		...(selectedUnitQuery.data &&
		!(unitsQuery.data?.items.some((unit) => unit.id === selectedUnitQuery.data?.id) ?? false)
			? [selectedUnitQuery.data]
			: [])
	]);

	function buildInput(): AnnouncementInput {
		const attachmentUnchanged = attachment === (announcement?.attachment_file ?? '');
		const publishAt = apiScheduleFromLocal(publishDate, publishTime);
		const expireAt = apiScheduleFromLocal(expireDate, expireTime);
		return {
			title: title.trim(),
			body: body.trim(),
			audience_type: audienceType,
			audience_value: audienceType === 'all' ? null : audienceValue.trim(),
			publish_at: publishAt ?? (announcement?.publish_at ? '' : undefined),
			expire_at: expireAt ?? (announcement?.expire_at ? '' : undefined),
			attachment_file_id: attachmentUnchanged
				? undefined
				: attachment
					? fileIdFromRef(attachment)
					: ''
		};
	}

	const saveMutation = createMutation(() => ({
		mutationFn: () =>
			announcement
				? updateAnnouncement(announcement.id, buildInput())
				: createAnnouncement(buildingId, buildInput()),
		onSuccess: async (saved) => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ['announcements'] }),
				queryClient.invalidateQueries({ queryKey: ['notifications'] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(buildingId) }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.resident() })
			]);
			onDone(saved);
		}
	}));

	async function save(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		if (!title.trim() || !body.trim()) {
			error = fa.required;
			return;
		}
		if (audienceType !== 'all' && !audienceValue.trim()) {
			error = fa.required;
			return;
		}
		try {
			await saveMutation.mutateAsync();
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

<form class="space-y-5 card p-5 sm:p-7" onsubmit={save}>
	<div class="space-y-2">
		<label class="block text-sm font-semibold" for="announcement-title"
			>{fa.announcementTitle}</label
		>
		<input
			id="announcement-title"
			class="input-base bg-white"
			bind:value={title}
			maxlength="200"
			required
		/>
	</div>
	<div class="space-y-2">
		<label class="block text-sm font-semibold" for="announcement-body">{fa.announcementBody}</label>
		<textarea id="announcement-body" class="min-h-40 input-base bg-white" bind:value={body} required
		></textarea>
	</div>
	<div class="grid gap-4 sm:grid-cols-2">
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="announcement-audience">{fa.audience}</label>
			<select
				id="announcement-audience"
				class="input-base bg-white"
				bind:value={audienceType}
				onchange={() => (audienceValue = '')}
			>
				{#each ANNOUNCEMENT_AUDIENCES as value (value)}<option {value}
						>{audienceTypeLabel(value)}</option
					>{/each}
			</select>
		</div>
		{#if audienceType === 'block'}
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="announcement-audience-value"
					>{fa.announcementTargetValue}</label
				><input
					id="announcement-audience-value"
					class="input-base bg-white"
					bind:value={audienceValue}
					maxlength="20"
				/>
			</div>
		{:else if audienceType === 'floor'}
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="announcement-audience-value"
					>{fa.announcementTargetValue}</label
				><input
					id="announcement-audience-value"
					class="input-base bg-white"
					bind:value={audienceValue}
					inputmode="numeric"
					dir="ltr"
				/>
			</div>
		{:else if audienceType === 'unit'}
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="announcement-unit-search"
					>{fa.selectAnnouncementUnit}</label
				>
				<input
					id="announcement-unit-search"
					class="input-base bg-white"
					bind:value={unitSearch}
					placeholder={fa.announcementSearchUnit}
				/>
				<select
					class="input-base bg-white"
					bind:value={audienceValue}
					aria-label={fa.announcementSearchUnit}
				>
					<option value="">{fa.selectAnnouncementUnit}</option>
					{#each unitOptions as unit (unit.id)}<option value={unit.id}
							>{unit.block ? `${unit.block} · ` : ''}{fa.floor}
							{toPersianDigits(String(unit.floor))} · {fa.unit}
							{toPersianDigits(unit.number)}</option
						>{/each}
				</select>
				{#if unitsQuery.isError}<p class="text-xs text-danger">
						{unitsQuery.error instanceof ApiError ? unitsQuery.error.message : fa.errorGeneric}
					</p>{/if}
				{#if unitSearch.trim() && !unitsQuery.isPending && !unitsQuery.data?.items.length}<p
						class="text-xs text-on-surface-variant"
					>
						{fa.noAnnouncementUnits}
					</p>{/if}
			</div>
		{/if}
	</div>
	<div class="grid gap-4 sm:grid-cols-2">
		<div class="space-y-3 rounded-xl bg-surface p-4">
			<JalaliDateInput
				id="announcement-publish-date"
				label={fa.publishAt}
				bind:value={publishDate}
			/>
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="announcement-publish-time"
					>{fa.scheduleTime}</label
				><input
					id="announcement-publish-time"
					class="input-base bg-white"
					type="time"
					dir="ltr"
					bind:value={publishTime}
				/>
			</div>
		</div>
		<div class="space-y-3 rounded-xl bg-surface p-4">
			<JalaliDateInput id="announcement-expire-date" label={fa.expireAt} bind:value={expireDate} />
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="announcement-expire-time"
					>{fa.scheduleTime}</label
				><input
					id="announcement-expire-time"
					class="input-base bg-white"
					type="time"
					dir="ltr"
					bind:value={expireTime}
				/>
			</div>
		</div>
	</div>
	<FileUploadField bind:value={attachment} label={fa.attachment} previewLabel={fa.attachment} />
	{#if error}<p class="text-sm text-danger" role="alert">{error}</p>{/if}
	<div class="flex flex-wrap gap-3">
		<button class="btn-primary min-h-11" type="submit" disabled={saveMutation.isPending}
			>{saveMutation.isPending ? fa.saving : fa.save}</button
		>
		<button class="btn-secondary min-h-11" type="button" onclick={onCancel}>{fa.cancel}</button>
	</div>
</form>
