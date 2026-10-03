<script lang="ts">
	/**
	 * Manager "record payment" form (FR-022). Partial payments are allowed and
	 * a surplus becomes unit credit; the amount is validated as a positive
	 * integer Toman before it is sent.
	 */
	import { untrack } from 'svelte';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { recordManualPayment, type ManualPaymentInput } from '#lib/api/endpoints/invoices';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import { toIsoDate } from '#lib/format/jalali';
	import {
		formatTomanWithUnit,
		parseTomanInput,
		tomanToNumber,
		type Toman
	} from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	interface Props {
		invoiceId: string;
		buildingId: string;
		/** Pre-fills the amount; omitted → the field starts empty. */
		outstanding?: Toman;
		onDone: () => void;
		onCancel: () => void;
	}

	let { invoiceId, buildingId, outstanding = '' as Toman, onDone, onCancel }: Props = $props();

	const queryClient = useQueryClient();

	let amount = $state<string>(untrack(() => outstanding));
	let paidAt = $state(toIsoDate(new Date()));
	let trackingNumber = $state('');
	let formError = $state('');

	const mutation = createMutation(() => ({
		mutationFn: (input: ManualPaymentInput) => recordManualPayment(invoiceId, input),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.invoices.detail(invoiceId) }),
				queryClient.invalidateQueries({ queryKey: ['invoices', buildingId] }),
				queryClient.invalidateQueries({ queryKey: ['payments', 'ledger', buildingId] }),
				queryClient.invalidateQueries({ queryKey: ['payments', 'me'] }),
				queryClient.invalidateQueries({ queryKey: ['balances'] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(buildingId) })
			]);
			onDone();
		}
	}));

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		formError = '';

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

		const input: ManualPaymentInput = { amount: parsed };
		if (paidAt) input.paid_at = paidAt;
		if (trackingNumber.trim()) input.tracking_number = trackingNumber.trim();

		try {
			await mutation.mutateAsync(input);
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
	<h3 class="text-lg font-bold">{fa.recordPayment}</h3>
	<p class="text-xs text-on-surface-variant">{fa.partialPaymentHint}</p>

	<div class="grid gap-4 sm:grid-cols-2">
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="payment-amount"
				>{fa.paymentAmount}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="payment-amount"
				class={`input-base bg-white ${formError ? 'border-danger' : ''}`}
				bind:value={amount}
				inputmode="numeric"
				dir="ltr"
			/>
			{#if outstanding}
				<p class="text-xs text-on-surface-variant">
					{fa.outstanding}: {formatTomanWithUnit(outstanding)}
				</p>
			{/if}
		</div>

		<JalaliDateInput id="payment-date" label={fa.paymentDate} bind:value={paidAt} />

		<div class="space-y-2 sm:col-span-2">
			<label class="block text-sm font-semibold" for="payment-tracking"
				>{fa.trackingNumber}
				<span class="text-xs font-normal text-on-surface-variant">({fa.optional})</span></label
			>
			<input
				id="payment-tracking"
				class="input-base bg-white"
				bind:value={trackingNumber}
				dir="ltr"
			/>
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
			{mutation.isPending ? fa.saving : fa.recordPayment}
		</button>
	</div>
</form>
