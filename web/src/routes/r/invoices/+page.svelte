<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listMyInvoices } from '#lib/api/endpoints/invoices';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import { invoiceStatusLabel, invoiceStatusTone } from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;
	let page = $state(1);

	const filterKey = $derived(JSON.stringify({ page }));

	const invoicesQuery = createQuery(() => ({
		queryKey: qk.invoices.mine(filterKey),
		queryFn: () => listMyInvoices({ page, page_size: PAGE_SIZE })
	}));

	const invoices = $derived(invoicesQuery.data?.items ?? []);
	const total = $derived(invoicesQuery.data?.total ?? 0);

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="space-y-6">
	<header>
		<h1 class="text-2xl font-bold tracking-tight sm:text-3xl">{fa.myInvoices}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.emptyInvoicesBody}</p>
	</header>

	<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
		{#if invoicesQuery.isPending}
			<div class="space-y-2 p-5">
				{#each [1, 2, 3, 4] as row (row)}
					<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
				{/each}
			</div>
		{:else if invoicesQuery.isError}
			<div class="p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{invoicesQuery.error instanceof ApiError ? invoicesQuery.error.message : fa.errorGeneric}
				</p>
				<button class="mt-5 btn-primary" type="button" onclick={() => void invoicesQuery.refetch()}
					>{fa.retry}</button
				>
			</div>
		{:else if invoices.length}
			<ul class="divide-y divide-outline-variant/40">
				{#each invoices as invoice (invoice.id)}
					<li>
						<a class="block px-5 py-4 hover:bg-surface/60" href={`/r/invoices/${invoice.id}`}>
							<div class="flex flex-wrap items-center justify-between gap-3">
								<div>
									<p class="num font-bold">{invoice.invoice_number}</p>
									<p class="mt-1 text-xs text-on-surface-variant">
										{invoice.period_title ?? '—'} · {fa.unit}
										{toPersianDigits(invoice.unit_number ?? '—')}
									</p>
								</div>
								<StatusChip
									tone={invoiceStatusTone(invoice.status)}
									label={invoiceStatusLabel(invoice.status)}
								/>
							</div>
							<div class="mt-3 flex flex-wrap items-end justify-between gap-3">
								<div>
									<p class="text-xs text-on-surface-variant">{fa.finalAmount}</p>
									<p class="mt-0.5 font-bold text-primary">
										{formatTomanWithUnit(invoice.final_amount)}
									</p>
								</div>
								<div class="text-end">
									<p class="text-xs text-on-surface-variant">{fa.dueDateInvoice}</p>
									<p class="mt-0.5 text-sm font-semibold">{dateLabel(invoice.due_date)}</p>
								</div>
							</div>
						</a>
					</li>
				{/each}
			</ul>
			<Pagination {page} pageSize={PAGE_SIZE} {total} onChange={(next) => (page = next)} />
		{:else}
			<div class="p-10 text-center">
				<p class="text-lg font-bold">{fa.emptyInvoicesTitle}</p>
				<p class="mt-2 text-sm text-on-surface-variant">{fa.noInvoicesHint}</p>
			</div>
		{/if}
	</section>
</div>
