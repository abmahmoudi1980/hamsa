<script lang="ts">
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		calculatePeriod,
		closePeriod,
		deleteCostItem,
		getPeriod,
		getPreview,
		issuePeriod,
		listCostItems,
		reopenPeriod,
		type CostItem
	} from '#lib/api/endpoints/billing';
	import CostItemForm from '#lib/components/CostItemForm.svelte';
	import PreviewGrid from '#lib/components/PreviewGrid.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import {
		calcMethodLabel,
		lateFeeTypeLabel,
		periodStatusLabel,
		periodStatusTone
	} from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const periodId = $derived(page.params.id ?? '');

	const periodQuery = createQuery(() => ({
		queryKey: qk.periods.detail(periodId),
		queryFn: () => getPeriod(periodId),
		enabled: !!periodId
	}));
	const period = $derived(periodQuery.data);
	const status = $derived(period?.status ?? '');
	const buildingId = $derived(period?.building_id ?? '');

	const costItemsQuery = createQuery(() => ({
		queryKey: qk.periods.costItems(periodId),
		queryFn: () => listCostItems(periodId),
		enabled: !!periodId
	}));
	const costItems = $derived(costItemsQuery.data ?? []);

	// Shares only exist after a calculation; previewing a draft would show
	// meaningless zeros and a false "not reconciled" flag.
	const previewQuery = createQuery(() => ({
		queryKey: qk.periods.preview(periodId),
		queryFn: () => getPreview(periodId),
		enabled: !!periodId && status !== 'draft'
	}));
	const preview = $derived(previewQuery.data);

	let actionError = $state('');
	let showNewForm = $state(false);
	let editingItem = $state<CostItem | null>(null);

	const removeMutation = createMutation(() => ({
		mutationFn: (id: string) => deleteCostItem(id),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.periods.costItems(periodId) }),
				queryClient.invalidateQueries({ queryKey: qk.periods.preview(periodId) })
			]);
		}
	}));

	const calculateMutation = createMutation(() => ({
		mutationFn: () => calculatePeriod(periodId),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['periods'] });
		}
	}));

	const reopenMutation = createMutation(() => ({
		mutationFn: () => reopenPeriod(periodId),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['periods'] });
		}
	}));

	const issueMutation = createMutation(() => ({
		mutationFn: () => issuePeriod(periodId),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ['periods'] }),
				queryClient.invalidateQueries({ queryKey: ['invoices'] }),
				queryClient.invalidateQueries({ queryKey: ['balances'] }),
				queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
				queryClient.invalidateQueries({ queryKey: ['notifications'] })
			]);
		}
	}));

	const closeMutation = createMutation(() => ({
		mutationFn: () => closePeriod(periodId),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['periods'] });
		}
	}));

	async function run(action: () => Promise<unknown>) {
		actionError = '';
		try {
			await action();
		} catch (error) {
			actionError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
	}

	async function calculate() {
		await run(() => calculateMutation.mutateAsync());
	}

	async function removeItem(item: CostItem) {
		if (!confirm(fa.confirmDeleteCostItem)) return;
		await run(() => removeMutation.mutateAsync(item.id));
	}

	async function reopen() {
		if (!confirm(fa.reopenPeriod)) return;
		await run(() => reopenMutation.mutateAsync());
	}

	async function issue() {
		const count = preview?.invoices.length ?? 0;
		const message = `${fa.issueConfirm} (${toPersianDigits(String(count))} ${fa.invoice})`;
		if (!confirm(message)) return;
		await run(() => issueMutation.mutateAsync());
	}

	async function close() {
		if (!confirm(fa.closePeriod)) return;
		await run(() => closeMutation.mutateAsync());
	}

	function openNew() {
		editingItem = null;
		showNewForm = true;
	}

	function openEdit(item: CostItem) {
		showNewForm = false;
		editingItem = item;
	}

	function doneForm() {
		showNewForm = false;
		editingItem = null;
	}

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}

	function lateFeeText(): string {
		if (!period) return '—';
		if (!period.late_fee_type || period.late_fee_type === 'none') return fa.noLateFee;
		const label = lateFeeTypeLabel(period.late_fee_type);
		const value =
			period.late_fee_type === 'percent'
				? `${toPersianDigits(String(period.late_fee_value))}٪`
				: formatTomanWithUnit(String(period.late_fee_value));
		return `${label}: ${value}`;
	}
