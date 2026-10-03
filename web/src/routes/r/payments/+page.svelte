<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listMyPayments } from '#lib/api/endpoints/payments';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import { paymentMethodLabel, paymentStatusLabel, paymentStatusTone } from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;
	let page = $state(1);

	const filterKey = $derived(JSON.stringify({ page }));

	const paymentsQuery = createQuery(() => ({
		queryKey: qk.payments.mine(filterKey),
		queryFn: () => listMyPayments({ page, page_size: PAGE_SIZE })
	}));

	const payments = $derived(paymentsQuery.data?.items ?? []);
	const total = $derived(paymentsQuery.data?.total ?? 0);

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="space-y-6">
	<header>
		<h1 class="text-2xl font-bold tracking-tight sm:text-3xl">{fa.myPayments}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.paymentHistory}</p>
	</header>

	<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
		{#if paymentsQuery.isPending}
			<div class="space-y-2 p-5">
				{#each [1, 2, 3, 4] as row (row)}
					<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
				{/each}
			</div>
		{:else if paymentsQuery.isError}
			<div class="p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{paymentsQuery.error instanceof ApiError ? paymentsQuery.error.message : fa.errorGeneric}
				</p>
				<button class="mt-5 btn-primary" type="button" onclick={() => void paymentsQuery.refetch()}
					>{fa.retry}</button
				>
			</div>
		{:else if payments.length}
			<ul class="divide-y divide-outline-variant/40">
				{#each payments as payment (payment.id)}
					<li class="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
						<div>
							<p class="font-bold">{formatTomanWithUnit(payment.amount)}</p>
							<p class="mt-1 text-xs text-on-surface-variant">
								{paymentMethodLabel(payment.method)} · {dateLabel(payment.paid_at)}
								{#if payment.tracking_number}
									· <span class="num">{payment.tracking_number}</span>
								{/if}
							</p>
						</div>
						<div class="flex items-center gap-3">
							<StatusChip
								tone={paymentStatusTone(payment.status)}
								label={paymentStatusLabel(payment.status)}
							/>
							{#if payment.invoice_id}
								<a
									class="text-xs font-semibold text-primary hover:underline"
									href={`/r/invoices/${payment.invoice_id}`}>{fa.invoice}</a
								>
							{/if}
						</div>
					</li>
				{/each}
			</ul>
			<Pagination {page} pageSize={PAGE_SIZE} {total} onChange={(next) => (page = next)} />
		{:else}
			<div class="p-10 text-center">
				<p class="text-lg font-bold">{fa.emptyPaymentsTitle}</p>
				<p class="mt-2 text-sm text-on-surface-variant">{fa.noPaymentsHint}</p>
			</div>
		{/if}
	</section>
</div>
