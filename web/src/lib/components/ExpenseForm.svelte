<script lang="ts">
	/**
	 * Expense create/edit form (US6, contracts/api.md "Expenses").
	 *
	 * The receipt is uploaded first (`POST /files`) and the returned file id is
	 * bound on submit via `receipt_file_id`; the preview goes through
	 * `AttachmentPreview` (F2), never a bare `/files/...` URL.
	 */
	import { untrack } from 'svelte';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		EXPENSE_CATEGORIES,
		createExpense,
		updateExpense,
		type Expense,
		type ExpenseInput
	} from '#lib/api/endpoints/expenses';
	import { ACCEPTED_UPLOAD_TYPES, MAX_UPLOAD_BYTES, uploadFile } from '#lib/api/endpoints/files';
	import { listPersons } from '#lib/api/endpoints/persons';
	import AttachmentPreview from '#lib/components/AttachmentPreview.svelte';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import { toIsoDate } from '#lib/format/jalali';
	import { approvalStatusLabel, expenseCategoryLabel } from '#lib/format/labels';
	import { parseTomanInput, tomanToNumber, type Toman } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	interface Props {
		buildingId: string;
		/** Present in edit mode. */
		expenseId?: string;
		/** Present in edit mode; seeds the fields. */
		initial?: Expense;
		onDone: (expense: Expense) => void;
		onCancel: () => void;
	}

	let { buildingId, expenseId, initial, onDone, onCancel }: Props = $props();

	const queryClient = useQueryClient();

	let title = $state(untrack(() => initial?.title ?? ''));
	let category = $state(untrack(() => initial?.category ?? 'other'));
	let amount = $state(untrack(() => (initial ? String(initial.amount) : '')));
	let expenseDate = $state(untrack(() => initial?.expense_date ?? toIsoDate(new Date())));
	let payerPersonId = $state(untrack(() => initial?.payer_person_id ?? ''));
	let description = $state(untrack(() => initial?.description ?? ''));
	let approvalStatus = $state(untrack(() => initial?.approval_status ?? 'pending'));

	/** A newly uploaded file id (replaces any existing receipt on submit). */
	let uploadedFileId = $state<string | null>(null);
	/** MIME type of the freshly uploaded file, so the preview can render it. */
	let uploadedFileType = $state<string | null>(null);
	/** True when the user removed the existing receipt without uploading a new one. */
	let receiptCleared = $state(false);
	let uploading = $state(false);
	let receiptError = $state('');

	let titleError = $state('');
	let amountError = $state('');
	let formError = $state('');

	const personsQuery = createQuery(() => ({
		queryKey: qk.persons.list(buildingId, 'expense-payer'),
		queryFn: () => listPersons(buildingId, { page_size: 100 }),
		enabled: !!buildingId
	}));

	const existingReceipt = $derived(initial?.receipt_file ?? null);
	const previewRef = $derived(uploadedFileId ?? (receiptCleared ? null : existingReceipt));

	const mutation = createMutation(() => ({
		mutationFn: (input: ExpenseInput) =>
			expenseId ? updateExpense(expenseId, input) : createExpense(buildingId, input),
		onSuccess: async (expense) => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ['expenses', buildingId] }),
				queryClient.invalidateQueries({ queryKey: qk.dashboard.manager(buildingId) })
			]);
			onDone(expense);
		}
	}));

	function onFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		receiptError = '';
		if (!file) return;
		if (file.size > MAX_UPLOAD_BYTES) {
			receiptError = fa.receiptTooLarge;
			input.value = '';
			return;
		}
		const allowed = file.type.startsWith('image/') || file.type === 'application/pdf';
		if (!allowed) {
			receiptError = fa.receiptInvalidType;
			input.value = '';
			return;
		}
		uploading = true;
		uploadFile(file)
			.then((uploaded) => {
				uploadedFileId = uploaded.id;
				uploadedFileType = file.type;
				receiptCleared = false;
			})
			.catch((error) => {
				receiptError = error instanceof ApiError ? error.message : fa.uploadFailed;
			})
			.finally(() => {
				uploading = false;
				input.value = '';
			});
	}

	function removeReceipt() {
		uploadedFileId = null;
		uploadedFileType = null;
		if (existingReceipt) receiptCleared = true;
		receiptError = '';
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		titleError = '';
		amountError = '';
		formError = '';

		if (title.trim() === '') {
			titleError = fa.required;
			return;
		}

		let parsed: Toman | null;
		try {
			parsed = parseTomanInput(amount);
		} catch {
			amountError = fa.invalidAmount;
			return;
		}
		if (!parsed || tomanToNumber(parsed) <= 0) {
			amountError = fa.invalidAmount;
			return;
		}

		const input: ExpenseInput = {
			title: title.trim(),
			category,
			amount: tomanToNumber(parsed),
			expense_date: expenseDate,
			approval_status: approvalStatus,
			description: description.trim() || null,
			payer_person_id: payerPersonId || null
		};

		if (uploadedFileId) input.receipt_file_id = uploadedFileId;
		else if (receiptCleared) input.receipt_file_id = '';

		try {
			await mutation.mutateAsync(input);
		} catch (error) {
			if (error instanceof ApiError) formError = error.message;
			else formError = fa.errorGeneric;
		}
	}
