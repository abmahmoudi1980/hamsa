<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { archiveUnit, getUnit, getUnitHistory } from '#lib/api/endpoints/units';
	import {
		addOccupancy,
		endOccupancy,
		listOccupancies,
		listOccupantCounts,
		recordOccupantCount,
		type Relationship
	} from '#lib/api/endpoints/occupancy';
	import { listPersons } from '#lib/api/endpoints/persons';
	import JalaliDateInput from '#lib/components/JalaliDateInput.svelte';
	import StatusChip from '#lib/components/StatusChip.svelte';
	import { latinDigitsOnly, toPersianDigits } from '#lib/format/digits';
	import { formatJalaliDate, formatJalaliDateTime, fromIsoDate } from '#lib/format/jalali';
	import { relationshipLabel, unitStatusLabel, unitStatusTone } from '#lib/format/labels';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const unitId = $derived(page.params.id ?? '');

	const unitQuery = createQuery(() => ({
		queryKey: qk.units.detail(unitId),
		queryFn: () => getUnit(unitId),
		enabled: !!unitId
	}));
	const unit = $derived(unitQuery.data);
	const buildingId = $derived(unit?.building_id ?? '');

	const historyQuery = createQuery(() => ({
		queryKey: qk.units.history(unitId),
		queryFn: () => getUnitHistory(unitId),
		enabled: !!unitId
	}));

	const occupanciesQuery = createQuery(() => ({
		queryKey: qk.units.occupancies(unitId),
		queryFn: () => listOccupancies(unitId),
		enabled: !!unitId
	}));

	const countsQuery = createQuery(() => ({
		queryKey: qk.units.occupantCounts(unitId),
		queryFn: () => listOccupantCounts(unitId),
		enabled: !!unitId
	}));

	const personsQuery = createQuery(() => ({
		queryKey: qk.persons.list(buildingId, 'all'),
		queryFn: () => listPersons(buildingId, { page_size: 100 }),
		enabled: !!buildingId
	}));

	const occupancies = $derived(occupanciesQuery.data?.items ?? []);
	const activeOccupancies = $derived(occupancies.filter((o) => o.is_active));
	const persons = $derived(personsQuery.data?.items ?? []);

	// --- add occupancy ---
	let personId = $state('');
	let relationship = $state<Relationship>('owner');
	let startDate = $state('');
	let addError = $state('');

	const addOccMutation = createMutation(() => ({
		mutationFn: () =>
			addOccupancy(unitId, { person_id: personId, relationship, start_date: startDate }),
		onSuccess: async () => {
			personId = '';
			startDate = '';
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.units.occupancies(unitId) }),
				queryClient.invalidateQueries({ queryKey: qk.units.history(unitId) })
			]);
		}
	}));

	async function submitOccupancy(event: SubmitEvent) {
		event.preventDefault();
		addError = '';
		if (!personId || !startDate) {
			addError = fa.required;
			return;
		}
		try {
			await addOccMutation.mutateAsync();
		} catch (error) {
			addError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
	}

	// --- end occupancy ---
	let endingId = $state('');
	let endDate = $state('');
	let endError = $state('');

	const endOccMutation = createMutation(() => ({
		mutationFn: (id: string) => endOccupancy(id, endDate),
		onSuccess: async () => {
			endingId = '';
			endDate = '';
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: qk.units.occupancies(unitId) }),
				queryClient.invalidateQueries({ queryKey: qk.units.history(unitId) })
			]);
		}
	}));

	async function confirmEnd(id: string) {
		endError = '';
		if (!endDate) {
			endError = fa.required;
			return;
		}
		try {
			await endOccMutation.mutateAsync(id);
		} catch (error) {
			endError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
	}

	// --- occupant count ---
	let countValue = $state('');
	let countDate = $state('');
	let countError = $state('');

	const countMutation = createMutation(() => ({
		mutationFn: () =>
			recordOccupantCount(unitId, {
				occupant_count: Number(latinDigitsOnly(countValue) || '0'),
				effective_from: countDate
			}),
		onSuccess: async () => {
			countValue = '';
			countDate = '';
			await queryClient.invalidateQueries({ queryKey: qk.units.occupantCounts(unitId) });
		}
	}));

	async function submitCount(event: SubmitEvent) {
		event.preventDefault();
		countError = '';
		if (latinDigitsOnly(countValue) === '' || !countDate) {
			countError = fa.required;
			return;
		}
		try {
			await countMutation.mutateAsync();
		} catch (error) {
			countError = error instanceof ApiError ? error.message : fa.errorGeneric;
		}
	}

	// --- archive ---
	const archiveMutation = createMutation(() => ({
		mutationFn: () => archiveUnit(unitId),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['units'] });
			await goto('/m/units', { replace: true });
		}
	}));

	async function archive() {
		if (!confirm(fa.confirmArchiveUnit)) return;
		await archiveMutation.mutateAsync();
	}

	function dateLabel(iso: string | null | undefined): string {
		const date = fromIsoDate(iso);
		return date ? formatJalaliDate(date) : '—';
	}

	const historyLabels: Record<string, string> = {
		'unit.create': fa.historyCreate,
		'unit.update': fa.historyUpdate,
		'unit.delete': fa.historyDelete
	};
