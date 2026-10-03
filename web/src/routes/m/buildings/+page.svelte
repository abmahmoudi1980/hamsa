<script lang="ts">
	import { goto } from '$app/navigation';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listBuildings } from '#lib/api/endpoints/buildings';
	import { selectBuilding } from '#lib/building/selection.svelte';
	import { formatJalaliDate, fromIsoDate } from '#lib/format/jalali';
	import { fa } from '#i18n/fa';
	import { qk } from '#lib/query/keys';

	const buildingsQuery = createQuery(() => ({
		queryKey: qk.buildings.list(),
		queryFn: listBuildings
	}));

	function dateLabel(value: string): string {
		const date = fromIsoDate(value);
		return date ? formatJalaliDate(date) : '';
	}

	function openBuilding(id: string) {
		selectBuilding(id);
		void goto('/m');
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navBuildings}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.buildingRegistryDescription}</p>
		</div>
		<a href="/m/buildings/new" class="btn-primary min-h-11 gap-2">
			<span class="text-xl leading-none" aria-hidden="true">＋</span>
			{fa.newBuilding}
		</a>
	</header>

	{#if buildingsQuery.isPending}
		<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3" aria-label={fa.loading}>
			{#each [1, 2, 3] as item (item)}
				<div class="h-48 animate-pulse rounded-2xl bg-white"></div>
			{/each}
		</div>
	{:else if buildingsQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{buildingsQuery.error instanceof ApiError ? buildingsQuery.error.message : fa.errorGeneric}
			</p>
			<button class="mt-5 btn-primary" type="button" onclick={() => void buildingsQuery.refetch()}
				>{fa.retry}</button
			>
		</section>
	{:else if buildingsQuery.data?.length}
		<div class="flex items-center gap-2 text-sm text-on-surface-variant">
			<span class="grid size-8 place-items-center rounded-lg bg-primary/10 font-bold text-primary"
				>{buildingsQuery.data.length.toLocaleString('fa-IR')}</span
			>
			{fa.buildingsCount}
		</div>
		<section class="grid gap-4 md:grid-cols-2 xl:grid-cols-3" aria-label={fa.navBuildings}>
			{#each buildingsQuery.data as building (building.id)}
				<div
					class="group relative overflow-hidden rounded-2xl border border-outline-variant/60 bg-white shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/35 hover:shadow-md"
				>
					<button
						class="w-full p-5 text-start"
						type="button"
						onclick={() => openBuilding(building.id)}
					>
						<div class="flex items-start justify-between gap-3">
							<div
								class="grid size-12 shrink-0 place-items-center rounded-2xl bg-primary/10 text-primary"
							>
								<svg viewBox="0 0 24 24" fill="none" class="size-6" aria-hidden="true">
									<path
										d="M4 21V5l8-3v19m0-12h8v12M2 21h20M8 7h1m-1 4h1m-1 4h1m7-3h1m-1 4h1"
										stroke="currentColor"
										stroke-width="1.7"
										stroke-linecap="round"
										stroke-linejoin="round"
									/>
								</svg>
							</div>
							<span
								class="rounded-full bg-success-soft px-3 py-1 text-xs font-semibold text-success"
								>{fa.statusActive}</span
							>
						</div>
						<h2 class="mt-5 truncate text-lg font-bold">{building.name}</h2>
						<p class="mt-1 min-h-6 truncate text-sm text-on-surface-variant">
							{building.address || fa.addressNotSet}
						</p>
						<div class="mt-5 grid grid-cols-3 gap-2 border-t border-outline-variant/40 pt-4">
							<div>
								<p class="text-[11px] text-on-surface-variant">{fa.blockCount}</p>
								<p class="mt-1 font-bold">{building.block_count.toLocaleString('fa-IR')}</p>
							</div>
							<div>
								<p class="text-[11px] text-on-surface-variant">{fa.floorCount}</p>
								<p class="mt-1 font-bold">{building.floor_count.toLocaleString('fa-IR')}</p>
							</div>
							<div>
								<p class="text-[11px] text-on-surface-variant">{fa.unitCount}</p>
								<p class="mt-1 font-bold">{building.unit_count.toLocaleString('fa-IR')}</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-between text-xs text-on-surface-variant">
							<span>{building.manager_phone || fa.managerPhoneNotSet}</span>
							{#if building.created_at}
								<span>{dateLabel(building.created_at)}</span>
							{/if}
						</div>
					</button>
					<div
						class="flex items-center justify-end gap-4 border-t border-outline-variant/40 px-5 py-3 text-xs"
					>
						<a
							class="font-semibold text-primary hover:underline"
							href={`/m/buildings/${building.id}/edit`}>{fa.edit}</a
						>
						<a
							class="font-semibold text-primary hover:underline"
							href={`/m/buildings/${building.id}/managers`}>{fa.managers}</a
						>
					</div>
				</div>
			{/each}
		</section>
	{:else}
		<section
			class="relative overflow-hidden rounded-3xl border border-outline-variant/60 bg-white p-7 shadow-sm sm:p-10"
		>
			<div
				aria-hidden="true"
				class="absolute -end-12 -top-16 size-64 rounded-full border border-primary/5"
			></div>
			<div class="relative max-w-2xl">
				<span class="grid size-14 place-items-center rounded-2xl bg-primary/10 text-primary">
					<svg viewBox="0 0 24 24" fill="none" class="size-7" aria-hidden="true"
						><path
							d="M4 21V5l8-3v19m0-12h8v12M2 21h20M8 7h1m-1 4h1m-1 4h1m7-3h1m-1 4h1"
							stroke="currentColor"
							stroke-width="1.7"
							stroke-linecap="round"
							stroke-linejoin="round"
						/></svg
					>
				</span>
				<h2 class="mt-5 text-2xl font-bold">{fa.emptyBuildingsTitle}</h2>
				<p class="mt-2 max-w-xl text-sm leading-7 text-on-surface-variant">
					{fa.emptyBuildingsBody}
				</p>
				<a class="mt-6 btn-primary" href="/m/buildings/new">{fa.createFirstBuilding}</a>
			</div>
		</section>
	{/if}
</div>
