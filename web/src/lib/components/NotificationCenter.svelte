<script lang="ts">
	import { goto } from '$app/navigation';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		listNotifications,
		markAllNotificationsRead,
		markNotificationRead,
		type Notification
	} from '#lib/api/endpoints/notifications';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { formatJalaliDateTime } from '#lib/format/jalali';
	import { toPersianDigits } from '#lib/format/digits';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	interface Props {
		role: 'manager' | 'resident';
	}

	let { role }: Props = $props();
	const queryClient = useQueryClient();
	const PAGE_SIZE = 20;
	let unreadOnly = $state(false);
	let pageNumber = $state(1);
	let error = $state('');
	const filters = $derived(JSON.stringify({ unreadOnly, pageNumber }));
	const notificationsQuery = createQuery(() => ({
		queryKey: qk.notifications.list(filters),
		queryFn: () => listNotifications({ unreadOnly, page: pageNumber, page_size: PAGE_SIZE })
	}));
	const notifications = $derived(notificationsQuery.data?.items ?? []);
	const total = $derived(notificationsQuery.data?.total ?? 0);

	async function invalidateNotifications() {
		await queryClient.invalidateQueries({ queryKey: ['notifications'] });
	}

	const markOneMutation = createMutation(() => ({
		mutationFn: markNotificationRead,
		onSuccess: invalidateNotifications
	}));
	const markAllMutation = createMutation(() => ({
		mutationFn: markAllNotificationsRead,
		onSuccess: invalidateNotifications
	}));

	function targetPath(item: Notification): string | null {
		if (!item.ref_id) return null;
		if (item.ref_type === 'maintenance_request') {
			return role === 'manager' ? `/m/maintenance/${item.ref_id}` : `/r/maintenance/${item.ref_id}`;
		}
		if (item.ref_type === 'announcement') {
			return role === 'manager'
				? `/m/announcements/${item.ref_id}/edit`
				: `/r/announcements/${item.ref_id}`;
		}
		if (item.ref_type === 'invoice') {
			return role === 'manager' ? `/m/invoices/${item.ref_id}` : `/r/invoices/${item.ref_id}`;
		}
		if (item.ref_type === 'expense' && role === 'manager') return `/m/expenses/${item.ref_id}`;
		return null;
	}

	function dateTime(value: string): string {
		const date = new Date(value);
		return Number.isFinite(date.getTime()) ? formatJalaliDateTime(date) : '—';
	}

	async function open(item: Notification) {
		const path = targetPath(item);
		if (!path) return;
		if (!item.is_read) {
			try {
				await markOneMutation.mutateAsync(item.id);
			} catch (cause) {
				error = cause instanceof ApiError ? cause.message : fa.errorGeneric;
				return;
			}
		}
		await goto(path);
	}

	async function markAll() {
		error = '';
		try {
			await markAllMutation.mutateAsync();
		} catch (cause) {
			error = cause instanceof ApiError ? cause.message : fa.errorGeneric;
		}
	}

	async function markOne(id: string) {
		error = '';
		try {
			await markOneMutation.mutateAsync(id);
		} catch (cause) {
			error = cause instanceof ApiError ? cause.message : fa.errorGeneric;
		}
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">
				{role === 'manager' ? fa.managerConsole : fa.navHome}
			</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.notifications}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.notificationListDescription}</p>
		</div>
		<button
			class="btn-secondary min-h-10"
			type="button"
			disabled={markAllMutation.isPending || total === 0}
			onclick={() => void markAll()}>{fa.markAllRead}</button
		>
	</header>
	<div
		class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-outline-variant/60 bg-white p-4"
	>
		<label
			class="flex min-h-10 items-center gap-3 text-sm font-semibold"
			for="notifications-unread-only"
			><input
				id="notifications-unread-only"
				class="size-4 accent-primary"
				type="checkbox"
				bind:checked={unreadOnly}
				onchange={() => (pageNumber = 1)}
			/>{fa.unreadOnly}</label
		>
		<p class="text-xs text-on-surface-variant">
			{toPersianDigits(String(total))}
			{fa.resultsCount}
		</p>
	</div>
	<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
		{#if notificationsQuery.isPending}
			<div class="space-y-2 p-5">
				{#each [1, 2, 3] as row (row)}<div
						class="h-24 animate-pulse rounded-xl bg-surface-high"
					></div>{/each}
			</div>
		{:else if notificationsQuery.isError}
			<div class="p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{notificationsQuery.error instanceof ApiError
						? notificationsQuery.error.message
						: fa.errorGeneric}
				</p>
				<button
					class="mt-5 btn-primary"
					type="button"
					onclick={() => void notificationsQuery.refetch()}>{fa.retry}</button
				>
			</div>
		{:else if notifications.length}
			<ul class="divide-y divide-outline-variant/40">
				{#each notifications as item (item.id)}
					<li class={`space-y-3 p-5 ${item.is_read ? '' : 'bg-primary/[0.035]'}`}>
						<div class="flex items-start justify-between gap-3">
							<div class="min-w-0">
								<h2 class="font-bold">{item.title}</h2>
								<p class="mt-1 text-xs text-on-surface-variant">{dateTime(item.created_at)}</p>
							</div>
							<StatusChip
								tone={item.is_read ? 'neutral' : 'info'}
								label={item.is_read ? fa.announcementAlreadyRead : fa.announcementUnread}
							/>
						</div>
						{#if item.body}<p class="text-sm leading-6 whitespace-pre-wrap text-on-surface-variant">
								{item.body}
							</p>{/if}
						<div class="flex flex-wrap gap-3">
							{#if targetPath(item)}<button
									class="text-sm font-semibold text-primary hover:underline"
									type="button"
									onclick={() => void open(item)}>{fa.notificationRelatedItem}</button
								>{/if}
							{#if !item.is_read}<button
									class="text-sm font-semibold text-on-surface-variant hover:text-primary"
									type="button"
									disabled={markOneMutation.isPending}
									onclick={() => void markOne(item.id)}>{fa.markAsRead}</button
								>{/if}
						</div>
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
			<p class="p-10 text-center text-sm text-on-surface-variant">{fa.noNotifications}</p>
		{/if}
	</section>
	{#if error}<p class="text-sm text-danger" role="alert">{error}</p>{/if}
</div>
