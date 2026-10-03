<script lang="ts">
	/**
	 * Explicit invoice correction (FR-017): appends a debit or credit note
	 * without touching the frozen invoice amounts. A reason is required.
	 */
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		addAdjustment,
		type AdjustmentInput,
		type AdjustmentKind
	} from '#lib/api/endpoints/invoices';
	import { parseTomanInput, tomanToNumber, type Toman } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	interface Props {
		invoiceId: string;
		buildingId: string;
		onDone: () => void;
		onCancel: () => void;
	}

	let { invoiceId, buildingId, onDone, onCancel }: Props = $props();

	const queryClient = useQueryClient();

	let kind = $state<AdjustmentKind>('debit');
	let amount = $state('');
	let reason = $state('');
	let formError = $state('');

	const mutation = createMutation(() => ({
		mutationFn: (input: AdjustmentInput) => addAdjustment(invoiceId, input),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.invoices.detail(invoiceId) }),
				queryClient.invalidateQueries({ queryKey: ['invoices', buildingId] }),
				queryClient.invalidateQueries({ queryKey: ['balances'] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(buildingId) })
			]);
			onDone();
		}
	}));

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		formError = '';

		if (reason.trim() === '') {
			formError = fa.adjustmentReason + ' ' + fa.required;
			return;
		}

		let parsed: Toman | null;
		try {
			parsed = parseTomanInput(amount);
		} catch {
			formError = fa.invalidAmount;
			return;
		}
		if (!parsed || tomanToNumber(parsed) <= 0) {
			formError = fa.invalidAmount;
			return;
		}

		try {
			await mutation.mutateAsync({ kind, amount: parsed, reason: reason.trim() });
		} catch (caught) {
			formError = caught instanceof ApiError ? caught.message : fa.errorGeneric;
		}
	}
</script>

<form
	class="space-y-5 rounded-2xl border border-primary/30 bg-white p-5 sm:p-6"
	onsubmit={submit}
	novalidate
>
	<h3 class="text-lg font-bold">{fa.addAdjustment}</h3>

	<div class="grid gap-4 sm:grid-cols-2">
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="adjustment-kind">{fa.adjustmentKind}</label>
			<select id="adjustment-kind" class="input-base bg-white" bind:value={kind}>
				<option value="debit">{fa.adjustmentDebit}</option>
				<option value="credit">{fa.adjustmentCredit}</option>
			</select>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="adjustment-amount"
				>{fa.adjustmentAmount}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="adjustment-amount"
				class={`input-base bg-white ${formError ? 'border-danger' : ''}`}
				bind:value={amount}
				inputmode="numeric"
				dir="ltr"
			/>
		</div>

		<div class="space-y-2 sm:col-span-2">
			<label class="block text-sm font-semibold" for="adjustment-reason"
				>{fa.adjustmentReason}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<textarea id="adjustment-reason" class="min-h-24 input-base bg-white" bind:value={reason}
			></textarea>
		</div>
	</div>

	{#if formError}
		<p
			class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
			role="alert"
		>
			{formError}
		</p>
	{/if}

	<div
		class="flex flex-col-reverse gap-3 border-t border-outline-variant/50 pt-4 sm:flex-row sm:justify-end"
	>
		<button class="btn-secondary" type="button" onclick={onCancel}>{fa.cancel}</button>
		<button class="btn-primary" type="submit" disabled={mutation.isPending}>
			{mutation.isPending ? fa.saving : fa.save}
		</button>
	</div>
</form>
