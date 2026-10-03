<script lang="ts">
	import { page } from '$app/state';
	import { createMutation, createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getMyInvoice, startGatewayPayment } from '#lib/api/endpoints/invoices';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { PAYMENT_RESULT_PATH } from '#lib/config';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import {
		adjustmentKindLabel,
		calcMethodLabel,
		invoiceItemKindLabel,
		invoiceStatusLabel,
		invoiceStatusTone
	} from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { invoiceOutstanding, isPayable } from '#lib/payments/balance';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const invoiceId = $derived(page.params.id ?? '');

	const invoiceQuery = createQuery(() => ({
		queryKey: qk.invoices.detail(invoiceId),
		queryFn: () => getMyInvoice(invoiceId),
		enabled: !!invoiceId
	}));

	const invoice = $derived(invoiceQuery.data);
	const outstanding = $derived(invoice ? invoiceOutstanding(invoice) : '0');
	const payable = $derived(invoice ? isPayable(invoice) : false);

	let actionError = $state('');

	const payMutation = createMutation(() => ({
		mutationFn: () => startGatewayPayment(invoiceId, { return_path: PAYMENT_RESULT_PATH }),
		onSuccess: (result) => {
			window.location.href = result.payment_url;
		}
	}));

	async function pay() {
		actionError = '';
		try {
			await payMutation.mutateAsync();
		} catch (caught) {
			actionError = caught instanceof ApiError ? caught.message : fa.errorGeneric;
		}
	}

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		href="/r/invoices"
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
		{fa.myInvoices}
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
		<header class="flex flex-wrap items-center justify-between gap-3">
			<div>
				<p class="num text-lg font-bold">{invoice.invoice_number}</p>
				<p class="mt-1 text-xs text-on-surface-variant">
					{invoice.period_title ?? '—'} · {fa.unit}
					{toPersianDigits(invoice.unit_number ?? '—')}
				</p>
			</div>
			<StatusChip
				tone={invoiceStatusTone(invoice.status)}
				label={invoiceStatusLabel(invoice.status)}
			/>
		</header>

		<section class="relative overflow-hidden rounded-3xl bg-primary p-6 text-on-primary sm:p-9">
			<p class="text-sm font-semibold text-white/75">{fa.outstandingAmount}</p>
			<p class="mt-2 text-3xl font-bold tracking-tight sm:text-5xl">
				{formatTomanWithUnit(outstanding)}
			</p>
			<p class="mt-3 text-sm text-white/75">
				{fa.dueDateInvoice}: {dateLabel(invoice.due_date)}
			</p>
			{#if payable}
				<button
					class="mt-6 btn-primary bg-white text-primary hover:bg-white/90"
					type="button"
					disabled={payMutation.isPending}
					onclick={() => void pay()}
				>
					{payMutation.isPending ? fa.gatewayRedirectHint : fa.payNow}
				</button>
			{/if}
		</section>

		{#if actionError}
			<p
				class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
				role="alert"
			>
				{actionError}
			</p>
		{:else if payable}
			<p class="text-xs text-on-surface-variant">{fa.payOnlineHint}</p>
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
			</dl>
		</section>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			<h2 class="border-b border-outline-variant/50 px-5 py-4 text-lg font-bold">
				{fa.itemsBreakdown}
			</h2>
			{#if invoice.items && invoice.items.length}
				<ul class="divide-y divide-outline-variant/40">
					{#each invoice.items as item (item.id)}
						<li class="flex items-center justify-between gap-4 px-5 py-4">
							<div>
								<p class="font-semibold">{item.title}</p>
								<p class="mt-1 text-xs text-on-surface-variant">
									{invoiceItemKindLabel(item.kind)}
									{item.method ? ` · ${calcMethodLabel(item.method)}` : ''}
								</p>
							</div>
							<span class="num font-semibold">{formatTomanWithUnit(item.amount)}</span>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="p-6 text-sm text-on-surface-variant">{fa.noInvoiceItems}</p>
			{/if}
		</section>

		{#if invoice.adjustments && invoice.adjustments.length}
			<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
				<h2 class="border-b border-outline-variant/50 px-5 py-4 text-lg font-bold">
					{fa.adjustments}
				</h2>
				<ul class="divide-y divide-outline-variant/40">
					{#each invoice.adjustments as adjustment (adjustment.id)}
						<li class="flex items-center justify-between gap-4 px-5 py-4">
							<div>
								<p class="font-semibold">
									{adjustmentKindLabel(adjustment.kind)} — {formatTomanWithUnit(adjustment.amount)}
								</p>
								<p class="mt-1 text-xs text-on-surface-variant">{adjustment.reason}</p>
							</div>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<a class="btn-secondary" href={`/invoice/${invoice.id}/print`} target="_blank" rel="noopener">
			{fa.printInvoice}
		</a>
	{/if}
</div>
