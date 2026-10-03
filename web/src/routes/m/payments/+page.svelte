<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listPayments } from '#lib/api/endpoints/payments';
	import { listUnits } from '#lib/api/endpoints/units';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import { paymentMethodLabel, paymentStatusLabel, paymentStatusTone } from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;

	const buildingId = $derived(buildingSelection.id ?? '');

	let unitId = $state('');
	let method = $state('');
	let from = $state('');
	let to = $state('');
	let page = $state(1);

	const filterKey = $derived(JSON.stringify({ unitId, method, from, to, page }));

	const paymentsQuery = createQuery(() => ({
		queryKey: qk.payments.ledger(buildingId, filterKey),
		queryFn: () =>
			listPayments(buildingId, {
				unit_id: unitId || undefined,
				method: method || undefined,
				from: from || undefined,
				to: to || undefined,
				page,
				page_size: PAGE_SIZE
			}),
		enabled: !!buildingId
	}));

	const unitsQuery = createQuery(() => ({
		queryKey: qk.units.list(buildingId, 'payment-filter'),
		queryFn: () => listUnits(buildingId, { page_size: 100 }),
		enabled: !!buildingId
	}));

	const payments = $derived(paymentsQuery.data?.items ?? []);
	const total = $derived(paymentsQuery.data?.total ?? 0);

	function resetFilters() {
		unitId = '';
		method = '';
		from = '';
		to = '';
		page = 1;
	}

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.ledger}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.ledgerDescription}</p>
		</div>
		{#if buildingId}
			<a class="btn-primary min-h-11 gap-2" href="/m/payments/new">
				<span class="text-xl leading-none" aria-hidden="true">＋</span>
				{fa.recordPayment}
			</a>
		{/if}
	</header>

	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.noBuildingsHint}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else}
		<form
			class="grid gap-3 rounded-2xl border border-outline-variant/60 bg-white p-4 sm:grid-cols-2 lg:grid-cols-4"
			onsubmit={(event) => event.preventDefault()}
		>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="ledger-unit">{fa.filterByUnit}</label>
				<select
					id="ledger-unit"
					class="input-base bg-white"
					bind:value={unitId}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allUnits}</option>
					{#each unitsQuery.data?.items ?? [] as unit (unit.id)}
						<option value={unit.id}>{toPersianDigits(unit.number)}</option>
					{/each}
				</select>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="ledger-method">{fa.paymentMethod}</label>
				<select
					id="ledger-method"
					class="input-base bg-white"
					bind:value={method}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allMethods}</option>
					<option value="manual">{fa.methodManual}</option>
					<option value="gateway">{fa.methodGateway}</option>
					<option value="transfer">{fa.methodTransfer}</option>
				</select>
			</div>
			<JalaliDateInput id="ledger-from" label={fa.fromDate} bind:value={from} />
			<JalaliDateInput id="ledger-to" label={fa.toDate} bind:value={to} />
			<div class="flex items-end justify-end sm:col-span-2 lg:col-span-4">
				<button class="btn-secondary min-h-10" type="button" onclick={resetFilters}
					>{fa.reset}</button
				>
			</div>
		</form>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if paymentsQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3, 4, 5] as row (row)}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{/each}
				</div>
			{:else if paymentsQuery.isError}
				<div class="p-8 text-center">
					<p class="text-lg font-bold">{fa.errorGeneric}</p>
					<p class="mt-2 text-sm text-on-surface-variant">
						{paymentsQuery.error instanceof ApiError
							? paymentsQuery.error.message
							: fa.errorGeneric}
					</p>
					<button
						class="mt-5 btn-primary"
						type="button"
						onclick={() => void paymentsQuery.refetch()}>{fa.retry}</button
					>
				</div>
			{:else if payments.length}
				<div
					class="flex items-center gap-2 border-b border-outline-variant/50 px-5 py-3 text-sm text-on-surface-variant"
				>
					<span class="rounded-lg bg-primary/10 px-2 py-0.5 font-bold text-primary"
						>{toPersianDigits(String(total))}</span
					>
					{fa.resultsCount}
				</div>
				<div class="overflow-x-auto">
					<table class="w-full min-w-[52rem] text-sm">
						<thead class="bg-surface/70 text-xs text-on-surface-variant">
							<tr>
								<th class="px-5 py-3 text-start font-semibold">{fa.paymentDate}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.unit}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.paymentAmount}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.paymentMethod}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.paymentStatus}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.trackingNumber}</th>
								<th class="px-5 py-3 text-start font-semibold">
									<span class="sr-only">{fa.invoice}</span>
								</th>
							</tr>
						</thead>
						<tbody>
							{#each payments as payment (payment.id)}
								<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
									<td class="px-5 py-3">{dateLabel(payment.paid_at)}</td>
									<td class="px-5 py-3">{toPersianDigits(payment.unit_number ?? '—')}</td>
									<td class="px-5 py-3 font-semibold">{formatTomanWithUnit(payment.amount)}</td>
									<td class="px-5 py-3">{paymentMethodLabel(payment.method)}</td>
									<td class="px-5 py-3">
										<StatusChip
											tone={paymentStatusTone(payment.status)}
											label={paymentStatusLabel(payment.status)}
										/>
									</td>
									<td class="px-5 py-3">
										{#if payment.tracking_number}
											<span class="num">{payment.tracking_number}</span>
										{:else}
											—
										{/if}
									</td>
									<td class="px-5 py-3 text-end">
										{#if payment.invoice_id}
											<a
												class="text-xs font-semibold text-primary hover:underline"
												href={`/m/payments/new?invoice_id=${payment.invoice_id}`}
												>{fa.recordPayment}</a
											>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<Pagination {page} pageSize={PAGE_SIZE} {total} onChange={(next) => (page = next)} />
			{:else}
				<div class="p-10 text-center">
					<p class="text-lg font-bold">{fa.emptyPaymentsTitle}</p>
					<p class="mt-2 text-sm text-on-surface-variant">{fa.noPaymentsHint}</p>
				</div>
			{/if}
		</section>
	{/if}
</div>
