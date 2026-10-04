<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getFinancialReport } from '#lib/api/endpoints/expenses';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import { monthKey, reportMonthLabel } from '#lib/expenses/report';
	import { fromIsoDate, toIsoDate } from '#lib/format/jalali';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const buildingId = $derived(buildingSelection.id ?? '');

	/** Any Jalali day the manager picks; the report is scoped to its month. */
	let monthRef = $state(toIsoDate(new Date()));
	const refDate = $derived(fromIsoDate(monthRef) ?? new Date());
	const month = $derived(monthKey(refDate));
	const monthLabel = $derived(reportMonthLabel(refDate));

	const reportQuery = createQuery(() => ({
		queryKey: qk.expenses.report(buildingId, month),
		queryFn: () => getFinancialReport(buildingId, month),
		enabled: !!buildingId
	}));

	const report = $derived(reportQuery.data);

	const rows = $derived(
		report
			? [
					{ label: fa.monthlyIncome, value: report.monthly_income, tone: 'income' as const },
					{ label: fa.monthlyExpense, value: report.monthly_expense, tone: 'expense' as const },
					{ label: fa.totalResidentDebt, value: report.total_debt, tone: 'expense' as const },
					{ label: fa.totalPayments, value: report.total_payments, tone: 'income' as const },
					{ label: fa.totalExpenses, value: report.total_expenses, tone: 'expense' as const }
				]
			: []
	);
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		href="/m/expenses"
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
		{fa.backToExpenses}
	</a>

	<header>
		<p class="text-sm font-semibold text-primary">{fa.navExpenses}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.financialReport}</h1>
		<p class="mt-2 text-sm leading-6 text-on-surface-variant">{fa.reportDescription}</p>
	</header>

	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else}
		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<div class="grid gap-4 sm:grid-cols-2 sm:items-end">
				<JalaliDateInput id="report-month" label={fa.reportMonth} bind:value={monthRef} />
				<div>
					<p class="text-xs text-on-surface-variant">{fa.reportMonthHint}</p>
					<p class="mt-2 text-xl font-bold text-primary">{monthLabel}</p>
				</div>
			</div>
		</section>

		{#if reportQuery.isPending}
			<div class="h-56 animate-pulse rounded-3xl bg-white"></div>
		{:else if reportQuery.isError}
			<section class="card p-8 text-center">
				<p class="text-lg font-bold">{fa.errorGeneric}</p>
				<p class="mt-2 text-sm text-on-surface-variant">
					{reportQuery.error instanceof ApiError ? reportQuery.error.message : fa.errorGeneric}
				</p>
				<button class="mt-5 btn-primary" type="button" onclick={() => void reportQuery.refetch()}
					>{fa.retry}</button
				>
			</section>
		{:else if report}
			<section
				class="rounded-3xl border p-6 sm:p-8 {report.net >= 0
					? 'border-primary/25 bg-primary-soft/30'
					: 'border-danger/25 bg-danger-soft/40'}"
			>
				<p class="text-sm font-semibold text-on-surface-variant">{fa.netIncome}</p>
				<p
					class="mt-2 text-3xl font-bold tracking-tight sm:text-4xl {report.net >= 0
						? 'text-primary'
						: 'text-danger'}"
				>
					{formatTomanWithUnit(report.net)}
				</p>
				<p class="mt-2 text-xs text-on-surface-variant">
					{report.net >= 0 ? fa.reportNetSurplus : fa.reportNetDeficit}
				</p>
			</section>

			<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
				<ul>
					{#each rows as row (row.label)}
						<li
							class="flex items-center justify-between gap-4 border-t border-outline-variant/40 px-5 py-4 first:border-t-0"
						>
							<span class="text-sm text-on-surface-variant">{row.label}</span>
							<span class="font-semibold {row.tone === 'income' ? 'text-primary' : 'text-danger'}">
								{formatTomanWithUnit(row.value)}
							</span>
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	{/if}
</div>
