<script lang="ts">
	/**
	 * Billing period create/edit form. Dates bind as Gregorian ISO (the wire
	 * format) but are entered as Solar Hijri through `JalaliDateInput`.
	 */
	import { goto } from '$app/navigation';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		createPeriod,
		updatePeriod,
		type BillingPeriod,
		type PeriodInput
	} from '#lib/api/endpoints/billing';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import { latinDecimalOnly } from '#lib/format/digits';
	import { parseTomanInput, tomanToNumber } from '#lib/format/money';
	import { fa } from '#i18n/fa';

	interface Props {
		buildingId: string;
		/** Present in edit mode. */
		periodId?: string;
		initial?: BillingPeriod;
	}

	let { buildingId, periodId, initial }: Props = $props();

	const queryClient = useQueryClient();

	let title = $state('');
	let startDate = $state('');
	let endDate = $state('');
	let dueDate = $state('');
	let lateFeeType = $state('none');
	let lateFeeValue = $state('');

	let titleError = $state('');
	let formError = $state('');

	let seeded = false;
	$effect(() => {
		if (initial && !seeded) {
			title = initial.title;
			startDate = initial.start_date;
			endDate = initial.end_date;
			dueDate = initial.due_date;
			lateFeeType = initial.late_fee_type || 'none';
			lateFeeValue = initial.late_fee_value ? String(initial.late_fee_value) : '';
			seeded = true;
		}
	});

	function parseFee(): { value?: number; error?: string } {
		if (lateFeeType === 'none') return { value: 0 };
		if (lateFeeType === 'percent') {
			const raw = latinDecimalOnly(lateFeeValue);
			const percent = Number(raw);
			if (raw === '' || !Number.isFinite(percent) || percent <= 0 || percent > 100) {
				return { error: fa.invalidAmount };
			}
			return { value: percent };
		}
		try {
			const parsed = parseTomanInput(lateFeeValue);
			if (!parsed) return { error: fa.required };
			const amount = tomanToNumber(parsed);
			if (amount <= 0) return { error: fa.invalidAmount };
			return { value: amount };
		} catch {
			return { error: fa.invalidAmount };
		}
	}

	const mutation = createMutation(() => ({
		mutationFn: (input: PeriodInput) =>
			periodId ? updatePeriod(periodId, input) : createPeriod(buildingId, input),
		onSuccess: async (period) => {
			await queryClient.invalidateQueries({ queryKey: ['periods', buildingId] });
			await goto(`/m/periods/${period.id}`, { replace: true });
		}
	}));

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		titleError = '';
		formError = '';

		if (title.trim() === '') {
			titleError = fa.required;
			return;
		}
		if (!startDate || !endDate || !dueDate) {
			formError = fa.required;
			return;
		}
		// ISO dates compare correctly as strings.
		if (endDate < startDate || dueDate < endDate) {
			formError = fa.periodDatesInvalid;
			return;
		}
		const fee = parseFee();
		if (fee.error) {
			formError = fee.error;
			return;
		}

		const input: PeriodInput = {
			title: title.trim(),
			start_date: startDate,
			end_date: endDate,
			due_date: dueDate,
			late_fee_type: lateFeeType,
			late_fee_value: fee.value ?? 0
		};

		try {
			await mutation.mutateAsync(input);
		} catch (error) {
			formError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
	}
</script>

<form
	class="overflow-hidden rounded-3xl border border-outline-variant/60 bg-white shadow-sm"
	onsubmit={submit}
	novalidate
>
	<div class="grid gap-5 p-5 sm:grid-cols-2 sm:p-8">
		<div class="space-y-2 sm:col-span-2">
			<label class="block text-sm font-semibold" for="period-title"
				>{fa.periodTitle}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="period-title"
				class={`input-base bg-white ${titleError ? 'border-danger' : ''}`}
				bind:value={title}
				placeholder={fa.periodTitle}
				aria-invalid={titleError ? 'true' : undefined}
			/>
			{#if titleError}
				<p class="text-xs text-danger" aria-live="polite">{titleError}</p>
			{/if}
		</div>

		<JalaliDateInput id="period-start" label={fa.startDatePeriod} bind:value={startDate} required />
		<JalaliDateInput id="period-end" label={fa.endDatePeriod} bind:value={endDate} required />
		<JalaliDateInput id="period-due" label={fa.dueDate} bind:value={dueDate} required />

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="period-late-fee">{fa.lateFeeMode}</label>
			<select id="period-late-fee" class="input-base bg-white" bind:value={lateFeeType}>
				<option value="none">{fa.noLateFee}</option>
				<option value="fixed">{fa.lateFeeFixed}</option>
				<option value="percent">{fa.lateFeePercent}</option>
				<option value="per_day">{fa.lateFeePerDay}</option>
			</select>
		</div>

		{#if lateFeeType !== 'none'}
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="period-late-fee-value"
					>{fa.lateFeeValue}</label
				>
				<input
					id="period-late-fee-value"
					class="input-base bg-white"
					bind:value={lateFeeValue}
					inputmode="numeric"
					dir="ltr"
				/>
			</div>
		{/if}

		{#if formError}
			<p
				class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger sm:col-span-2"
				role="alert"
			>
				{formError}
			</p>
		{/if}
	</div>

	<div
		class="flex flex-col-reverse gap-3 border-t border-outline-variant/50 bg-surface/70 p-5 sm:flex-row sm:justify-end sm:px-8"
	>
		<a class="btn-secondary" href={periodId ? `/m/periods/${periodId}` : '/m/periods'}
			>{fa.cancel}</a
		>
		<button class="btn-primary" type="submit" disabled={mutation.isPending}>
			{mutation.isPending ? fa.saving : fa.save}
		</button>
	</div>
</form>