</script>

<form
	class="overflow-hidden rounded-3xl border border-outline-variant/60 bg-white shadow-sm"
	onsubmit={submit}
	novalidate
>
	<div class="grid gap-5 p-5 sm:p-8">
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="expense-title"
				>{fa.expenseTitle}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="expense-title"
				class={`input-base bg-white ${titleError ? 'border-danger' : ''}`}
				bind:value={title}
				placeholder={fa.expenseTitle}
				aria-invalid={titleError ? 'true' : undefined}
			/>
			{#if titleError}<p class="text-xs text-danger" aria-live="polite">{titleError}</p>{/if}
		</div>

		<div class="grid gap-5 sm:grid-cols-2">
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="expense-category">{fa.category}</label>
				<select id="expense-category" class="input-base bg-white" bind:value={category}>
					{#each EXPENSE_CATEGORIES as value (value)}
						<option {value}>{expenseCategoryLabel(value)}</option>
					{/each}
				</select>
			</div>

			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="expense-amount"
					>{fa.expenseAmount}<span class="text-danger" aria-hidden="true">*</span></label
				>
				<input
					id="expense-amount"
					class={`input-base bg-white ${amountError ? 'border-danger' : ''}`}
					bind:value={amount}
					inputmode="numeric"
					dir="ltr"
					aria-invalid={amountError ? 'true' : undefined}
				/>
				{#if amountError}<p class="text-xs text-danger" aria-live="polite">{amountError}</p>{/if}
			</div>

			<JalaliDateInput id="expense-date" label={fa.expenseDate} required bind:value={expenseDate} />

			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="expense-approval">{fa.approvalStatus}</label
				>
				<select id="expense-approval" class="input-base bg-white" bind:value={approvalStatus}>
					<option value="pending">{approvalStatusLabel('pending')}</option>
					<option value="approved">{approvalStatusLabel('approved')}</option>
					<option value="rejected">{approvalStatusLabel('rejected')}</option>
				</select>
			</div>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="expense-payer">{fa.payerOptional}</label>
			<select id="expense-payer" class="input-base bg-white" bind:value={payerPersonId}>
				<option value="">{fa.noPayer}</option>
				{#each personsQuery.data?.items ?? [] as person (person.id)}
					<option value={person.id}>{person.full_name}</option>
				{/each}
			</select>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="expense-description">{fa.description}</label>
			<textarea
				id="expense-description"
				class="min-h-24 input-base bg-white"
				bind:value={description}></textarea>
		</div>

		<div class="space-y-3">
			<p class="text-sm font-semibold">{fa.receiptFile}</p>
			{#if previewRef}
				<div class="space-y-2">
					<AttachmentPreview
						fileRef={previewRef}
						label={fa.receipt}
						contentType={uploadedFileType ?? ''}
					/>
					<button
						class="btn-secondary min-h-9 text-xs text-danger"
						type="button"
						onclick={removeReceipt}
					>
						{fa.removeReceipt}
					</button>
				</div>
			{:else}
				<label
					class="flex cursor-pointer flex-col items-center gap-2 rounded-2xl border border-dashed border-outline-variant bg-surface/60 px-4 py-6 text-center text-sm text-on-surface-variant hover:border-primary/50"
				>
					<span class="font-semibold text-primary">{fa.uploadReceipt}</span>
					<span class="text-xs">{fa.chooseReceiptFile}</span>
					<input
						class="sr-only"
						type="file"
						accept={ACCEPTED_UPLOAD_TYPES}
						onchange={onFileChange}
						disabled={uploading}
					/>
				</label>
			{/if}
			{#if uploading}<p class="text-xs text-on-surface-variant">{fa.loading}</p>{/if}
			{#if receiptError}<p class="text-xs text-danger" role="alert">{receiptError}</p>{/if}
		</div>

		{#if formError}
			<p
				class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
				role="alert"
			>
				{formError}
			</p>
		{/if}
	</div>

	<div
		class="flex flex-col-reverse gap-3 border-t border-outline-variant/50 bg-surface/70 p-5 sm:flex-row sm:justify-end sm:px-8"
	>
		<button class="btn-secondary" type="button" onclick={onCancel}>{fa.cancel}</button>
		<button class="btn-primary" type="submit" disabled={mutation.isPending || uploading}>
			{mutation.isPending ? fa.saving : fa.save}
		</button>
	</div>
</form>
