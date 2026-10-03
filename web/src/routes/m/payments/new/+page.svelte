<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getInvoice, listInvoices } from '#lib/api/endpoints/invoices';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import RecordPaymentForm from '#lib/components/RecordPaymentForm.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { invoiceStatusLabel, invoiceStatusTone } from '#lib/format/labels';
	import { formatTomanWithUnit, toman } from '#lib/format/money';
	import { invoiceOutstanding } from '#lib/payments/balance';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const buildingId = $derived(buildingSelection.id ?? '');
	const queryInvoiceId = page.url.searchParams.get('invoice_id') ?? '';

	let selectedId = $state(queryInvoiceId);

	const unpaidQuery = createQuery(() => ({
		queryKey: qk.invoices.list(buildingId, 'pay-new-unpaid'),
		queryFn: () => listInvoices(buildingId, { status: 'unpaid', page_size: 100 }),
		enabled: !!buildingId && !selectedId
	}));
	const partialQuery = createQuery(() => ({
		queryKey: qk.invoices.list(buildingId, 'pay-new-partial'),
		queryFn: () => listInvoices(buildingId, { status: 'partial', page_size: 100 }),
		enabled: !!buildingId && !selectedId
	}));

	const payable = $derived([
		...(unpaidQuery.data?.items ?? []),
		...(partialQuery.data?.items ?? [])
	]);

	const detailQuery = createQuery(() => ({
		queryKey: qk.invoices.detail(selectedId),
		queryFn: () => getInvoice(selectedId),
		enabled: !!selectedId
	}));

	const selected = $derived(detailQuery.data);
	const outstanding = $derived(selected ? invoiceOutstanding(selected) : toman('0'));

	function choose(id: string) {
		selectedId = id;
	}

	function done() {
		void goto('/m/payments');
	}
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		href="/m/payments"
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
		{fa.ledger}
	</a>

	<header>
		<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.recordPayment}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.partialPaymentHint}</p>
	</header>

	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else if selectedId}
		{#if detailQuery.isPending}
			<div class="h-48 animate-pulse rounded-3xl bg-white"></div>
		{:else if detailQuery.isError}
			<section class="card p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{detailQuery.error instanceof ApiError ? detailQuery.error.message : fa.errorGeneric}
				</p>
				<button class="mt-5 btn-secondary" type="button" onclick={() => (selectedId = '')}
					>{fa.selectInvoice}</button
				>
			</section>
		{:else if selected}
			<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div>
						<p class="text-xs text-on-surface-variant">{fa.invoiceNumber}</p>
						<p class="num font-bold">{selected.invoice_number}</p>
						<p class="mt-1 text-xs text-on-surface-variant">
							{fa.unit}
							{toPersianDigits(selected.unit_number ?? '—')} · {selected.period_title ?? '—'}
						</p>
					</div>
					<StatusChip
						tone={invoiceStatusTone(selected.status)}
						label={invoiceStatusLabel(selected.status)}
					/>
				</div>
				<div class="mt-4 flex flex-wrap items-end justify-between gap-4">
					<div>
						<p class="text-xs text-on-surface-variant">{fa.outstanding}</p>
						<p class="mt-1 text-xl font-bold text-danger">{formatTomanWithUnit(outstanding)}</p>
					</div>
					<button class="btn-secondary min-h-10" type="button" onclick={() => (selectedId = '')}>
						{fa.selectInvoice}
					</button>
				</div>
			</section>

			<RecordPaymentForm
				invoiceId={selected.id}
				buildingId={selected.building_id}
				{outstanding}
				onDone={done}
				onCancel={() => void goto('/m/payments')}
			/>
		{/if}
	{:else}
		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			<h2 class="border-b border-outline-variant/50 px-5 py-4 text-lg font-bold">
				{fa.selectInvoice}
			</h2>
			{#if unpaidQuery.isPending || partialQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3] as row (row)}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{/each}
				</div>
			{:else if payable.length}
				<ul class="divide-y divide-outline-variant/40">
					{#each payable as invoice (invoice.id)}
						<li>
							<button
								class="flex w-full items-center justify-between gap-4 px-5 py-4 text-start hover:bg-surface/60"
								type="button"
								onclick={() => choose(invoice.id)}
							>
								<span>
									<span class="num block font-bold">{invoice.invoice_number}</span>
									<span class="mt-1 block text-xs text-on-surface-variant">
										{fa.unit}
										{toPersianDigits(invoice.unit_number ?? '—')} · {invoice.period_title ?? '—'}
									</span>
								</span>
								<span class="text-end">
									<span class="block font-semibold"
										>{formatTomanWithUnit(invoice.final_amount)}</span
									>
									<span class="mt-1 block text-xs text-on-surface-variant">{fa.outstanding}</span>
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{:else}
				<div class="p-10 text-center">
					<p class="text-lg font-bold">{fa.emptyInvoicesTitle}</p>
					<p class="mt-2 text-sm text-on-surface-variant">{fa.noInvoicesHint}</p>
					<a class="mt-5 btn-primary" href="/m/invoices">{fa.navInvoices}</a>
				</div>
			{/if}
		</section>
	{/if}
</div>
