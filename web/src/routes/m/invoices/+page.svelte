<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listPeriods } from '#lib/api/endpoints/billing';
	import { listInvoices } from '#lib/api/endpoints/invoices';
	import { listUnits } from '#lib/api/endpoints/units';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import { invoiceStatusLabel, invoiceStatusTone } from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;

	const buildingId = $derived(buildingSelection.id ?? '');

	let periodId = $state('');
	let unitId = $state('');
	let status = $state('');
	let page = $state(1);

	const filterKey = $derived(JSON.stringify({ periodId, unitId, status, page }));

	const invoicesQuery = createQuery(() => ({
		queryKey: qk.invoices.list(buildingId, filterKey),
		queryFn: () =>
			listInvoices(buildingId, {
				period_id: periodId || undefined,
				unit_id: unitId || undefined,
				status: status || undefined,
				page,
				page_size: PAGE_SIZE
			}),
		enabled: !!buildingId
	}));

	const periodsQuery = createQuery(() => ({
		queryKey: qk.periods.list(buildingId),
		queryFn: () => listPeriods(buildingId),
		enabled: !!buildingId
	}));

	const unitsQuery = createQuery(() => ({
		queryKey: qk.units.list(buildingId, 'invoice-filter'),
		queryFn: () => listUnits(buildingId, { page_size: 100 }),
		enabled: !!buildingId
	}));

	const invoices = $derived(invoicesQuery.data?.items ?? []);
	const total = $derived(invoicesQuery.data?.total ?? 0);

	function resetFilters() {
		periodId = '';
		unitId = '';
		status = '';
		page = 1;
	}

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
</script>

<div class="space-y-6">
	<header>
		<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navInvoices}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.invoiceListDescription}</p>
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
				<label class="block text-xs font-semibold" for="invoice-period">{fa.filterByPeriod}</label>
				<select
					id="invoice-period"
					class="input-base bg-white"
					bind:value={periodId}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allPeriods}</option>
					{#each periodsQuery.data ?? [] as period (period.id)}
						<option value={period.id}>{period.title}</option>
					{/each}
				</select>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="invoice-unit">{fa.filterByUnit}</label>
				<select
					id="invoice-unit"
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
				<label class="block text-xs font-semibold" for="invoice-status">{fa.status}</label>
				<select
					id="invoice-status"
					class="input-base bg-white"
					bind:value={status}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allStatuses}</option>
					<option value="unpaid">{invoiceStatusLabel('unpaid')}</option>
					<option value="partial">{invoiceStatusLabel('partial')}</option>
					<option value="paid">{invoiceStatusLabel('paid')}</option>
					<option value="cancelled">{invoiceStatusLabel('cancelled')}</option>
					<option value="expired">{invoiceStatusLabel('expired')}</option>
				</select>
			</div>
			<div class="flex items-end justify-end">
				<button class="btn-secondary min-h-10" type="button" onclick={resetFilters}
					>{fa.reset}</button
				>
			</div>
		</form>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if invoicesQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3, 4, 5] as row (row)}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{/each}
				</div>
			{:else if invoicesQuery.isError}
				<div class="p-8 text-center">
					<p class="text-lg font-bold">{fa.errorGeneric}</p>
					<p class="mt-2 text-sm text-on-surface-variant">
						{invoicesQuery.error instanceof ApiError
							? invoicesQuery.error.message
							: fa.errorGeneric}
					</p>
					<button
						class="mt-5 btn-primary"
						type="button"
						onclick={() => void invoicesQuery.refetch()}>{fa.retry}</button
					>
				</div>
			{:else if invoices.length}
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
								<th class="px-5 py-3 text-start font-semibold">{fa.invoiceNumber}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.unit}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.period}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.finalAmount}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.paidAmount}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.dueDateInvoice}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.status}</th>
								<th class="px-5 py-3 text-start font-semibold">
									<span class="sr-only">{fa.invoiceDetail}</span>
								</th>
							</tr>
						</thead>
						<tbody>
							{#each invoices as invoice (invoice.id)}
								<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
									<td class="px-5 py-3 font-bold">
										<a class="text-primary hover:underline" href={`/m/invoices/${invoice.id}`}
											>{invoice.invoice_number}</a
										>
									</td>
									<td class="px-5 py-3">{toPersianDigits(invoice.unit_number ?? '—')}</td>
									<td class="px-5 py-3">{invoice.period_title ?? '—'}</td>
									<td class="px-5 py-3 font-semibold"
										>{formatTomanWithUnit(invoice.final_amount)}</td
									>
									<td class="px-5 py-3">{formatTomanWithUnit(invoice.paid_amount)}</td>
									<td class="px-5 py-3">{dateLabel(invoice.due_date)}</td>
									<td class="px-5 py-3">
										<StatusChip
											tone={invoiceStatusTone(invoice.status)}
											label={invoiceStatusLabel(invoice.status)}
										/>
									</td>
									<td class="px-5 py-3 text-end">
										<a
											class="text-xs font-semibold text-primary hover:underline"
											href={`/m/invoices/${invoice.id}`}>{fa.invoiceDetail}</a
										>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<Pagination {page} pageSize={PAGE_SIZE} {total} onChange={(next) => (page = next)} />
			{:else}
				<div class="p-10 text-center">
					<p class="text-lg font-bold">{fa.emptyInvoicesTitle}</p>
					<p class="mt-2 text-sm text-on-surface-variant">{fa.emptyInvoicesBody}</p>
				</div>
			{/if}
		</section>
	{/if}
</div>
