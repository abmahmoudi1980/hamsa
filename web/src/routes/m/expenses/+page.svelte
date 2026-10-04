<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { EXPENSE_CATEGORIES, listExpenses } from '#lib/api/endpoints/expenses';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import {
		approvalStatusLabel,
		approvalStatusTone,
		expenseCategoryLabel
	} from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const PAGE_SIZE = 20;

	const buildingId = $derived(buildingSelection.id ?? '');

	let category = $state('');
	let approval = $state('');
	let from = $state('');
	let to = $state('');
	let page = $state(1);

	const filterKey = $derived(JSON.stringify({ category, approval, from, to, page }));

	const expensesQuery = createQuery(() => ({
		queryKey: qk.expenses.list(buildingId, filterKey),
		queryFn: () =>
			listExpenses(buildingId, {
				category: category || undefined,
				approval: approval || undefined,
				from: from || undefined,
				to: to || undefined,
				page,
				page_size: PAGE_SIZE
			}),
		enabled: !!buildingId
	}));

	const expenses = $derived(expensesQuery.data?.items ?? []);
	const total = $derived(expensesQuery.data?.total ?? 0);

	function resetFilters() {
		category = '';
		approval = '';
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
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navExpenses}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.expenseListDescription}</p>
		</div>
		{#if buildingId}
			<div class="flex flex-wrap items-center gap-3">
				<a class="btn-secondary min-h-11" href="/m/report">{fa.financialReport}</a>
				<a class="btn-primary min-h-11 gap-2" href="/m/expenses/new">
					<span class="text-xl leading-none" aria-hidden="true">＋</span>
					{fa.newExpense}
				</a>
			</div>
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
				<label class="block text-xs font-semibold" for="expense-filter-category"
					>{fa.category}</label
				>
				<select
					id="expense-filter-category"
					class="input-base bg-white"
					bind:value={category}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allExpenseCategories}</option>
					{#each EXPENSE_CATEGORIES as value (value)}
						<option {value}>{expenseCategoryLabel(value)}</option>
					{/each}
				</select>
			</div>
			<div class="space-y-1.5">
				<label class="block text-xs font-semibold" for="expense-filter-approval"
					>{fa.approvalStatus}</label
				>
				<select
					id="expense-filter-approval"
					class="input-base bg-white"
					bind:value={approval}
					onchange={() => (page = 1)}
				>
					<option value="">{fa.allApprovalStatuses}</option>
					<option value="pending">{approvalStatusLabel('pending')}</option>
					<option value="approved">{approvalStatusLabel('approved')}</option>
					<option value="rejected">{approvalStatusLabel('rejected')}</option>
				</select>
			</div>
			<JalaliDateInput id="expense-filter-from" label={fa.fromDate} bind:value={from} />
			<JalaliDateInput id="expense-filter-to" label={fa.toDate} bind:value={to} />
			<div class="flex items-end justify-end sm:col-span-2 lg:col-span-4">
				<button class="btn-secondary min-h-10" type="button" onclick={resetFilters}
					>{fa.reset}</button
				>
			</div>
		</form>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if expensesQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3, 4, 5] as row (row)}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{/each}
				</div>
			{:else if expensesQuery.isError}
				<div class="p-8 text-center">
					<p class="text-lg font-bold">{fa.errorGeneric}</p>
					<p class="mt-2 text-sm text-on-surface-variant">
						{expensesQuery.error instanceof ApiError
							? expensesQuery.error.message
							: fa.errorGeneric}
					</p>
					<button
						class="mt-5 btn-primary"
						type="button"
						onclick={() => void expensesQuery.refetch()}>{fa.retry}</button
					>
				</div>
			{:else if expenses.length}
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
								<th class="px-5 py-3 text-start font-semibold">{fa.expenseDate}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.expenseTitle}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.category}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.expenseAmount}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.approvalStatus}</th>
								<th class="px-5 py-3 text-start font-semibold">
									<span class="sr-only">{fa.receiptFile}</span>
								</th>
								<th class="px-5 py-3 text-start font-semibold">
									<span class="sr-only">{fa.expenseDetail}</span>
								</th>
							</tr>
						</thead>
						<tbody>
							{#each expenses as expense (expense.id)}
								<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
									<td class="px-5 py-3">{dateLabel(expense.expense_date)}</td>
									<td class="px-5 py-3">
										<a
											class="font-semibold text-primary hover:underline"
											href={`/m/expenses/${expense.id}`}>{expense.title}</a
										>
									</td>
									<td class="px-5 py-3">{expenseCategoryLabel(expense.category)}</td>
									<td class="px-5 py-3 font-semibold">{formatTomanWithUnit(expense.amount)}</td>
									<td class="px-5 py-3">
										<StatusChip
											tone={approvalStatusTone(expense.approval_status)}
											label={approvalStatusLabel(expense.approval_status)}
										/>
									</td>
									<td class="px-5 py-3">
										{#if expense.receipt_file}
											<svg
												viewBox="0 0 24 24"
												fill="none"
												class="size-5 text-on-surface-variant"
												aria-hidden="true"
											>
												<title>{fa.receiptFile}</title>
												<path
													d="M21 11.5 12.5 20a5 5 0 0 1-7-7l8.5-8.5a3.5 3.5 0 0 1 5 5L10.5 18a2 2 0 0 1-3-3l8-8"
													stroke="currentColor"
													stroke-width="1.7"
													stroke-linecap="round"
													stroke-linejoin="round"
												/>
											</svg>
										{:else}
											—
										{/if}
									</td>
									<td class="px-5 py-3 text-end">
										<a
											class="text-xs font-semibold text-primary hover:underline"
											href={`/m/expenses/${expense.id}`}>{fa.detail}</a
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
					<p class="text-lg font-bold">{fa.noExpenses}</p>
					<a class="mt-5 btn-primary" href="/m/expenses/new">{fa.newExpense}</a>
				</div>
			{/if}
		</section>
	{/if}
</div>
