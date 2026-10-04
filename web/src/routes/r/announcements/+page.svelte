<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listMyAnnouncements } from '#lib/api/endpoints/announcements';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { formatJalaliDate, fromIsoDate } from '#lib/format/jalali';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;
	let pageNumber = $state(1);
	const filters = $derived(JSON.stringify({ pageNumber }));
	const announcementsQuery = createQuery(() => ({
		queryKey: qk.announcements.mine(filters),
		queryFn: () => listMyAnnouncements(pageNumber, PAGE_SIZE)
	}));
	const announcements = $derived(announcementsQuery.data?.items ?? []);
	const total = $derived(announcementsQuery.data?.total ?? 0);

	function dateLabel(value: string): string {
		const date = fromIsoDate(value);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="space-y-6">
	<header>
		<p class="text-sm font-semibold text-primary">{fa.navHome}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navAnnouncements}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.latestAnnouncements}</p>
	</header>
	<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
		{#if announcementsQuery.isPending}
			<div class="space-y-2 p-5">
				{#each [1, 2, 3] as row (row)}<div
						class="h-24 animate-pulse rounded-xl bg-surface-high"
					></div>{/each}
			</div>
		{:else if announcementsQuery.isError}
			<div class="p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{announcementsQuery.error instanceof ApiError
						? announcementsQuery.error.message
						: fa.errorGeneric}
				</p>
				<button
					class="mt-5 btn-primary"
					type="button"
					onclick={() => void announcementsQuery.refetch()}>{fa.retry}</button
				>
			</div>
		{:else if announcements.length}
			<ul class="divide-y divide-outline-variant/40">
				{#each announcements as announcement (announcement.id)}
					<li>
						<a
							class="block space-y-2 px-5 py-4 transition-colors hover:bg-surface/70"
							href={`/r/announcements/${announcement.id}`}
						>
							<div class="flex items-center justify-between gap-3">
								<h2 class="min-w-0 truncate font-bold">{announcement.title}</h2>
								<StatusChip
									tone={announcement.is_read ? 'neutral' : 'info'}
									label={announcement.is_read ? fa.announcementAlreadyRead : fa.announcementUnread}
								/>
							</div>
							<p class="line-clamp-2 text-sm leading-6 text-on-surface-variant">
								{announcement.body}
							</p>
							<p class="text-xs text-on-surface-variant">{dateLabel(announcement.created_at)}</p>
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
			<p class="p-10 text-center text-sm text-on-surface-variant">{fa.noAnnouncements}</p>
		{/if}
	</section>
</div>
