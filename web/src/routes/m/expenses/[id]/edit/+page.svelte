<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { getExpense } from '#lib/api/endpoints/expenses';
	import ExpenseForm from '#lib/components/ExpenseForm.svelte';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const expenseId = $derived(page.params.id ?? '');

	const expenseQuery = createQuery(() => ({
		queryKey: qk.expenses.detail(expenseId),
		queryFn: () => getExpense(expenseId),
		enabled: !!expenseId
	}));

	const expense = $derived(expenseQuery.data);
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		href={`/m/expenses/${expenseId}`}
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
		<p class="text-sm font-semibold text-primary">{fa.expenseDetail}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.editExpense}</h1>
		<p class="mt-2 text-sm leading-6 text-on-surface-variant">{fa.editExpenseDescription}</p>
	</header>

	{#if expenseQuery.isPending}
		<div class="h-64 animate-pulse rounded-3xl bg-white"></div>
	{:else if expenseQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{expenseQuery.error instanceof ApiError ? expenseQuery.error.message : fa.errorGeneric}
			</p>
		</section>
	{:else if expense}
		<ExpenseForm
			buildingId={expense.building_id}
			{expenseId}
			initial={expense}
			onDone={() => void goto(`/m/expenses/${expenseId}`, { replace: true })}
			onCancel={() => void goto(`/m/expenses/${expenseId}`)}
		/>
	{:else}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
		</section>
	{/if}
</div>
