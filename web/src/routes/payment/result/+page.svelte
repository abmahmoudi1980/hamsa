<script lang="ts">
	/**
	 * Gateway return page (F3). The callback verified the payment server-side
	 * and 302'd here with `?payment_id=&status=&invoice_id=`. The status query
	 * value is the server's post-verification state, so the panel reflects the
	 * real outcome; the invoice is re-read for the amount and the receipt link.
	 */
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { getInvoice } from '#lib/api/endpoints/invoices';
	import { auth, isManager, restoreSession } from '#lib/auth/auth.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { invoiceOutstanding } from '#lib/payments/balance';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const status = page.url.searchParams.get('status') ?? '';
	const paymentId = page.url.searchParams.get('payment_id') ?? '';
	const invoiceId = page.url.searchParams.get('invoice_id') ?? '';

	const invoiceQuery = createQuery(() => ({
		queryKey: qk.invoices.detail(invoiceId),
		queryFn: () => getInvoice(invoiceId),
		enabled: !!invoiceId && auth.status === 'authenticated'
	}));

	const invoice = $derived(invoiceQuery.data);

	onMount(() => {
		if (auth.status === 'unknown') void restoreSession();
	});

	const outcome = $derived(
		status === 'verified' ? 'success' : status === 'failed' ? 'failed' : 'pending'
	);
	const homeHref = $derived(isManager() ? '/m' : '/r');
</script>

<main class="grid min-h-dvh place-items-center bg-surface px-4 py-10">
	<div class="w-full max-w-lg space-y-6">
		<div class="card p-6 text-center sm:p-8">
			<div
				class={`mx-auto grid size-16 place-items-center rounded-3xl ${
					outcome === 'success'
						? 'bg-success-soft text-success'
						: outcome === 'failed'
							? 'bg-danger-soft text-danger'
							: 'bg-warning-soft text-warning'
				}`}
				aria-hidden="true"
			>
				<span class="text-3xl font-bold"
					>{outcome === 'success' ? '✓' : outcome === 'failed' ? '✕' : '…'}</span
				>
			</div>

			<h1 class="mt-5 text-xl font-bold">{fa.paymentResultTitle}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">
				{outcome === 'success'
					? fa.paymentResultSuccess
					: outcome === 'failed'
						? fa.paymentResultFailed
						: fa.paymentResultPending}
			</p>

			<div class="mt-4 flex justify-center">
				<StatusChip
					tone={outcome === 'success' ? 'success' : outcome === 'failed' ? 'danger' : 'warning'}
					label={status === 'verified'
						? fa.paymentVerified
						: status === 'failed'
							? fa.paymentFailedStatus
							: fa.paymentPending}
				/>
			</div>

			{#if invoice}
				<div class="mt-6 space-y-2 rounded-2xl bg-surface px-4 py-4 text-sm">
					<div class="flex items-center justify-between gap-4">
						<span class="text-on-surface-variant">{fa.invoiceNumber}</span>
						<span class="num font-semibold">{invoice.invoice_number}</span>
					</div>
					<div class="flex items-center justify-between gap-4">
						<span class="text-on-surface-variant">{fa.unit}</span>
						<span class="font-semibold">{toPersianDigits(invoice.unit_number ?? '—')}</span>
					</div>
					<div class="flex items-center justify-between gap-4">
						<span class="text-on-surface-variant">{fa.outstandingAmount}</span>
						<span class="num font-bold text-primary"
							>{formatTomanWithUnit(invoiceOutstanding(invoice))}</span
						>
					</div>
				</div>
			{/if}

			{#if paymentId}
				<p class="mt-4 text-xs text-on-surface-variant">
					{fa.paymentReference}: <span class="num">{paymentId}</span>
				</p>
			{/if}

			<div class="mt-7 flex flex-col gap-3 sm:flex-row sm:justify-center">
				<a class="btn-primary" href={homeHref}>{fa.returnHome}</a>
				{#if invoice}
					<a
						class="btn-secondary"
						href={`/invoice/${invoice.id}/print`}
						target="_blank"
						rel="noopener"
					>
						{fa.printReceipt}
					</a>
				{/if}
			</div>
		</div>
	</div>
</main>
