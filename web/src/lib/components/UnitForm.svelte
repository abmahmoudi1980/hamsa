<script lang="ts">
	import { goto } from '$app/navigation';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { z } from 'zod';
	import { ApiError } from '#lib/api/apiError';
	import {
		createUnit,
		updateUnit,
		type Unit,
		type UnitInput,
		type UnitStatus
	} from '#lib/api/endpoints/units';
	import { latinDigitsOnly } from '#lib/format/digits';
	import { fa } from '#i18n/fa';

	interface Props {
		buildingId: string;
		/** Present in edit mode. */
		unitId?: string;
		initial?: Unit;
	}

	let { buildingId, unitId, initial }: Props = $props();

	const queryClient = useQueryClient();

	const formSchema = z.object({
		number: z.string().trim().min(1).max(60),
		block: z.string().trim().max(60),
		floor: z.number().int().min(0).max(1000),
		area_m2: z.number().int().min(1).max(1_000_000),
		parking_count: z.number().int().min(0).max(1000),
		storage_count: z.number().int().min(0).max(1000)
	});

	let number = $state('');
	let block = $state('');
	let floor = $state('0');
	let areaM2 = $state('');
	let parkingCount = $state('0');
	let parkingNumbers = $state('');
	let storageCount = $state('0');
	let storageNumbers = $state('');
	let status = $state<UnitStatus>('active');
	let notes = $state('');

	let numberError = $state('');
	let formError = $state('');

	let seeded = false;
	$effect(() => {
		if (initial && !seeded) {
			number = initial.number;
			block = initial.block ?? '';
			floor = String(initial.floor);
			areaM2 = String(initial.area_m2);
			parkingCount = String(initial.parking_count);
			parkingNumbers = initial.parking_numbers ?? '';
			storageCount = String(initial.storage_count);
			storageNumbers = initial.storage_numbers ?? '';
			status = initial.status as UnitStatus;
			notes = initial.notes ?? '';
			seeded = true;
		}
	});

	function countValue(value: string): number {
		const digits = latinDigitsOnly(value);
		return digits ? Number(digits) : Number.NaN;
	}

	const mutation = createMutation(() => ({
		mutationFn: (input: UnitInput) =>
			unitId ? updateUnit(unitId, input) : createUnit(buildingId, input),
		onSuccess: async (unit) => {
			await queryClient.invalidateQueries({ queryKey: ['units'] });
			await goto(`/m/units/${unit.id}`, { replaceState: true });
		}
	}));

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		numberError = '';
		formError = '';

		const parsed = formSchema.safeParse({
			number,
			block,
			floor: countValue(floor),
			area_m2: countValue(areaM2),
			parking_count: countValue(parkingCount),
			storage_count: countValue(storageCount)
		});

		if (!parsed.success) {
			numberError = parsed.error.issues.some((issue) => issue.path[0] === 'number')
				? fa.required
				: '';
			formError = parsed.error.issues.some((issue) => issue.path[0] !== 'number')
				? fa.invalidNumber
				: '';
			return;
		}

		const input: UnitInput = {
			number: parsed.data.number,
			block: parsed.data.block || undefined,
			floor: parsed.data.floor,
			area_m2: parsed.data.area_m2,
			parking_count: parsed.data.parking_count,
			parking_numbers: parkingNumbers || undefined,
			storage_count: parsed.data.storage_count,
			storage_numbers: storageNumbers || undefined,
			status,
			notes: notes || undefined
		};

		try {
			await mutation.mutateAsync(input);
		} catch (error) {
			if (error instanceof ApiError) {
				// A 409 duplicate lands on the field the server named, with its
				// Persian message (never the locale-neutral rule token).
				if (error.details.some((detail) => detail.field === 'number')) {
					numberError = error.message;
				} else {
					formError = error.message;
				}
			} else {
				formError = fa.errorGeneric;
			}
		}
	}
</script>

<form
	class="overflow-hidden rounded-3xl border border-outline-variant/60 bg-white shadow-sm"
	onsubmit={submit}
	novalidate
>
	<div class="grid gap-5 p-5 sm:grid-cols-2 sm:p-8">
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-number"
				>{fa.unitNumber}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="unit-number"
				class={`input-base bg-white ${numberError ? 'border-danger' : ''}`}
				bind:value={number}
				placeholder={fa.unitNumberPlaceholder}
				aria-invalid={numberError ? 'true' : undefined}
				aria-describedby={numberError ? 'unit-number-error' : undefined}
			/>
			{#if numberError}
				<p id="unit-number-error" class="text-xs text-danger" aria-live="polite">{numberError}</p>
			{/if}
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-block">{fa.block}</label>
			<input
				id="unit-block"
				class="input-base bg-white"
				bind:value={block}
				placeholder={fa.blockPlaceholder}
			/>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-floor">{fa.floor}</label>
			<input
				id="unit-floor"
				class="input-base bg-white"
				bind:value={floor}
				inputmode="numeric"
				dir="ltr"
			/>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-area"
				>{fa.area}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="unit-area"
				class="input-base bg-white"
				bind:value={areaM2}
				inputmode="numeric"
				dir="ltr"
			/>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-parking-count">{fa.parkingCount}</label>
			<input
				id="unit-parking-count"
				class="input-base bg-white"
				bind:value={parkingCount}
				inputmode="numeric"
				dir="ltr"
			/>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-parking-numbers"
				>{fa.parkingNumber}</label
			>
			<input id="unit-parking-numbers" class="input-base bg-white" bind:value={parkingNumbers} />
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-storage-count">{fa.storageCount}</label>
			<input
				id="unit-storage-count"
				class="input-base bg-white"
				bind:value={storageCount}
				inputmode="numeric"
				dir="ltr"
			/>
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-storage-numbers"
				>{fa.storageNumber}</label
			>
			<input id="unit-storage-numbers" class="input-base bg-white" bind:value={storageNumbers} />
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="unit-status">{fa.status}</label>
			<select id="unit-status" class="input-base bg-white" bind:value={status}>
				<option value="active">{fa.statusActive}</option>
				<option value="vacant">{fa.statusVacant}</option>
				<option value="occupied">{fa.statusOccupied}</option>
				<option value="inactive">{fa.statusInactive}</option>
			</select>
		</div>

		<div class="space-y-2 sm:col-span-2">
			<label class="block text-sm font-semibold" for="unit-notes">{fa.description}</label>
			<textarea id="unit-notes" class="min-h-24 input-base bg-white" bind:value={notes}></textarea>
		</div>

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
		<a class="btn-secondary" href={unitId ? `/m/units/${unitId}` : '/m/units'}>{fa.cancel}</a>
		<button class="btn-primary" type="submit" disabled={mutation.isPending}>
			{mutation.isPending ? fa.saving : fa.save}
		</button>
	</div>
</form>