</script>

<div class="mx-auto max-w-6xl space-y-6">
	<a
		href="/m/periods"
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
		{fa.backToPeriodsList}
	</a>

	{#if periodQuery.isPending}
		<div class="h-72 animate-pulse rounded-3xl bg-white"></div>
	{:else if periodQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{periodQuery.error instanceof ApiError ? periodQuery.error.message : fa.errorGeneric}
			</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void periodQuery.refetch()}
				>{fa.retry}</button
			>
		</section>
	{:else if period}
		<header class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<p class="text-sm font-semibold text-primary">{fa.periodDetail}</p>
				<h1 class="mt-1 flex items-center gap-3 text-2xl font-bold tracking-tight sm:text-3xl">
					<span>{period.title}</span>
					<StatusChip
						tone={periodStatusTone(period.status)}
						label={periodStatusLabel(period.status)}
					/>
				</h1>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				{#if status === 'draft'}
					<a class="btn-secondary min-h-10" href={`/m/periods/${periodId}/edit`}>{fa.editPeriod}</a>
					<button
						class="btn-primary min-h-10"
						type="button"
						disabled={costItems.length === 0 || calculateMutation.isPending}
						onclick={() => void calculate()}
					>
						{calculateMutation.isPending ? fa.calculating : fa.calculate}
					</button>
				{:else if status === 'calculated'}
					<button
						class="btn-secondary min-h-10"
						type="button"
						disabled={calculateMutation.isPending}
						onclick={() => void calculate()}
					>
						{calculateMutation.isPending ? fa.calculating : fa.recalculate}
					</button>
					<button
						class="btn-secondary min-h-10"
						type="button"
						disabled={reopenMutation.isPending}
						onclick={() => void reopen()}
					>
						{reopenMutation.isPending ? fa.reopening : fa.reopenPeriod}
					</button>
					<button
						class="btn-primary min-h-10"
						type="button"
						disabled={issueMutation.isPending || !preview}
						onclick={() => void issue()}
					>
						{issueMutation.isPending ? fa.issuing : fa.issueInvoices}
					</button>
				{:else if status === 'issued'}
					<button
						class="btn-secondary min-h-10"
						type="button"
						disabled={closeMutation.isPending}
						onclick={() => void close()}
					>
						{closeMutation.isPending ? fa.closing : fa.closePeriod}
					</button>
				{/if}
			</div>
		</header>

		{#if actionError}
			<p
				class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
				role="alert"
			>
				{actionError}
			</p>
		{/if}

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.periodInfo}</h2>
			<dl class="mt-4 grid gap-4 sm:grid-cols-3">
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.startDatePeriod}</dt>
					<dd class="mt-1 font-semibold">{dateLabel(period.start_date)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.endDatePeriod}</dt>
					<dd class="mt-1 font-semibold">{dateLabel(period.end_date)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.dueDate}</dt>
					<dd class="mt-1 font-semibold">{dateLabel(period.due_date)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.lateFee}</dt>
					<dd class="mt-1 font-semibold">{lateFeeText()}</dd>
				</div>
				{#if period.issued_at}
					<div>
						<dt class="text-xs text-on-surface-variant">{fa.issuedAt}</dt>
						<dd class="mt-1 font-semibold">{dateLabel(period.issued_at)}</dd>
					</div>
				{/if}
				{#if period.calculated_at}
					<div>
						<dt class="text-xs text-on-surface-variant">{fa.lastCalculatedAt}</dt>
						<dd class="mt-1 font-semibold">{dateLabel(period.calculated_at)}</dd>
					</div>
				{/if}
			</dl>
		</section>

		<section class="space-y-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h2 class="text-lg font-bold">{fa.costItems}</h2>
				{#if status === 'draft' && !showNewForm && !editingItem}
					<button class="btn-primary min-h-10 gap-2" type="button" onclick={openNew}>
						<span class="text-xl leading-none" aria-hidden="true">＋</span>
						{fa.addCostItem}
					</button>
				{/if}
			</div>

			{#if status === 'draft' && (showNewForm || editingItem)}
				<CostItemForm
					{periodId}
					{buildingId}
					initial={editingItem ?? undefined}
					onDone={doneForm}
					onCancel={doneForm}
				/>
			{/if}

			<div class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
				{#if costItemsQuery.isPending}
					<div class="space-y-2 p-5">
						{#each [1, 2, 3] as row (row)}
							<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
						{/each}
					</div>
				{:else if costItems.length === 0}
					<div class="p-8 text-center">
						<p class="text-lg font-bold">{fa.noCostItems}</p>
						<p class="mt-2 text-sm text-on-surface-variant">{fa.noCostItemsHint}</p>
					</div>
				{:else}
					<div class="overflow-x-auto">
						<table class="w-full min-w-[44rem] text-sm">
							<thead class="bg-surface/70 text-xs text-on-surface-variant">
								<tr>
									<th class="px-5 py-3 text-start font-semibold">{fa.costItemTitle}</th>
									<th class="px-5 py-3 text-start font-semibold">{fa.calcMethod}</th>
									<th class="px-5 py-3 text-start font-semibold">{fa.totalAmount}</th>
									<th class="px-5 py-3 text-start font-semibold">{fa.includeVacant}</th>
									{#if status === 'draft'}
										<th class="px-5 py-3 text-start font-semibold">
											<span class="sr-only">{fa.edit}</span>
										</th>
									{/if}
								</tr>
							</thead>
							<tbody>
								{#each costItems as item (item.id)}
									<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
										<td class="px-5 py-3 font-bold">{item.title}</td>
										<td class="px-5 py-3">{calcMethodLabel(item.method)}</td>
										<td class="px-5 py-3">
											{formatTomanWithUnit(item.total_amount)}
											{#if item.method === 'fixed' && item.fixed_amount_per_unit}
												<span class="block text-xs text-on-surface-variant">
													{fa.fixedAmountPerUnit}: {formatTomanWithUnit(item.fixed_amount_per_unit)}
												</span>
											{/if}
										</td>
										<td class="px-5 py-3">{item.include_vacant ? fa.yes : fa.no}</td>
										{#if status === 'draft'}
											<td class="px-5 py-3">
												<div class="flex items-center justify-end gap-3">
													<button
														class="text-xs font-semibold text-primary hover:underline"
														type="button"
														onclick={() => openEdit(item)}
													>
														{fa.edit}
													</button>
													<button
														class="text-xs font-semibold text-danger hover:underline"
														type="button"
														disabled={removeMutation.isPending}
														onclick={() => void removeItem(item)}
													>
														{fa.delete}
													</button>
												</div>
											</td>
										{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</section>

		{#if status === 'draft'}
			<section
				class="rounded-2xl border border-dashed border-outline-variant bg-surface/50 p-6 text-center"
			>
				{#if costItems.length === 0}
					<p class="text-sm font-semibold">{fa.noCostItemsHint}</p>
				{:else}
					<p class="text-sm font-semibold">{fa.mustCalculateFirst}</p>
					<p class="mt-1 text-xs text-on-surface-variant">{fa.previewHint}</p>
				{/if}
			</section>
		{:else if previewQuery.isPending}
			<div class="h-72 animate-pulse rounded-3xl bg-white"></div>
		{:else if previewQuery.isError}
			<section class="card p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{previewQuery.error instanceof ApiError ? previewQuery.error.message : fa.errorGeneric}
				</p>
				<button class="mt-5 btn-primary" type="button" onclick={() => void previewQuery.refetch()}
					>{fa.retry}</button
				>
			</section>
		{:else if preview}
			<PreviewGrid {preview} />
		{/if}
	{/if}
</div>
