<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { auth } from '#lib/auth/auth.svelte';
	import { ApiError } from '#lib/api/apiError';
	import { getResidentHome } from '#lib/api/endpoints/dashboard';
	import { formatJalaliDate, fromIsoDate } from '#lib/format/jalali';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { invoiceStatusLabel, maintenanceStatusLabel } from '#lib/format/labels';
	import { fa } from '#i18n/fa';
	import { qk } from '#lib/query/keys';

	const homeQuery = createQuery(() => ({
		queryKey: qk.dashboard.resident(),
		queryFn: getResidentHome
	}));

	function dateLabel(value: string | undefined): string {
		if (!value) return '';
		const date = fromIsoDate(value);
		return date ? formatJalaliDate(date) : '';
	}
</script>

{#if homeQuery.isPending}
	<div class="space-y-5" aria-label={fa.loading}>
		<div class="h-9 w-48 animate-pulse rounded-xl bg-surface-high"></div>
		<div class="h-48 animate-pulse rounded-3xl bg-primary/10"></div>
		<div class="grid gap-4 sm:grid-cols-2">
			<div class="h-36 animate-pulse rounded-2xl bg-white"></div>
			<div class="h-36 animate-pulse rounded-2xl bg-white"></div>
		</div>
	</div>
{:else if homeQuery.isError}
	<section class="card p-8 text-center">
		<h1 class="text-xl font-bold">{fa.residentHome}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">
			{homeQuery.error instanceof ApiError ? homeQuery.error.message : fa.errorGeneric}
		</p>
		<button class="mt-5 btn-primary" type="button" onclick={() => void homeQuery.refetch()}
			>{fa.retry}</button
		>
	</section>
{:else if homeQuery.data}
	{@const home = homeQuery.data}
	<div class="space-y-6 sm:space-y-8">
		<header>
			<p class="text-sm font-semibold text-primary">{fa.welcome}، {auth.user?.name}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.residentHome}</h1>
		</header>

		<section
			class="relative overflow-hidden rounded-3xl bg-primary p-6 text-on-primary shadow-[0_24px_70px_-35px_rgba(0,105,92,0.65)] sm:p-9"
		>
			<div
				aria-hidden="true"
				class="absolute -end-12 -top-20 size-64 rounded-full border border-white/10"
			></div>
			<div class="relative">
				<p class="text-sm font-semibold text-white/75">{fa.amountPayable}</p>
				<p class="mt-2 text-3xl font-bold tracking-tight sm:text-5xl">
					{formatTomanWithUnit(home.payable_amount)}
				</p>
				<p class="mt-3 text-sm text-white/75">
					{home.unit_count.toLocaleString('fa-IR')}
					{fa.unit}
				</p>
			</div>
		</section>

		<div class="grid gap-4 lg:grid-cols-2">
			<section class="card p-5 sm:p-6">
				<div class="flex items-center justify-between gap-3">
					<div>
						<p class="text-xs font-semibold text-primary">{fa.latestInvoice}</p>
						<h2 class="mt-1 text-lg font-bold">
							{home.latest_invoice?.period_title || fa.invoice}
						</h2>
					</div>
					{#if home.latest_invoice}
						<span class="rounded-full bg-warning-soft px-3 py-1 text-xs font-semibold text-warning"
							>{invoiceStatusLabel(home.latest_invoice.status)}</span
						>
					{/if}
				</div>
				{#if home.latest_invoice}
					<div class="mt-5 flex items-end justify-between gap-4">
						<div>
							<p class="text-xs text-on-surface-variant">{fa.finalAmount}</p>
							<p class="mt-1 text-xl font-bold">
								{formatTomanWithUnit(home.latest_invoice.final_amount)}
							</p>
						</div>
						<div class="text-end">
							<p class="text-xs text-on-surface-variant">{fa.dueDate}</p>
							<p class="mt-1 text-sm font-semibold">
								{dateLabel(home.latest_invoice.due_date) || '—'}
							</p>
						</div>
					</div>
				{:else}
					<p class="mt-6 rounded-xl bg-surface px-4 py-5 text-sm text-on-surface-variant">
						{fa.noInvoices}
					</p>
				{/if}
			</section>

			<section class="card p-5 sm:p-6">
				<div class="flex items-center justify-between gap-3">
					<div>
						<p class="text-xs font-semibold text-primary">{fa.myRequests}</p>
						<h2 class="mt-1 text-lg font-bold">{fa.openRequests}</h2>
					</div>
					<span
						class="grid size-10 place-items-center rounded-xl bg-warning-soft font-bold text-warning"
						>{home.open_request_count.toLocaleString('fa-IR')}</span
					>
				</div>
				{#if home.latest_request}
					<div class="mt-5 rounded-xl bg-surface px-4 py-4">
						<p class="font-semibold">{home.latest_request.title}</p>
						<p class="mt-2 text-xs text-on-surface-variant">
							{maintenanceStatusLabel(home.latest_request.status)}{dateLabel(
								home.latest_request.created_at
							)
								? ` · ${dateLabel(home.latest_request.created_at)}`
								: ''}
						</p>
					</div>
				{:else}
					<p class="mt-6 rounded-xl bg-surface px-4 py-5 text-sm text-on-surface-variant">
						{fa.noMaintenanceRequests}
					</p>
				{/if}
			</section>
		</div>

		<section class="card p-5 sm:p-6">
			<div class="flex items-center justify-between gap-4">
				<div>
					<p class="text-xs font-semibold text-primary">{fa.navBuildings}</p>
					<h2 class="mt-1 text-lg font-bold">{fa.latestAnnouncements}</h2>
				</div>
				{#if home.unread_announcement_count > 0}
					<span class="rounded-full bg-primary/10 px-3 py-1 text-xs font-bold text-primary"
						>{home.unread_announcement_count.toLocaleString('fa-IR')} {fa.unreadAnnouncements}</span
					>
				{/if}
			</div>
			{#if home.latest_announcements.length}
				<ul class="mt-4 divide-y divide-outline-variant/40">
					{#each home.latest_announcements as announcement (announcement.id)}
						<li class="flex items-center gap-3 py-4 first:pt-1 last:pb-1">
							<span
								class={`size-2 shrink-0 rounded-full ${announcement.is_read ? 'bg-outline-variant' : 'bg-primary'}`}
							></span>
							<span class="min-w-0 flex-1 truncate text-sm font-semibold">{announcement.title}</span
							>
							<span class="shrink-0 text-xs text-on-surface-variant"
								>{dateLabel(announcement.created_at)}</span
							>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="mt-4 rounded-xl bg-surface px-4 py-5 text-sm text-on-surface-variant">
					{fa.noAnnouncements}
				</p>
			{/if}
		</section>
	</div>
{/if}
