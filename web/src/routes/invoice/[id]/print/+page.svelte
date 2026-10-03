<script lang="ts">
	/**
	 * Print-optimised invoice (research R7). Lives outside the /m and /r shells
	 * so nothing but the document is rendered; the browser's own "Save as PDF"
	 * produces a shareable receipt with no PDF library. `GET /invoices/{id}`
	 * authorizes either the building's manager or the owning resident.
	 */
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getInvoice } from '#lib/api/endpoints/invoices';
	import { auth, restoreSession } from '#lib/auth/auth.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import {
		adjustmentKindLabel,
		calcMethodLabel,
		invoiceItemKindLabel,
		invoiceStatusLabel
	} from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { invoiceOutstanding } from '#lib/payments/balance';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const invoiceId = $derived(page.params.id ?? '');

	const invoiceQuery = createQuery(() => ({
		queryKey: qk.invoices.detail(invoiceId),
		queryFn: () => getInvoice(invoiceId),
		enabled: !!invoiceId && auth.status === 'authenticated'
	}));

	const invoice = $derived(invoiceQuery.data);
	const outstanding = $derived(invoice ? invoiceOutstanding(invoice) : '0');

	onMount(() => {
		if (auth.status === 'unknown') void restoreSession();
	});

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="mx-auto max-w-3xl bg-white p-6 text-on-surface sm:p-10">
	<div class="no-print mb-6 flex flex-wrap items-center justify-between gap-3">
		<button class="btn-secondary min-h-10" type="button" onclick={() => history.back()}
			>{fa.back}</button
		>
		<button class="btn-primary min-h-10" type="button" onclick={() => window.print()}>
			{fa.print}
		</button>
	</div>

	{#if invoiceQuery.isPending}
		<p class="text-sm text-on-surface-variant" role="status">{fa.loading}</p>
	{:else if invoiceQuery.isError}
		<p class="text-sm text-danger" role="alert">
			{invoiceQuery.error instanceof ApiError ? invoiceQuery.error.message : fa.errorGeneric}
		</p>
	{:else if invoice}
		<header class="flex items-start justify-between gap-6 border-b-2 border-primary pb-4">
			<div>
				<p class="text-2xl font-bold text-primary">{fa.appName}</p>
				<p class="mt-1 text-sm text-on-surface-variant">{fa.printInvoiceTitle}</p>
			</div>
			<div class="text-end">
				<p class="text-xs text-on-surface-variant">{fa.invoiceNumber}</p>
				<p class="num text-lg font-bold">{invoice.invoice_number}</p>
				<p class="mt-1 text-xs text-on-surface-variant">
					{fa.status}: {invoiceStatusLabel(invoice.status)}
				</p>
			</div>
		</header>

		<section class="mt-6 grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
			<div>
				<p class="text-xs text-on-surface-variant">{fa.unit}</p>
				<p class="mt-1 font-bold">{toPersianDigits(invoice.unit_number ?? '—')}</p>
			</div>
			<div>
				<p class="text-xs text-on-surface-variant">{fa.period}</p>
				<p class="mt-1 font-bold">{invoice.period_title ?? '—'}</p>
			</div>
			<div>
				<p class="text-xs text-on-surface-variant">{fa.invoiceDate}</p>
				<p class="mt-1 font-bold">{dateLabel(invoice.issue_date)}</p>
			</div>
			<div>
				<p class="text-xs text-on-surface-variant">{fa.dueDateInvoice}</p>
				<p class="mt-1 font-bold">{dateLabel(invoice.due_date)}</p>
			</div>
		</section>

		<section class="mt-6">
			<h2 class="mb-2 text-base font-bold">{fa.itemsBreakdown}</h2>
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-outline-variant">
						<th class="py-2 text-start font-semibold">{fa.detail}</th>
						<th class="py-2 text-start font-semibold">{fa.status}</th>
						<th class="py-2 text-start font-semibold">{fa.calcMethod}</th>
						<th class="py-2 text-start font-semibold">{fa.totalAmount}</th>
					</tr>
				</thead>
				<tbody>
					{#each invoice.items ?? [] as item (item.id)}
						<tr class="border-b border-outline-variant/50">
							<td class="py-2">{item.title}</td>
							<td class="py-2">{invoiceItemKindLabel(item.kind)}</td>
							<td class="py-2">{item.method ? calcMethodLabel(item.method) : '—'}</td>
							<td class="num py-2">{formatTomanWithUnit(item.amount)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</section>

		{#if invoice.adjustments && invoice.adjustments.length}
			<section class="avoid-break mt-6">
				<h2 class="mb-2 text-base font-bold">{fa.adjustments}</h2>
				<ul class="space-y-1 text-sm">
					{#each invoice.adjustments as adjustment (adjustment.id)}
						<li class="flex justify-between gap-4 border-b border-outline-variant/50 py-1.5">
							<span>
								{adjustmentKindLabel(adjustment.kind)} — {adjustment.reason}
							</span>
							<span class="num">{formatTomanWithUnit(adjustment.amount)}</span>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<section class="avoid-break ms-auto mt-6 w-full max-w-sm space-y-2 text-sm">
			<div class="flex justify-between">
				<span class="text-on-surface-variant">{fa.baseAmount}</span>
				<span class="num">{formatTomanWithUnit(invoice.base_amount)}</span>
			</div>
			<div class="flex justify-between">
				<span class="text-on-surface-variant">{fa.priorDebt}</span>
				<span class="num">{formatTomanWithUnit(invoice.prior_debt)}</span>
			</div>
			<div class="flex justify-between">
				<span class="text-on-surface-variant">{fa.lateFeeAmount}</span>
				<span class="num">{formatTomanWithUnit(invoice.late_fee_amount)}</span>
			</div>
			<div class="flex justify-between">
				<span class="text-on-surface-variant">{fa.credit}</span>
				<span class="num">{formatTomanWithUnit(invoice.credit_amount)}</span>
			</div>
			<div class="flex justify-between border-t border-outline-variant pt-2 font-bold">
				<span>{fa.finalAmount}</span>
				<span class="num">{formatTomanWithUnit(invoice.final_amount)}</span>
			</div>
			<div class="flex justify-between">
				<span class="text-on-surface-variant">{fa.paidAmount}</span>
				<span class="num">{formatTomanWithUnit(invoice.paid_amount)}</span>
			</div>
			<div
				class="flex justify-between border-t-2 border-primary pt-2 text-base font-bold text-primary"
			>
				<span>{fa.outstanding}</span>
				<span class="num">{formatTomanWithUnit(outstanding)}</span>
			</div>
		</section>

		<p class="mt-8 border-t border-outline-variant/50 pt-3 text-xs text-on-surface-variant">
			{fa.printInvoiceDescription}
		</p>
	{/if}
</div>
