<script lang="ts">
	/**
	 * Cost-item create/edit form. The method drives which fields matter:
	 * `equal`/`per_occupant`/`per_area` take a total, `fixed` a per-unit amount,
	 * `specific_units` a total plus an explicit selection, and `combined` a total
	 * plus weights that must sum to 100 (checked live by `weightsProblem`).
	 */
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		createCostItem,
		updateCostItem,
		type CalcMethod,
		type ComboWeight,
		type CostItem,
		type CostItemInput
	} from '#lib/api/endpoints/billing';
	import UnitMultiSelect from '#lib/components/UnitMultiSelect.svelte';
	import {
		addWeightRow as appendWeightRow,
		removeWeightRow,
		weightsProblem,
		weightSum,
		COMBO_METHODS,
		type ComboMethod,
		type WeightRow
	} from '#lib/billing/weights';
	import { latinDigitsOnly, toPersianDigits } from '#lib/format/digits';
	import { calcMethodLabel } from '#lib/format/labels';
	import { parseTomanInput, tomanToNumber, type Toman } from '#lib/format/money';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	interface Props {
		periodId: string;
		buildingId: string;
		initial?: CostItem;
		onDone: () => void;
		onCancel: () => void;
	}

	let { periodId, buildingId, initial, onDone, onCancel }: Props = $props();

	const queryClient = useQueryClient();

	const methodOptions: { value: CalcMethod; label: string }[] = [
		{ value: 'equal', label: fa.methodEqual },
		{ value: 'per_occupant', label: fa.methodPerPerson },
		{ value: 'per_area', label: fa.methodPerArea },
		{ value: 'fixed', label: fa.methodFixed },
		{ value: 'specific_units', label: fa.methodSpecificUnits },
		{ value: 'combined', label: fa.methodCombined }
	];

	let title = $state('');
	let method = $state<CalcMethod>('equal');
	let totalAmount = $state('');
	let fixedPerUnit = $state('');
	let includeVacant = $state(false);
	let unitIds = $state<string[]>([]);
	let weights = $state<WeightRow[]>([]);

	let titleError = $state('');
	let formError = $state('');

	let seeded = false;
	$effect(() => {
		if (initial && !seeded) {
			title = initial.title;
			method = initial.method as CalcMethod;
			totalAmount = initial.total_amount ?? '';
			fixedPerUnit = initial.fixed_amount_per_unit ?? '';
			includeVacant = initial.include_vacant;
			unitIds = initial.unit_ids ?? [];
			weights = (initial.combo_weights ?? []).map((weight) => ({
				method: weight.method as ComboMethod,
				weight: String(weight.weight)
			}));
			seeded = true;
		}
	});

	const weightsIssue = $derived(method === 'combined' ? weightsProblem(weights) : null);
	const weightsSum = $derived(weightSum(weights));

	function changeMethod(event: Event) {
		const next = (event.currentTarget as HTMLSelectElement).value as CalcMethod;
		method = next;
		if (next === 'combined' && weights.length === 0) {
			weights = appendWeightRow([]);
		}
	}

	function positiveToman(raw: string): Toman | null {
		try {
			const parsed = parseTomanInput(raw);
			if (!parsed || tomanToNumber(parsed) <= 0) return null;
			return parsed;
		} catch {
			return null;
		}
	}

	function weightsMessage(): string {
		switch (weightsIssue) {
			case 'empty':
			case 'non_positive':
			case 'sum':
				return fa.weightsInvalid;
			default:
				return '';
		}
	}

	function buildInput(): { input?: CostItemInput; error?: string } {
		const base = { title: title.trim(), include_vacant: includeVacant };
		if (method === 'fixed') {
			const fixed = positiveToman(fixedPerUnit);
			if (!fixed) return { error: fa.invalidAmount };
			return { input: { ...base, method, fixed_amount_per_unit: fixed } };
		}

		const total = positiveToman(totalAmount);
		if (!total) return { error: fa.invalidAmount };

		if (method === 'specific_units') {
			if (unitIds.length === 0) return { error: fa.selectUnits };
			return { input: { ...base, method, total_amount: total, unit_ids: unitIds } };
		}

		if (method === 'combined') {
			if (weightsIssue) return { error: fa.weightsInvalid };
			const comboWeights: ComboWeight[] = weights.map((row) => ({
				method: row.method,
				weight: Number(latinDigitsOnly(row.weight))
			}));
			return { input: { ...base, method, total_amount: total, combo_weights: comboWeights } };
		}

		return { input: { ...base, method, total_amount: total } };
	}

	const mutation = createMutation(() => ({
		mutationFn: (input: CostItemInput) =>
			initial ? updateCostItem(initial.id, input) : createCostItem(periodId, input),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.periods.costItems(periodId) }),
				queryClient.invalidateQueries({ queryKey: qk.periods.preview(periodId) })
			]);
			onDone();
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

		const { input, error } = buildInput();
		if (!input) {
			formError = error ?? fa.invalidAmount;
			return;
		}

		try {
			await mutation.mutateAsync(input);
		} catch (caught) {
			formError = caught instanceof ApiError ? caught.message : fa.errorGeneric;
		}
	}

	const showTotal = $derived(
		method === 'equal' ||
			method === 'per_occupant' ||
			method === 'per_area' ||
			method === 'specific_units' ||
			method === 'combined'
	);
