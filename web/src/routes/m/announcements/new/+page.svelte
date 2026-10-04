<script lang="ts">
	import { goto } from '$app/navigation';
	import { buildingSelection } from '#lib/building/selection.svelte';
	import AnnouncementForm from '#lib/components/AnnouncementForm.svelte';
	import { fa } from '#i18n/fa';

	const buildingId = $derived(buildingSelection.id ?? '');
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<a
		class="text-sm font-semibold text-on-surface-variant hover:text-primary"
		href="/m/announcements">{fa.navAnnouncements}</a
	>
	<header>
		<p class="text-sm font-semibold text-primary">{fa.managerConsole}</p>
		<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{fa.newAnnouncement}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">{fa.announcementRequestDescription}</p>
	</header>
	{#if !buildingId}
		<section class="card p-8 text-center">
			<p class="text-lg font-bold">{fa.noBuildings}</p>
			<a class="mt-5 btn-primary" href="/m/buildings">{fa.createFirstBuilding}</a>
		</section>
	{:else}
		<AnnouncementForm
			{buildingId}
			onDone={() => void goto('/m/announcements', { replace: true })}
			onCancel={() => void goto('/m/announcements')}
		/>
	{/if}
</div>
