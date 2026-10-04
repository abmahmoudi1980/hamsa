<script lang="ts">
	import type { MaintenanceRequest } from '#lib/api/endpoints/maintenance';
	import { formatJalaliDateTime } from '#lib/format/jalali';
	import { maintenanceStatusLabel } from '#lib/format/labels';
	import { fa } from '#i18n/fa';

	let { request }: { request: MaintenanceRequest } = $props();

	function dateTime(value: string | undefined): string {
		if (!value) return '—';
		const date = new Date(value);
		return Number.isFinite(date.getTime()) ? formatJalaliDateTime(date) : '—';
	}
</script>

<section class="card p-5 sm:p-6">
	<h2 class="text-lg font-bold">{fa.maintenanceStatusPath}</h2>
	<ol class="mt-4 space-y-3 border-e-2 border-outline-variant/60 pe-4">
		<li class="relative">
			<span class="absolute -end-[1.35rem] top-1 size-3 rounded-full bg-primary ring-4 ring-white"
			></span>
			<p class="text-sm font-semibold">{fa.maintenanceRequest}</p>
			<p class="mt-1 text-xs text-on-surface-variant">
				{fa.createdAt} · {dateTime(request.created_at)}
			</p>
		</li>
		<li class="relative">
			<span
				class={`absolute -end-[1.35rem] top-1 size-3 rounded-full ring-4 ring-white ${request.updated_at !== request.created_at ? 'bg-primary' : 'bg-outline-variant'}`}
			></span>
			<p class="text-sm font-semibold">
				{fa.maintenanceCurrentStatus}: {maintenanceStatusLabel(request.status)}
			</p>
			<p class="mt-1 text-xs text-on-surface-variant">
				{fa.lastUpdatedAt} · {dateTime(request.updated_at)}
			</p>
		</li>
		{#if request.closed_at}
			<li class="relative">
				<span
					class="absolute -end-[1.35rem] top-1 size-3 rounded-full bg-outline-variant ring-4 ring-white"
				></span>
				<p class="text-sm font-semibold">{fa.maintenanceClosed}</p>
				<p class="mt-1 text-xs text-on-surface-variant">
					{fa.closedAt} · {dateTime(request.closed_at)}
				</p>
			</li>
		{/if}
	</ol>
</section>
