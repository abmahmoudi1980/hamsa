<script lang="ts">
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getMyAnnouncement, markAnnouncementRead } from '#lib/api/endpoints/announcements';
	import AttachmentPreview from '#lib/components/AttachmentPreview.svelte';
	import { formatJalaliDateTime } from '#lib/format/jalali';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const announcementId = $derived(page.params.id ?? '');
	const announcementQuery = createQuery(() => ({
		queryKey: qk.announcements.detail(announcementId),
		queryFn: () => getMyAnnouncement(announcementId),
		enabled: !!announcementId
	}));
	let startedReadFor = '';
	let readError = $state('');

	const readMutation = createMutation(() => ({
		mutationFn: markAnnouncementRead,
		onError: (cause) => {
			readError = cause instanceof ApiError ? cause.message : fa.errorGeneric;
		},
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.announcements.detail(announcementId) }),
				queryClient.invalidateQueries({ queryKey: ['announcements', 'me'] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.resident() }),
				queryClient.invalidateQueries({ queryKey: ['notifications'] })
			]);
		}
	}));

	$effect(() => {
		const item = announcementQuery.data;
		if (!item || item.is_read || startedReadFor === item.id) return;
		startedReadFor = item.id;
		readMutation.mutate(item.id);
	});

	function dateTime(value: string | null | undefined): string {
		if (!value) return '—';
		const date = new Date(value);
		return Number.isFinite(date.getTime()) ? formatJalaliDateTime(date) : '—';
	}
</script>

{#if announcementQuery.isPending}
	<div class="mx-auto max-w-3xl space-y-4" aria-label={fa.loading}>
		<div class="h-10 animate-pulse rounded-xl bg-surface-high"></div>
		<div class="h-72 animate-pulse rounded-2xl bg-white"></div>
	</div>
{:else if announcementQuery.isError || !announcementQuery.data}
	<section class="mx-auto max-w-3xl card p-8 text-center">
		<p class="text-lg font-bold">{fa.errorGeneric}</p>
		<p class="mt-2 text-sm text-on-surface-variant">
			{announcementQuery.error instanceof ApiError
				? announcementQuery.error.message
				: fa.errorGeneric}
		</p>
		<a class="mt-5 btn-secondary" href="/r/announcements">{fa.back}</a>
	</section>
{:else}
	{@const announcement = announcementQuery.data}
	<div class="mx-auto max-w-3xl space-y-6">
		<a
			class="text-sm font-semibold text-on-surface-variant hover:text-primary"
			href="/r/announcements">{fa.navAnnouncements}</a
		>
		<article class="space-y-5 card p-5 sm:p-8">
			<header>
				<p class="text-sm font-semibold text-primary">{fa.announcementDetail}</p>
				<h1 class="mt-2 text-2xl font-bold tracking-tight sm:text-3xl">{announcement.title}</h1>
				<p class="mt-2 text-xs text-on-surface-variant">{dateTime(announcement.created_at)}</p>
			</header>
			<div class="text-sm leading-8 whitespace-pre-wrap">{announcement.body}</div>
			{#if announcement.attachment_file}<div
					class="space-y-2 border-t border-outline-variant/50 pt-4"
				>
					<p class="text-sm font-semibold">{fa.attachment}</p>
					<AttachmentPreview fileRef={announcement.attachment_file} label={fa.attachment} />
				</div>{/if}
			{#if readError}<p class="text-sm text-danger" role="alert">{readError}</p>{/if}
			{#if !announcement.is_read}<button
					class="btn-secondary min-h-10"
					type="button"
					onclick={() =>
						void readMutation
							.mutateAsync(announcement.id)
							.catch(
								(cause) => (readError = cause instanceof ApiError ? cause.message : fa.errorGeneric)
							)}>{fa.markAsRead}</button
				>{/if}
		</article>
	</div>
{/if}
