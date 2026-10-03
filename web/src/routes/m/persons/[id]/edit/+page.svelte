<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import { listPersons } from '#lib/api/endpoints/persons';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import PersonForm from '#lib/components/PersonForm.svelte';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	const personId = $derived(page.params.id ?? '');
	const buildingId = $derived(buildingSelection.id ?? '');

	// No single-person GET exists on the contract, so resolve the row from the
	// building's first page of persons.
	const personsQuery = createQuery(() => ({
		queryKey: qk.persons.list(buildingId, 'edit'),
		queryFn: () => listPersons(buildingId, { page_size: 100 }),
		enabled: !!buildingId
	}));

	const person = $derived(personsQuery.data?.items.find((p) => p.id === personId));
</script>

<div class="mx-auto max-w-2xl space-y-6">
	<a
		href="/m/persons"
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
		{fa.backToPeopleList}
	</a>

	<header>
		<p class="text-sm font-semibold text-primary">{fa.navPeople}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.editPerson}</h1>
	</header>

	{#if personsQuery.isPending}
		<div class="h-64 animate-pulse rounded-3xl bg-white"></div>
	{:else if personsQuery.isError}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
			<p class="mt-2 text-sm text-on-surface-variant">
				{personsQuery.error instanceof ApiError ? personsQuery.error.message : fa.errorGeneric}
			</p>
		</section>
	{:else if person}
		<PersonForm {buildingId} {personId} initial={person} />
	{:else}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.errorGeneric}</p>
		</section>
	{/if}
</div>
