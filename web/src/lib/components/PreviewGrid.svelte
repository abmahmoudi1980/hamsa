<script lang="ts">
	/**
	 * The pre-issuance review grid: per-unit resulting invoices with a
	 * drill-down into each cost item's exact and rounded share, plus the
	 * BR-08/BR-09 reconciliation footer (Σ rounded shares vs. the amount the
	 * engine actually distributed).
	 */
	import type { Preview } from '#lib/api/endpoints/billing';
	import { formatExactShare, reconcilePreview, shareForUnit } from '#lib/billing/reconcile';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { calcMethodLabel } from '#lib/format/labels';
	import { formatTomanWithUnit, sumToman } from '#lib/format/money';
	import { fa } from '#i18n/fa';

	interface Props {
		preview: Preview;
	}

	let { preview }: Props = $props();

	const recon = $derived(reconcilePreview(preview));
	const totalUsed = $derived(sumToman(recon.items.map((item) => item.totalUsed)));
	const totalRounded = $derived(sumToman(recon.items.map((item) => item.roundedSum)));

	let expandedUnit = $state<string | null>(null);

	function toggle(unitId: string) {
		expandedUnit = expandedUnit === unitId ? null : unitId;
	}

	function unitLabel(unitNumber: string | undefined, unitId: string): string {
		return unitNumber ? toPersianDigits(unitNumber) : unitId.slice(0, 8);
	}
</script>

