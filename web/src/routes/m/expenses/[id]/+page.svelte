<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		deleteExpense,
		getExpense,
		updateExpense,
		type ApprovalStatus
	} from '#lib/api/endpoints/expenses';
	import { listPersons } from '#lib/api/endpoints/persons';
	import AttachmentPreview from '#lib/components/AttachmentPreview.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { fromIsoDate, formatJalaliDate } from '#lib/format/jalali';
	import {
		approvalStatusLabel,
		approvalStatusTone,
		expenseCategoryLabel
	} from '#lib/format/labels';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const expenseId = $derived(page.params.id ?? '');

	const expenseQuery = createQuery(() => ({
		queryKey: qk.expenses.detail(expenseId),
		queryFn: () => getExpense(expenseId),
		enabled: !!expenseId
	}));

	const expense = $derived(expenseQuery.data);
	const buildingId = $derived(expense?.building_id ?? '');

	const personsQuery = createQuery(() => ({
		queryKey: qk.persons.list(buildingId, 'expense-payer'),
		queryFn: () => listPersons(buildingId, { page_size: 100 }),
		enabled: !!buildingId
	}));

	const payerName = $derived(
		expense?.payer_person_id
			? (personsQuery.data?.items.find((p) => p.id === expense.payer_person_id)?.full_name ?? '—')
			: null
	);

	const statusMutation = createMutation(() => ({
		mutationFn: (status: ApprovalStatus) => updateExpense(expenseId, { approval_status: status }),
		onSuccess: async (updated) => {
			queryClient.setQueryData(qk.expenses.detail(expenseId), updated);
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ['expenses', buildingId] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(buildingId) })
			]);
		}
	}));

	const deleteMutation = createMutation(() => ({
		mutationFn: () => deleteExpense(expenseId),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ['expenses', buildingId] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(buildingId) })
			]);
			await goto('/m/expenses', { replace: true });
		}
	}));

	async function setStatus(status: ApprovalStatus) {
		try {
			await statusMutation.mutateAsync(status);
		} catch {
			/* the chip keeps the server's state; a refetch will reconcile */
		}
	}

	async function remove() {
		if (!confirm(fa.confirmDeleteExpense)) return;
		await deleteMutation.mutateAsync();
	}

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}
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

	{#if expenseQuery.isPending}
		<div class="h-72 animate-pulse rounded-3xl bg-white"></div>
	{:else if expenseQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{expenseQuery.error instanceof ApiError ? expenseQuery.error.message : fa.errorGeneric}
			</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void expenseQuery.refetch()}
				>{fa.retry}</button
			>
		</section>
	{:else if expense}
		<header class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<p class="text-sm font-semibold text-primary">{fa.expenseDetail}</p>
				<h1
					class="mt-1 flex flex-wrap items-center gap-3 text-2xl font-bold tracking-tight sm:text-3xl"
				>
					<span>{expense.title}</span>
					<StatusChip
						tone={approvalStatusTone(expense.approval_status)}
						label={approvalStatusLabel(expense.approval_status)}
					/>
				</h1>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<a class="btn-secondary min-h-10" href={`/m/expenses/${expenseId}/edit`}>{fa.edit}</a>
				<button
					class="btn-secondary min-h-10 text-danger"
					type="button"
					disabled={deleteMutation.isPending}
					onclick={() => void remove()}
				>
					{fa.delete}
				</button>
			</div>
		</header>

		{#if expense.approval_status !== 'approved'}
			<button
				class="btn-primary min-h-10"
				type="button"
				disabled={statusMutation.isPending}
				onclick={() => void setStatus('approved')}
			>
				{fa.approveExpense}
			</button>
		{/if}
		{#if expense.approval_status !== 'rejected'}
			<button
				class="btn-secondary min-h-10 text-danger"
				type="button"
				disabled={statusMutation.isPending}
				onclick={() => void setStatus('rejected')}
			>
				{fa.rejectExpense}
			</button>
		{/if}
		<p class="text-xs text-on-surface-variant">{fa.expenseApprovalHint}</p>

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.expenseInfo}</h2>
			<dl class="mt-4 grid gap-4 sm:grid-cols-2">
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.expenseAmount}</dt>
					<dd class="mt-1 font-semibold">{formatTomanWithUnit(expense.amount)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.category}</dt>
					<dd class="mt-1 font-semibold">{expenseCategoryLabel(expense.category)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.expenseDate}</dt>
					<dd class="mt-1 font-semibold">{dateLabel(expense.expense_date)}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.payer}</dt>
					<dd class="mt-1 font-semibold">{payerName ?? '—'}</dd>
				</div>
				{#if expense.description}
					<div class="sm:col-span-2">
						<dt class="text-xs text-on-surface-variant">{fa.description}</dt>
						<dd class="mt-1 leading-7">{expense.description}</dd>
					</div>
				{/if}
			</dl>
		</section>

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.receiptFile}</h2>
			<div class="mt-4">
				<AttachmentPreview fileRef={expense.receipt_file} label={expense.title} />
			</div>
		</section>
	{/if}
</div>
