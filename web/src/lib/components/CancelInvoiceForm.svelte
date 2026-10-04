<script lang="ts">
	/**
	 * Cancel an invoice with a required reason (BR-10). The row is preserved —
	 * cancellation is a status change, never a delete.
	 */
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { cancelInvoice } from '#lib/api/endpoints/invoices';
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

	let reason = $state('');
	let formError = $state('');

	const mutation = createMutation(() => ({
		mutationFn: (value: string) => cancelInvoice(invoiceId, value),
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
			formError = fa.cancelReason + ' ' + fa.required;
			return;
		}
		try {
			await mutation.mutateAsync(reason.trim());
		} catch (caught) {
			formError = caught instanceof ApiError ? caught.message : fa.errorGeneric;
		}
	}
</script>

<form
	class="space-y-5 rounded-2xl border border-danger/30 bg-white p-5 sm:p-6"
	onsubmit={submit}
	novalidate
>
	<h3 class="text-lg font-bold text-danger">{fa.cancelInvoice}</h3>
	<p class="text-xs text-on-surface-variant">{fa.issueInvoicesHint}</p>

	<div class="space-y-2">
		<label class="block text-sm font-semibold" for="cancel-reason"
			>{fa.cancelReason}<span class="text-danger" aria-hidden="true">*</span></label
		>
		<textarea id="cancel-reason" class="min-h-24 input-base bg-white" bind:value={reason}
		></textarea>
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
		<button class="btn-secondary" type="button" onclick={onCancel}>{fa.back}</button>
		<button
			class="btn-primary bg-danger shadow-[0_8px_20px_-10px_rgba(198,40,40,0.75)] hover:bg-danger/90"
			type="submit"
			disabled={mutation.isPending}
		>
			{mutation.isPending ? fa.saving : fa.cancelInvoice}
		</button>
	</div>
</form>
