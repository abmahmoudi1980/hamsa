<script lang="ts">
	import { fa } from '#i18n/fa';
	import { toPersianDigits } from '#lib/format/digits';

	interface Props {
		page: number;
		pageSize: number;
		total: number;
		onChange: (page: number) => void;
	}

	let { page, pageSize, total, onChange }: Props = $props();

	const pageCount = $derived(Math.max(1, Math.ceil(total / pageSize)));
	const from = $derived(total === 0 ? 0 : (page - 1) * pageSize + 1);
	const to = $derived(Math.min(page * pageSize, total));
</script>

{#if pageCount > 1}
	<nav
		class="flex flex-wrap items-center justify-between gap-3 border-t border-outline-variant/50 px-5 py-4 text-sm sm:px-6"
		aria-label={fa.pageLabel}
	>
		<p class="text-on-surface-variant">
			{fa.showingRange}
			{toPersianDigits(String(from))}–{toPersianDigits(String(to))}
			{fa.pageLabel}
			{toPersianDigits(String(page))} / {toPersianDigits(String(pageCount))}
		</p>
		<div class="flex items-center gap-2">
			<button
				class="btn-secondary min-h-9 px-4 text-xs"
				type="button"
				disabled={page <= 1}
				onclick={() => onChange(page - 1)}
			>
				{fa.previous}
			</button>
			<button
				class="btn-secondary min-h-9 px-4 text-xs"
				type="button"
				disabled={page >= pageCount}
				onclick={() => onChange(page + 1)}
			>
				{fa.next}
			</button>
		</div>
	</nav>
{/if}