<div class="space-y-6">
	<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
		<div
			class="flex flex-wrap items-center justify-between gap-3 border-b border-outline-variant/50 px-5 py-4"
		>
			<h2 class="text-lg font-bold">{fa.preview}</h2>
			<StatusChip
				tone={recon.reconciled ? 'success' : 'danger'}
				label={recon.reconciled ? fa.reconciledLabel : fa.unreconciledLabel}
			/>
		</div>

		{#if preview.invoices.length === 0}
			<p class="p-6 text-sm text-on-surface-variant">{fa.previewEmpty}</p>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full min-w-[52rem] text-sm">
					<thead class="bg-surface/70 text-xs text-on-surface-variant">
						<tr>
							<th class="px-5 py-3 text-start font-semibold">{fa.unit}</th>
							<th class="px-5 py-3 text-start font-semibold">{fa.baseAmount}</th>
							<th class="px-5 py-3 text-start font-semibold">{fa.priorDebt}</th>
							<th class="px-5 py-3 text-start font-semibold">{fa.lateFeeAmount}</th>
							<th class="px-5 py-3 text-start font-semibold">{fa.credit}</th>
							<th class="px-5 py-3 text-start font-semibold">{fa.finalAmount}</th>
							<th class="px-5 py-3 text-start font-semibold">
								<span class="sr-only">{fa.unitShares}</span>
							</th>
						</tr>
					</thead>
					<tbody>
						{#each preview.invoices as invoice (invoice.id)}
							{@const open = expandedUnit === invoice.unit_id}
							<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
								<td class="px-5 py-3 font-bold">
									{fa.unit}
									{unitLabel(invoice.unit_number, invoice.unit_id)}
								</td>
								<td class="px-5 py-3">{formatTomanWithUnit(invoice.base_amount)}</td>
								<td class="px-5 py-3">{formatTomanWithUnit(invoice.prior_debt)}</td>
								<td class="px-5 py-3">{formatTomanWithUnit(invoice.late_fee_amount)}</td>
								<td class="px-5 py-3">{formatTomanWithUnit(invoice.credit_amount)}</td>
								<td class="px-5 py-3 font-bold text-primary">
									{formatTomanWithUnit(invoice.final_amount)}
								</td>
								<td class="px-5 py-3 text-end">
									<button
										class="text-xs font-semibold text-primary hover:underline"
										type="button"
										aria-expanded={open}
										onclick={() => toggle(invoice.unit_id)}
									>
										{open ? fa.close : fa.unitShares}
									</button>
								</td>
							</tr>
							{#if open}
								<tr class="border-t border-outline-variant/30 bg-surface/40">
									<td colspan="7" class="px-5 py-4">
										<table class="w-full text-xs">
											<thead class="text-on-surface-variant">
												<tr>
													<th class="py-1 text-start font-semibold">{fa.costItemTitle}</th>
													<th class="py-1 text-start font-semibold">{fa.calcMethod}</th>
													<th class="py-1 text-start font-semibold">{fa.totalUsed}</th>
													<th class="py-1 text-start font-semibold">{fa.exactShare}</th>
													<th class="py-1 text-start font-semibold">{fa.roundedShare}</th>
												</tr>
											</thead>
											<tbody>
												{#each preview.items as item (item.id)}
													{@const share = shareForUnit(item, invoice.unit_id)}
													{#if share}
														<tr class="border-t border-outline-variant/30">
															<td class="py-1.5 font-semibold">{item.title}</td>
															<td class="py-1.5">{calcMethodLabel(item.method)}</td>
															<td class="py-1.5">{formatTomanWithUnit(item.total_used)}</td>
															<td class="num py-1.5">{formatExactShare(share.exact_share)}</td>
															<td class="py-1.5 font-semibold">
																{formatTomanWithUnit(share.rounded_share)}
															</td>
														</tr>
													{/if}
												{/each}
											</tbody>
										</table>
									</td>
								</tr>
							{/if}
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<section
		class="sticky bottom-0 overflow-hidden rounded-2xl border border-outline-variant/60 bg-white shadow-[0_-8px_24px_-18px_rgba(0,0,0,0.35)]"
	>
		<div
			class="flex flex-wrap items-center justify-between gap-3 border-b border-outline-variant/50 px-5 py-4"
		>
			<h2 class="text-lg font-bold">{fa.reconciliation}</h2>
			<StatusChip
				tone={recon.reconciled ? 'success' : 'danger'}
				label={recon.reconciled ? fa.reconciledLabel : fa.unreconciledLabel}
			/>
		</div>
		<div class="overflow-x-auto">
			<table class="w-full min-w-[40rem] text-sm">
				<thead class="bg-surface/70 text-xs text-on-surface-variant">
					<tr>
						<th class="px-5 py-3 text-start font-semibold">{fa.costItemTitle}</th>
						<th class="px-5 py-3 text-start font-semibold">{fa.calcMethod}</th>
						<th class="px-5 py-3 text-start font-semibold">{fa.totalUsed}</th>
						<th class="px-5 py-3 text-start font-semibold">{fa.sumRoundedShares}</th>
						<th class="px-5 py-3 text-start font-semibold">{fa.residualLabel}</th>
						<th class="px-5 py-3 text-start font-semibold">{fa.status}</th>
					</tr>
				</thead>
				<tbody>
					{#each preview.items as item, index (item.id)}
						{@const check = recon.items[index]}
						{#if check}
							<tr class="border-t border-outline-variant/40">
								<td class="px-5 py-3 font-semibold">{item.title}</td>
								<td class="px-5 py-3">{calcMethodLabel(item.method)}</td>
								<td class="px-5 py-3">{formatTomanWithUnit(check.totalUsed)}</td>
								<td class="px-5 py-3">{formatTomanWithUnit(check.roundedSum)}</td>
								<td
									class={`px-5 py-3 font-semibold ${
										check.reconciled ? 'text-success' : 'text-danger'
									}`}
								>
									{formatTomanWithUnit(check.residual)}
								</td>
								<td class="px-5 py-3">
									<StatusChip
										tone={check.reconciled ? 'success' : 'danger'}
										label={check.reconciled ? fa.reconciledLabel : fa.unreconciledLabel}
									/>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
				<tfoot class="border-t-2 border-outline-variant/60 bg-surface/50 font-bold">
					<tr>
						<td class="px-5 py-4">{fa.overallReconciliation}</td>
						<td class="px-5 py-4">—</td>
						<td class="px-5 py-4">{formatTomanWithUnit(totalUsed)}</td>
						<td class="px-5 py-4">{formatTomanWithUnit(totalRounded)}</td>
						<td class={`px-5 py-4 ${recon.reconciled ? 'text-success' : 'text-danger'}`}>
							{formatTomanWithUnit(recon.residual)}
						</td>
						<td class="px-5 py-4">
							<StatusChip
								tone={recon.reconciled ? 'success' : 'danger'}
								label={recon.reconciled ? fa.reconciledLabel : fa.unreconciledLabel}
							/>
						</td>
					</tr>
				</tfoot>
			</table>
		</div>
	</section>
</div>
