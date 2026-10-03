<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { archivePerson, listPersons } from '#lib/api/endpoints/persons';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const queryClient = useQueryClient();
	const PAGE_SIZE = 20;

	const buildingId = $derived(buildingSelection.id ?? '');

	let q = $state('');
	let page = $state(1);
	const filterKey = $derived(JSON.stringify({ q, page }));

	const personsQuery = createQuery(() => ({
		queryKey: qk.persons.list(buildingId, filterKey),
		queryFn: () => listPersons(buildingId, { q, page, page_size: PAGE_SIZE }),
		enabled: !!buildingId
	}));

	const persons = $derived(personsQuery.data?.items ?? []);
	const total = $derived(personsQuery.data?.total ?? 0);

	const archiveMutation = createMutation(() => ({
		mutationFn: (id: string) => archivePerson(id),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['persons'] });
		}
	}));

	async function archive(id: string) {
		if (!confirm(fa.confirmArchivePerson)) return;
		await archiveMutation.mutateAsync(id);
	}
</script>

<div class="space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.navPeople}</h1>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.filterPeople}</p>
		</div>
		{#if buildingId}
			<a href="/m/persons/new" class="btn-primary min-h-11 gap-2">
				<span class="text-xl leading-none" aria-hidden="true">＋</span>
				{fa.newPerson}
			</a>
		{/if}
	</header>

	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.noBuildingsHint}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else}
		<form
			class="rounded-2xl border border-outline-variant/60 bg-white p-4"
			onsubmit={(event) => event.preventDefault()}
		>
			<div class="max-w-sm space-y-1.5">
				<label class="block text-xs font-semibold" for="person-q">{fa.search}</label>
				<input
					id="person-q"
					class="input-base bg-white"
					bind:value={q}
					oninput={() => (page = 1)}
					placeholder={fa.searchPlaceholder}
				/>
			</div>
		</form>

		<section class="overflow-hidden rounded-2xl border border-outline-variant/60 bg-white">
			{#if personsQuery.isPending}
				<div class="space-y-2 p-5">
					{#each [1, 2, 3, 4, 5] as row (row)}
						<div class="h-12 animate-pulse rounded-xl bg-surface-high"></div>
					{/each}
				</div>
			{:else if personsQuery.isError}
				<div class="p-8 text-center">
					<p class="text-lg font-bold">{fa.errorGeneric}</p>
					<p class="mt-2 text-sm text-on-surface-variant">
						{personsQuery.error instanceof ApiError ? personsQuery.error.message : fa.errorGeneric}
					</p>
					<button class="mt-5 btn-primary" type="button" onclick={() => void personsQuery.refetch()}
						>{fa.retry}</button
					>
				</div>
			{:else if persons.length}
				<div
					class="flex items-center gap-2 border-b border-outline-variant/50 px-5 py-3 text-sm text-on-surface-variant"
				>
					<span class="rounded-lg bg-primary/10 px-2 py-0.5 font-bold text-primary"
						>{toPersianDigits(String(total))}</span
					>
					{fa.resultsCount}
				</div>
				<div class="overflow-x-auto">
					<table class="w-full min-w-[36rem] text-sm">
						<thead class="bg-surface/70 text-xs text-on-surface-variant">
							<tr>
								<th class="px-5 py-3 text-start font-semibold">{fa.fullName}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.phone}</th>
								<th class="px-5 py-3 text-start font-semibold">{fa.nationalId}</th>
								<th class="px-5 py-3 text-start font-semibold"
									><span class="sr-only">{fa.edit}</span></th
								>
							</tr>
						</thead>
						<tbody>
							{#each persons as person (person.id)}
								<tr class="border-t border-outline-variant/40 hover:bg-surface/60">
									<td class="px-5 py-3 font-semibold">{person.full_name}</td>
									<td class="px-5 py-3" dir="ltr">{person.phone ?? '—'}</td>
									<td class="px-5 py-3" dir="ltr">
										{person.national_id ? toPersianDigits(person.national_id) : '—'}
									</td>
									<td class="px-5 py-3">
										<div class="flex items-center justify-end gap-3">
											<a
												class="text-xs font-semibold text-primary hover:underline"
												href={`/m/persons/${person.id}/edit`}>{fa.edit}</a
											>
											<button
												class="text-xs font-semibold text-danger hover:underline"
												type="button"
												onclick={() => void archive(person.id)}>{fa.archive}</button
											>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<Pagination {page} pageSize={PAGE_SIZE} {total} onChange={(next) => (page = next)} />
			{:else}
				<div class="p-10 text-center">
					<p class="text-lg font-bold">{fa.noPeople}</p>
					<p class="mt-2 text-sm text-on-surface-variant">{fa.noPeopleHint}</p>
					<a class="mt-5 btn-primary" href="/m/persons/new">{fa.newPerson}</a>
				</div>
			{/if}
		</section>
	{/if}
</div>
