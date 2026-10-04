<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getAnnouncement } from '#lib/api/endpoints/announcements';
	import AnnouncementForm from '#lib/components/AnnouncementForm.svelte';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const announcementId = $derived(page.params.id ?? '');
	const announcementQuery = createQuery(() => ({
		queryKey: qk.announcements.detail(announcementId),
		queryFn: () => getAnnouncement(announcementId),
		enabled: !!announcementId
	}));
	const buildingId = $derived(announcementQuery.data?.building_id ?? '');
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		class="text-sm font-semibold text-on-surface-variant hover:text-primary"
		href="/m/announcements">{fa.navAnnouncements}</a
	>
	<header>
		<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.editAnnouncement}</h1>
	</header>
	{#if announcementQuery.isPending}
		<div class="h-80 animate-pulse rounded-2xl bg-white" aria-label={fa.loading}></div>
	{:else if announcementQuery.isError || !announcementQuery.data}
		<section class="card p-8 text-center">
			<p class="text-sm text-on-surface-variant">
				{announcementQuery.error instanceof ApiError
					? announcementQuery.error.message
					: fa.errorGeneric}
			</p>
			<a class="mt-5 btn-secondary" href="/m/announcements">{fa.back}</a>
		</section>
	{:else}
		<AnnouncementForm
			{buildingId}
			announcement={announcementQuery.data}
			onDone={() => void goto('/m/announcements', { replace: true })}
			onCancel={() => void goto('/m/announcements')}
		/>
	{/if}
</div>
