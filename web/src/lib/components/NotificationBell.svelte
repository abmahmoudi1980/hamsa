<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { listNotifications } from '#lib/api/endpoints/notifications';
	import { toPersianDigits } from '#lib/format/digits';
	import { qk } from '#lib/query/keys';
	import { fa } from '#i18n/fa';

	let { href }: { href: string } = $props();
	const unreadQuery = createQuery(() => ({
		queryKey: qk.notifications.unreadCount(),
		queryFn: async () =>
			(await listNotifications({ unreadOnly: true, page: 1, page_size: 1 })).total
	}));
	const count = $derived(unreadQuery.data ?? 0);
</script>

<a
	class="relative icon-button"
	{href}
	aria-label={`${fa.navNotifications}: ${toPersianDigits(String(count))}`}
>
	<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
		<path
			d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9ZM10 21h4"
			stroke="currentColor"
			stroke-width="1.7"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	</svg>
	{#if count > 0}<span
			class="absolute -end-1 -top-1 grid min-h-5 min-w-5 place-items-center rounded-full bg-danger px-1 text-[10px] leading-none font-bold text-white"
			>{toPersianDigits(String(count))}</span
		>{/if}
</a>
