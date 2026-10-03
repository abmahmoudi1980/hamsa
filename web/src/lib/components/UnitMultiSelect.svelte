<script lang="ts">
	/**
	 * Searchable multi-select over a building's units, used by the
	 * `specific_units` cost method. Filters server-side (`?q=&block=&floor=`) so
	 * the picker stays fast on a building with hundreds of units.
	 */
	import { createQuery } from '@tanstack/svelte-query';
	import { listUnits } from '#lib/api/endpoints/units';
	import { toPersianDigits } from '#lib/format/digits';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	interface Props {
		buildingId: string;
		selected: string[];
		onChange: (ids: string[]) => void;
		disabled?: boolean;
	}

	let { buildingId, selected, onChange, disabled = false }: Props = $props();

	let q = $state('');
	let block = $state('');
	let floor = $state('');

	const filterKey = $derived(JSON.stringify({ q, block, floor }));

	const unitsQuery = createQuery(() => ({
		queryKey: qk.units.list(buildingId, filterKey),
		queryFn: () => listUnits(buildingId, { q, block, floor, page_size: 100 }),
		enabled: !!buildingId
	}));

	const units = $derived(unitsQuery.data?.items ?? []);

	function toggle(id: string) {
		onChange(selected.includes(id) ? selected.filter((value) => value !== id) : [...selected, id]);
	}
</script>

<div class="space-y-3 rounded-2xl border border-outline-variant/60 bg-surface/50 p-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<p class="text-sm font-semibold">{fa.specificUnits}</p>
		<span class="rounded-full bg-primary/10 px-3 py-1 text-xs font-bold text-primary">
			{toPersianDigits(String(selected.length))}
			{fa.selectUnitsCount}
		</span>
	</div>

	<div class="grid gap-2 sm:grid-cols-3">
		<input
			class="input-base bg-white"
			bind:value={q}
			placeholder={fa.searchPlaceholder}
			aria-label={fa.search}
		/>
		<input
			class="input-base bg-white"
			bind:value={block}
			placeholder={fa.blockPlaceholder}
			aria-label={fa.block}
		/>
		<input
			class="input-base bg-white"
			bind:value={floor}
			placeholder={fa.floor}
			inputmode="numeric"
			dir="ltr"
			aria-label={fa.floor}
		/>
	</div>

	{#if unitsQuery.isPending}
		<div class="h-24 animate-pulse rounded-xl bg-surface-high"></div>
	{:else if units.length === 0}
		<p class="text-sm text-on-surface-variant">{fa.noUnits}</p>
	{:else}
		<ul class="max-h-64 space-y-1 overflow-y-auto pe-1">
			{#each units as unit (unit.id)}
				{@const checked = selected.includes(unit.id)}
				<li>
					<label
						class={`flex cursor-pointer items-center gap-3 rounded-xl px-3 py-2 text-sm ${
							checked ? 'bg-primary/10 font-semibold text-primary' : 'hover:bg-surface-high'
						}`}
					>
						<input
							type="checkbox"
							class="size-4 accent-primary"
							{checked}
							{disabled}
							onchange={() => toggle(unit.id)}
						/>
						<span class="min-w-0 flex-1 truncate">
							{fa.unit}
							{toPersianDigits(unit.number)}
							{#if unit.block}<span class="text-on-surface-variant">
									· {fa.block} {unit.block}</span
								>{/if}
							<span class="text-on-surface-variant">
								· {fa.floor} {toPersianDigits(String(unit.floor))}</span
							>
						</span>
					</label>
				</li>
			{/each}
		</ul>
	{/if}
</div>