</script>

<form
	class="space-y-5 rounded-2xl border border-primary/30 bg-white p-5 sm:p-6"
	onsubmit={submit}
	novalidate
>
	<h3 class="text-lg font-bold">{initial ? fa.editCostItem : fa.newCostItem}</h3>

	<div class="grid gap-4 sm:grid-cols-2">
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="cost-title"
				>{fa.costItemTitle}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="cost-title"
				class={`input-base bg-white ${titleError ? 'border-danger' : ''}`}
				bind:value={title}
				aria-invalid={titleError ? 'true' : undefined}
			/>
			{#if titleError}
				<p class="text-xs text-danger" aria-live="polite">{titleError}</p>
			{/if}
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="cost-method">{fa.calcMethod}</label>
			<select id="cost-method" class="input-base bg-white" value={method} onchange={changeMethod}>
				{#each methodOptions as option (option.value)}
					<option value={option.value}>{option.label}</option>
				{/each}
			</select>
		</div>

		{#if showTotal}
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="cost-total">{fa.totalAmount}</label>
				<input
					id="cost-total"
					class="input-base bg-white"
					bind:value={totalAmount}
					inputmode="numeric"
					dir="ltr"
				/>
			</div>
		{/if}

		{#if method === 'fixed'}
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="cost-fixed">{fa.fixedAmountPerUnit}</label>
				<input
					id="cost-fixed"
					class="input-base bg-white"
					bind:value={fixedPerUnit}
					inputmode="numeric"
					dir="ltr"
				/>
			</div>
		{/if}

		<label class="flex items-center gap-3 pt-2 text-sm font-semibold sm:col-span-2">
			<input type="checkbox" class="size-4 accent-primary" bind:checked={includeVacant} />
			{fa.includeVacant}
		</label>
	</div>

	{#if method === 'specific_units'}
		<UnitMultiSelect {buildingId} selected={unitIds} onChange={(ids) => (unitIds = ids)} />
	{/if}

	{#if method === 'combined'}
		<div class="space-y-3 rounded-2xl border border-outline-variant/60 bg-surface/50 p-4">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<p class="text-sm font-semibold">{fa.weights}</p>
				<span
					class={`rounded-full px-3 py-1 text-xs font-bold ${
						weightsIssue ? 'bg-danger-soft text-danger' : 'bg-success-soft text-success'
					}`}
				>
					{fa.weightSumLabel}:
					{Number.isNaN(weightsSum) ? '—' : `${toPersianDigits(String(weightsSum))}٪`}
				</span>
			</div>

			<ul class="space-y-2">
				{#each weights as row, index (index)}
					<li class="flex flex-wrap items-center gap-2">
						<select
							class="input-base max-w-52 bg-white"
							aria-label={fa.calcMethod}
							bind:value={row.method}
						>
							{#each COMBO_METHODS as comboMethod (comboMethod)}
								<option value={comboMethod}>{calcMethodLabel(comboMethod)}</option>
							{/each}
						</select>
						<div class="flex items-center gap-1">
							<input
								class="input-base max-w-28 bg-white"
								bind:value={row.weight}
								inputmode="numeric"
								dir="ltr"
								aria-label={fa.weightPercent}
							/>
							<span class="text-sm text-on-surface-variant">٪</span>
						</div>
						<button
							class="btn-secondary min-h-10 text-danger"
							type="button"
							onclick={() => (weights = removeWeightRow(weights, index))}
						>
							{fa.removeWeightRow}
						</button>
					</li>
				{/each}
			</ul>

			<div class="flex flex-wrap items-center justify-between gap-3">
				<button
					class="btn-secondary min-h-10"
					type="button"
					disabled={weights.length >= COMBO_METHODS.length}
					onclick={() => (weights = appendWeightRow(weights))}
				>
					{fa.addWeightRow}
				</button>
				<p class="text-xs text-on-surface-variant">{fa.weightsHint}</p>
			</div>
			{#if weightsIssue && weights.length > 0}
				<p class="text-xs text-danger" role="alert">{weightsMessage()}</p>
			{/if}
		</div>
	{/if}

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
