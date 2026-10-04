<script lang="ts">
	import { goto } from '$app/navigation';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import ExpenseForm from '#lib/components/ExpenseForm.svelte';
	import { fa } from '#i18n/fa';

	const buildingId = $derived(buildingSelection.id ?? '');
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
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.newExpense}</h1>
		<p class="mt-2 text-sm leading-6 text-on-surface-variant">{fa.newExpenseDescription}</p>
	</header>

	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else}
		<ExpenseForm
			{buildingId}
			onDone={(expense) => void goto(`/m/expenses/${expense.id}`, { replace: true })}
			onCancel={() => void goto('/m/expenses')}
		/>
	{/if}
</div>
