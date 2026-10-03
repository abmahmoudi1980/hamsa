<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getInvoice } from '#lib/api/endpoints/invoices';
	import AdjustmentForm from '#lib/components/AdjustmentForm.svelte';
	import CancelInvoiceForm from '#lib/components/CancelInvoiceForm.svelte';
	import RecordPaymentForm from '#lib/components/RecordPaymentForm.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import {
		adjustmentKindLabel,
		calcMethodLabel,
		invoiceItemKindLabel,
		invoiceStatusLabel,
		invoiceStatusTone
	} from '#lib/format/labels';
	import { formatTomanWithUnit, toman } from '#lib/format/money';
	import { invoiceEffective, invoiceOutstanding, isPayable } from '#lib/payments/balance';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const invoiceId = $derived(page.params.id ?? '');

	const invoiceQuery = createQuery(() => ({
		queryKey: qk.invoices.detail(invoiceId),
		queryFn: () => getInvoice(invoiceId),
		enabled: !!invoiceId
	}));

	const invoice = $derived(invoiceQuery.data);
	const outstanding = $derived(invoice ? invoiceOutstanding(invoice) : toman('0'));
	const effective = $derived(invoice ? invoiceEffective(invoice) : '0');
	const payable = $derived(invoice ? isPayable(invoice) : false);
	const cancellable = $derived(
		invoice ? ['unpaid', 'partial', 'expired'].includes(invoice.status) : false
	);

	let panel = $state<'none' | 'payment' | 'adjustment' | 'cancel'>('none');
	let notice = $state('');

	function closePanel(message = '') {
		panel = 'none';
		notice = message;
	}

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="mx-auto max-w-5xl space-y-6">
	<a
		href="/m/invoices"
		class="inline-flex items-center gap-2 text-sm font-semibold text-on-surface-variant transition-colors hover:text-primary"
	>
		<svg viewBox="0 0 20 20" fill="none" class="size-4 -scale-x-100" aria-hidden="true"
			><path
				d="M16 10H4m0 0 4.5-4.5M4 10l4.5 4.5"
				stroke="currentColor"
				stroke-width="1.8"
				stroke-linecap="round"
				stroke-linejoin="round"
			/></svg
		>
		{fa.backToInvoices}
	</a>

	{#if invoiceQuery.isPending}
		<div class="h-72 animate-pulse rounded-3xl bg-white"></div>
	{:else if invoiceQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{invoiceQuery.error instanceof ApiError ? invoiceQuery.error.message : fa.errorGeneric}
			</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void invoiceQuery.refetch()}
				>{fa.retry}</button
			>
		</section>
	{:else if invoice}
		<header class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<p class="text-sm font-semibold text-primary">{fa.invoiceDetail}</p>
				<h1 class="mt-1 flex items-center gap-3 text-2xl font-bold tracking-tight sm:text-3xl">
					<span class="num">{invoice.invoice_number}</span>
					<StatusChip
						tone={invoiceStatusTone(invoice.status)}
						label={invoiceStatusLabel(invoice.status)}
					/>
				</h1>
				<p class="mt-2 text-sm text-on-surface-variant">
					{fa.unit}
					{toPersianDigits(invoice.unit_number ?? '—')} · {invoice.period_title ?? '—'}
				</p>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<a class="btn-secondary min-h-10" href={`/m/units/${invoice.unit_id}/balance`}>
					{fa.viewBalance}
				</a>
				<a
					class="btn-secondary min-h-10"
					href={`/invoice/${invoice.id}/print`}
					target="_blank"
					rel="noopener"
				>
					{fa.printInvoice}
				</a>
				{#if payable}
					<button class="btn-primary min-h-10" type="button" onclick={() => (panel = 'payment')}
						>{fa.recordPayment}</button
					>
				{/if}
			</div>
		</header>

		{#if notice}
			<p
				class="rounded-xl border border-success/20 bg-success-soft px-4 py-3 text-sm leading-6 text-success"
				role="status"
			>
				{notice}
			</p>
		{/if}

		{#if panel === 'payment'}
			<RecordPaymentForm
				invoiceId={invoice.id}
				buildingId={invoice.building_id}
				{outstanding}
				onDone={() => closePanel(fa.paymentRecordedSuccess)}
				onCancel={() => closePanel()}
			/>
		{:else if panel === 'adjustment'}
			<AdjustmentForm
				invoiceId={invoice.id}
				buildingId={invoice.building_id}
				onDone={() => closePanel(fa.adjustmentAdded)}
				onCancel={() => closePanel()}
			/>
		{:else if panel === 'cancel'}
			<CancelInvoiceForm
				invoiceId={invoice.id}
				buildingId={invoice.building_id}
				onDone={() => closePanel(fa.invoiceCancelSuccess)}
				onCancel={() => closePanel()}
			/>
		{/if}

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.invoiceAmounts}</h2>
			<dl class="mt-4 grid gap-4 sm:grid-cols-3">
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.baseAmount}</dt>
					<dd class="mt-1 font-semibold">{formatTomanWithUnit(invoice.base_amount)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.priorDebt}</dt>
					<dd class="mt-1 font-semibold">{formatTomanWithUnit(invoice.prior_debt)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.lateFeeAmount}</dt>
					<dd class="mt-1 font-semibold">{formatTomanWithUnit(invoice.late_fee_amount)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.credit}</dt>
					<dd class="mt-1 font-semibold">{formatTomanWithUnit(invoice.credit_amount)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.finalAmount}</dt>
					<dd class="mt-1 font-bold text-primary">{formatTomanWithUnit(invoice.final_amount)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.paidAmount}</dt>
					<dd class="mt-1 font-semibold">{formatTomanWithUnit(invoice.paid_amount)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.invoiceDate}</dt>
					<dd class="mt-1 font-semibold">{dateLabel(invoice.issue_date)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.dueDateInvoice}</dt>
					<dd class="mt-1 font-semibold">{dateLabel(invoice.due_date)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.outstanding}</dt>
					<dd class="mt-1 font-bold text-danger">{formatTomanWithUnit(outstanding)}</dd>
				</div>
			</dl>
			{#if effective !== invoice.final_amount}
				<p class="mt-4 rounded-xl bg-surface px-4 py-3 text-xs text-on-surface-variant">
					{fa.invoiceAmounts} + {fa.adjustments}:
					<span class="num font-bold">{formatTomanWithUnit(effective)}</span>
				</p>
			{/if}
		</section>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			<h2 class="border-b border-outline-variant/50 px-5 py-4 text-lg font-bold">
				{fa.itemsBreakdown}
			</h2>
			{#if invoice.items && invoice.items.length}
				<div class="overflow-x-auto">
					<table class="w-full min-w-[40rem] text-sm">
						<thead class="bg-surface/70 text-xs text-on-surface-variant">
							<tr>
								<th class="px-5 py-3 text-start font-semibold">{fa.detail}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.status}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.calcMethod}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.totalAmount}</th>
							</tr>
						</thead>
						<tbody>
							{#each invoice.items as item (item.id)}
								<tr class="border-t border-outline-variant/40">
									<td class="px-5 py-3 font-semibold">{item.title}</td>
									<td class="px-5 py-3">{invoiceItemKindLabel(item.kind)}</td>
									<td class="px-5 py-3">{item.method ? calcMethodLabel(item.method) : '—'}</td>
									<td class="px-5 py-3 font-semibold">{formatTomanWithUnit(item.amount)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<p class="p-6 text-sm text-on-surface-variant">{fa.noInvoiceItems}</p>
			{/if}
		</section>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			<div
				class="flex flex-wrap items-center justify-between gap-3 border-b border-outline-variant/50 px-5 py-4"
			>
				<h2 class="text-lg font-bold">{fa.adjustments}</h2>
				{#if !['cancelled'].includes(invoice.status) && panel === 'none'}
					<button
						class="btn-secondary min-h-10"
						type="button"
						onclick={() => (panel = 'adjustment')}
					>
						{fa.addAdjustment}
					</button>
				{/if}
			</div>
			{#if invoice.adjustments && invoice.adjustments.length}
				<ul class="divide-y divide-outline-variant/40">
					{#each invoice.adjustments as adjustment (adjustment.id)}
						<li class="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
							<div>
								<p class="font-semibold">
									{adjustmentKindLabel(adjustment.kind)} — {formatTomanWithUnit(adjustment.amount)}
								</p>
								<p class="mt-1 text-xs text-on-surface-variant">{adjustment.reason}</p>
							</div>
							<span class="text-xs text-on-surface-variant">{dateLabel(adjustment.created_at)}</span
							>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="p-6 text-sm text-on-surface-variant">{fa.noAdjustments}</p>
			{/if}
		</section>

		{#if cancellable && panel === 'none'}
			<section class="rounded-2xl border border-danger/30 bg-danger-soft/40 p-5 sm:p-6">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div>
						<h2 class="text-lg font-bold text-danger">{fa.cancelInvoice}</h2>
						<p class="mt-1 text-xs text-on-surface-variant">{fa.issueInvoicesHint}</p>
					</div>
					<button
						class="btn-secondary min-h-10 text-danger"
						type="button"
						onclick={() => (panel = 'cancel')}>{fa.cancelInvoice}</button
					>
				</div>
			</section>
		{/if}
	{/if}
</div>
