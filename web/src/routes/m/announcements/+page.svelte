<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { deleteAnnouncement, listBuildingAnnouncements } from '#lib/api/endpoints/announcements';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import { audienceTypeLabel } from '#lib/format/labels';
	import { formatJalaliDateTime } from '#lib/format/jalali';
	import { qk } from '#lib/query/keys';
	import { toPersianDigits } from '#lib/format/digits';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const PAGE_SIZE = 20;
	const buildingId = $derived(buildingSelection.id ?? '');
	let pageNumber = $state(1);
	const filters = $derived(JSON.stringify({ pageNumber }));
	const announcementsQuery = createQuery(() => ({
		queryKey: qk.announcements.list(buildingId, filters),
		queryFn: () => listBuildingAnnouncements(buildingId, pageNumber, PAGE_SIZE),
		enabled: !!buildingId
	}));
	const announcements = $derived(announcementsQuery.data?.items ?? []);
	const total = $derived(announcementsQuery.data?.total ?? 0);
	let error = $state('');

	const deleteMutation = createMutation(() => ({
		mutationFn: (id: string) => deleteAnnouncement(id),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ['announcements'] }),
				queryClient.invalidateQueries({ queryKey: ['notifications'] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(buildingId) }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.resident() })
			]);
		}
	}));

	async function remove(id: string) {
		if (!confirm(fa.confirmDeleteAnnouncement)) return;
		error = '';
		try {
			await deleteMutation.mutateAsync(id);
		} catch (cause) {
			error = cause instanceof ApiError ? cause.message : fa.errorGeneric;
		}
	}

	function dateTime(value: string | null | undefined): string {
		if (!value) return '—';
		const date = new Date(value);
		return Number.isFinite(date.getTime()) ? formatJalaliDateTime(date) : '—';
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navAnnouncements}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.announcementListDescription}</p>
		</div>
		{#if buildingId}<a class="btn-primary min-h-11" href="/m/announcements/new"
				>{fa.newAnnouncement}</a
			>{/if}
	</header>

	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.noBuildingsHint}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else}
		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if announcementsQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3] as row (row)}<div
							class="h-28 animate-pulse rounded-xl bg-surface-high"
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
				<div
					class="flex items-center gap-2 border-b border-outline-variant/50 px-5 py-3 text-sm text-on-surface-variant"
				>
					<span class="rounded-lg bg-primary/10 px-2 py-0.5 font-bold text-primary"
						>{toPersianDigits(String(total))}</span
					>{fa.resultsCount}
				</div>
				<ul class="divide-y divide-outline-variant/40">
					{#each announcements as announcement (announcement.id)}
						<li class="space-y-3 p-5">
							<div class="flex flex-wrap items-start justify-between gap-3">
								<div class="min-w-0 flex-1">
									<h2 class="font-bold">{announcement.title}</h2>
									<p class="mt-1 text-xs text-on-surface-variant">
										{audienceTypeLabel(announcement.audience_type)} · {fa.announcementPublishedAt}: {dateTime(
											announcement.publish_at
										)}
									</p>
								</div>
								<div class="flex shrink-0 gap-2">
									<a
										class="btn-secondary min-h-9 text-xs"
										href={`/m/announcements/${announcement.id}/edit`}>{fa.edit}</a
									><button
										class="min-h-9 px-2 text-xs font-semibold text-danger hover:underline"
										type="button"
										disabled={deleteMutation.isPending}
										onclick={() => void remove(announcement.id)}>{fa.delete}</button
									>
								</div>
							</div>
							<p class="line-clamp-3 text-sm leading-6 whitespace-pre-wrap text-on-surface-variant">
								{announcement.body}
							</p>
							{#if announcement.attachment_file}<span class="text-xs font-semibold text-primary"
									>{fa.attachment}</span
								>{/if}
							{#if announcement.expire_at}<p class="text-xs text-on-surface-variant">
									{fa.announcementExpiresAt}: {dateTime(announcement.expire_at)}
								</p>{/if}
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
					<p class="text-lg font-bold">{fa.noAnnouncements}</p>
					<a class="mt-5 btn-primary" href="/m/announcements/new">{fa.newAnnouncement}</a>
				</div>
			{/if}
		</section>
		{#if error}<p class="text-sm text-danger" role="alert">{error}</p>{/if}
	{/if}
</div>
