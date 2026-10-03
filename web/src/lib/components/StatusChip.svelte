<script lang="ts">
	import type { Tone } from '#lib/format/labels';

	interface Props {
		tone: Tone;
		label: string;
	}

	let { tone, label }: Props = $props();

	const toneClass: Record<Tone, string> = {
		neutral: 'bg-surface-high text-on-surface-variant',
		success: 'bg-success-soft text-success',
		warning: 'bg-warning-soft text-warning',
		danger: 'bg-danger-soft text-danger',
		info: 'bg-info-soft text-info'
	};
</script>

<span
	class="inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold {toneClass[tone]}"
	>{label}</span
>
