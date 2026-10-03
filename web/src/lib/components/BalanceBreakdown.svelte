<script lang="ts">
	/**
	 * The unit-balance view (spec §9): all components of the balance, the
	 * formula rendered with the real numbers, and a reconciliation check that
	 * the displayed components reproduce the server's stored balance.
	 */
	import type { UnitBalance } from '#lib/api/endpoints/payments';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { formatTomanWithUnit } from '#lib/format/money';
	import { balanceFromComponents, balanceReconciles } from '#lib/payments/balance';
	import { fa } from '#i18n/fa';

	interface Props {
		balance: UnitBalance;
	}

	let { balance }: Props = $props();

	const computed = $derived(
		balanceFromComponents({
			prior_debt: balance.prior_debt,
			current_invoice_amount: balance.current_invoice_amount,
			late_fee_total: balance.late_fee_total,
			credit: balance.credit,
			paid_total: balance.paid_total
		})
	);
	const reconciled = $derived(balanceReconciles(balance));

	const rows = $derived([
		{ label: fa.priorDebt, value: balance.prior_debt },
		{ label: fa.currentInvoice, value: balance.current_invoice_amount },
		{ label: fa.lateFeeAmount, value: balance.late_fee_total },
		{ label: fa.paidTotal, value: balance.paid_total },
		{ label: fa.credit, value: balance.credit }
	]);
</script>

<section class="space-y-5">
	<div class="relative overflow-hidden rounded-3xl bg-primary p-6 text-on-primary sm:p-9">
		<p class="text-sm font-semibold text-white/75">{fa.outstanding}</p>
		<p class="mt-2 text-3xl font-bold tracking-tight sm:text-5xl">
			{formatTomanWithUnit(balance.balance)}
		</p>
		{#if balance.credit_asset !== '0'}
			<p class="mt-3 text-sm text-white/80">
				{fa.creditAsset}: {formatTomanWithUnit(balance.credit_asset)}
			</p>
		{/if}
	</div>

	<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h2 class="text-lg font-bold">{fa.balanceComponents}</h2>
			<StatusChip
				tone={reconciled ? 'success' : 'danger'}
				label={reconciled ? fa.balanceReconciled : fa.balanceMismatch}
			/>
		</div>

		<dl class="mt-4 divide-y divide-outline-variant/40">
			{#each rows as row (row.label)}
				<div class="flex items-center justify-between gap-4 py-3">
					<dt class="text-sm text-on-surface-variant">{row.label}</dt>
					<dd class="num font-semibold">{formatTomanWithUnit(row.value)}</dd>
				</div>
			{/each}
		</dl>

		<div class="mt-4 rounded-xl bg-surface px-4 py-4">
			<p class="text-xs font-semibold text-on-surface-variant">{fa.balanceFormula}</p>
			<p class="num mt-2 text-sm font-bold">{formatTomanWithUnit(computed)}</p>
		</div>
	</section>
</section>