</script>

<div class="mx-auto max-w-5xl space-y-6">
	<a
		href="/m/units"
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
		{fa.backToUnitsList}
	</a>

	{#if unitQuery.isPending}
		<div class="h-72 animate-pulse rounded-3xl bg-white"></div>
	{:else if unitQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{unitQuery.error instanceof ApiError ? unitQuery.error.message : fa.errorGeneric}
			</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void unitQuery.refetch()}
				>{fa.retry}</button
			>
		</section>
	{:else if unit}
		<header class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<p class="text-sm font-semibold text-primary">{fa.unitDetail}</p>
				<h1 class="mt-1 flex items-center gap-3 text-2xl font-bold tracking-tight sm:text-3xl">
					<span>{fa.unit} {toPersianDigits(unit.number)}</span>
					<StatusChip tone={unitStatusTone(unit.status)} label={unitStatusLabel(unit.status)} />
				</h1>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<a class="btn-secondary min-h-10" href={`/m/units/${unitId}/edit`}>{fa.edit}</a>
				<button
					class="btn-secondary min-h-10 text-danger"
					type="button"
					disabled={archiveMutation.isPending}
					onclick={() => void archive()}
				>
					{fa.archiveUnit}
				</button>
			</div>
		</header>

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.unitInfo}</h2>
			<dl class="mt-4 grid gap-4 sm:grid-cols-3">
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.block}</dt>
					<dd class="mt-1 font-semibold">{unit.block ?? '—'}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.floor}</dt>
					<dd class="mt-1 font-semibold">{toPersianDigits(String(unit.floor))}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.area}</dt>
					<dd class="mt-1 font-semibold">{toPersianDigits(String(unit.area_m2))}</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.parkingCount}</dt>
					<dd class="mt-1 font-semibold">
						{toPersianDigits(String(unit.parking_count))}
						{#if unit.parking_numbers}<span class="text-on-surface-variant">
								({unit.parking_numbers})</span
							>{/if}
					</dd>
				</div>
				<div>
					<dt class="text-xs text-on-surface-variant">{fa.storageCount}</dt>
					<dd class="mt-1 font-semibold">
						{toPersianDigits(String(unit.storage_count))}
						{#if unit.storage_numbers}<span class="text-on-surface-variant">
								({unit.storage_numbers})</span
							>{/if}
					</dd>
				</div>
				<div class="sm:col-span-3">
					<dt class="text-xs text-on-surface-variant">{fa.description}</dt>
					<dd class="mt-1 whitespace-pre-wrap">{unit.notes || '—'}</dd>
				</div>
			</dl>
		</section>

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.occupancyHistoryLabel}</h2>

			{#if activeOccupancies.length}
				<ul class="mt-4 space-y-3">
					{#each activeOccupancies as occ (occ.id)}
						<li class="rounded-xl border border-success/20 bg-success-soft/50 p-4">
							<div class="flex flex-wrap items-center justify-between gap-3">
								<div>
									<p class="font-semibold">
										{occ.person?.full_name ?? '—'}
										<span class="text-sm font-normal text-on-surface-variant"
											>· {relationshipLabel(occ.relationship)}</span
										>
									</p>
									<p class="mt-1 text-xs text-on-surface-variant">
										{fa.startDate}: {dateLabel(occ.start_date)}
									</p>
								</div>
								{#if endingId === occ.id}
									<div class="flex flex-wrap items-end gap-2">
										<div class="w-40">
											<JalaliDateInput
												id={`end-${occ.id}`}
												label={fa.endDate}
												bind:value={endDate}
												required
											/>
										</div>
										<button
											class="btn-primary min-h-10"
											type="button"
											disabled={endOccMutation.isPending}
											onclick={() => void confirmEnd(occ.id)}
										>
											{fa.confirm}
										</button>
										<button
											class="btn-secondary min-h-10"
											type="button"
											onclick={() => {
												endingId = '';
												endDate = '';
												endError = '';
											}}>{fa.cancel}</button
										>
									</div>
								{:else}
									<button
										class="btn-secondary min-h-9 text-xs"
										type="button"
										onclick={() => {
											endingId = occ.id;
											endDate = '';
											endError = '';
										}}
									>
										{fa.endOccupancyTitle}
									</button>
								{/if}
							</div>
							{#if endingId === occ.id && endError}
								<p class="mt-2 text-xs text-danger" role="alert">{endError}</p>
							{/if}
						</li>
					{/each}
				</ul>
			{:else}
				<p class="mt-3 text-sm text-on-surface-variant">{fa.noPeople}</p>
			{/if}

			<form
				class="mt-6 border-t border-outline-variant/50 pt-5"
				onsubmit={submitOccupancy}
				novalidate
			>
				<h3 class="text-sm font-bold">{fa.addOccupancy}</h3>
				{#if !persons.length}
					<p class="mt-2 text-sm text-on-surface-variant">{fa.noPersonsForOccupancy}</p>
				{:else}
					<div class="mt-3 grid gap-4 sm:grid-cols-3">
						<div class="space-y-2">
							<label class="block text-sm font-semibold" for="occ-person">{fa.person}</label>
							<select id="occ-person" class="input-base bg-white" bind:value={personId}>
								<option value="">{fa.selectPerson}</option>
								{#each persons as person (person.id)}
									<option value={person.id}>{person.full_name}</option>
								{/each}
							</select>
						</div>
						<div class="space-y-2">
							<label class="block text-sm font-semibold" for="occ-relationship"
								>{fa.relationship}</label
							>
							<select id="occ-relationship" class="input-base bg-white" bind:value={relationship}>
								<option value="owner">{fa.relationshipOwner}</option>
								<option value="tenant">{fa.relationshipTenant}</option>
								<option value="non_resident_owner">{fa.relationshipNonResidentOwner}</option>
							</select>
						</div>
						<JalaliDateInput id="occ-start" label={fa.startDate} bind:value={startDate} required />
					</div>
					{#if addError}
						<p class="mt-3 text-sm text-danger" role="alert">{addError}</p>
					{/if}
					<button
						class="mt-4 btn-primary min-h-10"
						type="submit"
						disabled={addOccMutation.isPending}
					>
						{addOccMutation.isPending ? fa.saving : fa.add}
					</button>
				{/if}
			</form>

			{#if occupancies.length > activeOccupancies.length}
				<h3 class="mt-6 border-t border-outline-variant/50 pt-5 text-sm font-bold">
					{fa.formerResident}
				</h3>
				<ul class="mt-3 space-y-2">
					{#each occupancies.filter((o) => !o.is_active) as occ (occ.id)}
						<li
							class="flex flex-wrap items-center justify-between gap-2 rounded-xl bg-surface/70 px-4 py-2 text-sm"
						>
							<span>{occ.person?.full_name ?? '—'} · {relationshipLabel(occ.relationship)}</span>
							<span class="text-xs text-on-surface-variant">
								{dateLabel(occ.start_date)} — {dateLabel(occ.end_date)}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.occupantCountHistory}</h2>
			{#if countsQuery.isPending}
				<div class="mt-4 h-16 animate-pulse rounded-xl bg-surface-high"></div>
			{:else if (countsQuery.data?.items.length ?? 0) === 0}
				<p class="mt-3 text-sm text-on-surface-variant">{fa.noOccupantCounts}</p>
			{:else}
				<ul class="mt-4 divide-y divide-outline-variant/40">
					{#each countsQuery.data?.items ?? [] as row (row.id)}
						<li class="flex items-center justify-between py-2 text-sm">
							<span class="font-semibold"
								>{toPersianDigits(String(row.occupant_count))} {fa.person}</span
							>
							<span class="text-xs text-on-surface-variant"
								>{fa.effectiveFrom}: {dateLabel(row.effective_from)}</span
							>
						</li>
					{/each}
				</ul>
			{/if}

			<form class="mt-5 border-t border-outline-variant/50 pt-5" onsubmit={submitCount} novalidate>
				<h3 class="text-sm font-bold">{fa.recordCountTitle}</h3>
				<div class="mt-3 grid gap-4 sm:grid-cols-3">
					<div class="space-y-2">
						<label class="block text-sm font-semibold" for="count-value">{fa.occupantCount}</label>
						<input
							id="count-value"
							class="input-base bg-white"
							bind:value={countValue}
							inputmode="numeric"
							dir="ltr"
						/>
					</div>
					<JalaliDateInput
						id="count-date"
						label={fa.effectiveFrom}
						bind:value={countDate}
						required
					/>
				</div>
				{#if countError}
					<p class="mt-3 text-sm text-danger" role="alert">{countError}</p>
				{/if}
				<button class="mt-4 btn-primary min-h-10" type="submit" disabled={countMutation.isPending}>
					{countMutation.isPending ? fa.saving : fa.save}
				</button>
			</form>
		</section>

		<section class="rounded-2xl border border-outline-variant/60 bg-white p-5 sm:p-6">
			<h2 class="text-lg font-bold">{fa.unitHistory}</h2>
			{#if historyQuery.isPending}
				<div class="mt-4 h-16 animate-pulse rounded-xl bg-surface-high"></div>
			{:else if (historyQuery.data?.items.length ?? 0) === 0}
				<p class="mt-3 text-sm text-on-surface-variant">{fa.noHistory}</p>
			{:else}
				<ul class="mt-4 divide-y divide-outline-variant/40">
					{#each historyQuery.data?.items ?? [] as entry (entry.id)}
						<li class="flex flex-wrap items-center justify-between gap-2 py-3 text-sm">
							<span class="font-semibold">{historyLabels[entry.action] ?? entry.action}</span>
							<span class="text-xs text-on-surface-variant">
								{fromIsoDate(entry.created_at)
									? formatJalaliDateTime(fromIsoDate(entry.created_at) as Date)
									: '—'}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>
