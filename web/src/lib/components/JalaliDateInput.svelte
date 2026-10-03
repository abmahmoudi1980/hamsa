<script lang="ts">
	/**
	 * Jalali date input (R6: the wire is Gregorian ISO; users only ever see and
	 * type Solar Hijri). The bound `value` is always `YYYY-MM-DD` or ''.
	 */
	import { formatJalaliInput, parseJalaliInput } from '#lib/format/dateInput';
	import { fa } from '#i18n/fa';

	interface Props {
		value?: string;
		id: string;
		label: string;
		required?: boolean;
		disabled?: boolean;
		/** Externally supplied message (e.g. from the server); shown verbatim. */
		error?: string;
	}

	let {
		value = $bindable(''),
		id,
		label,
		required = false,
		disabled = false,
		error = ''
	}: Props = $props();

	/**
	 * The raw typed text wins while the user edits; otherwise the field mirrors
	 * the bound value. A draft rather than a syncing `$effect`, so an external
	 * reset is reflected without clobbering an in-progress edit.
	 */
	let draft = $state<string | null>(null);
	let localError = $state('');

	const text = $derived(draft ?? formatJalaliInput(value));

	function onInput(event: Event) {
		draft = (event.currentTarget as HTMLInputElement).value;
		localError = '';
		const iso = parseJalaliInput(draft);
		if (iso !== null && iso !== value) value = iso;
	}

	function onBlur() {
		const current = draft ?? '';
		if (current.trim() === '') {
			value = '';
			draft = null;
			localError = '';
			return;
		}
		if (parseJalaliInput(current) === null) {
			localError = fa.invalidDate;
			return;
		}
		// Valid: drop the draft so the field shows the canonical value.
		draft = null;
	}

	const message = $derived(error || localError);
</script>

<div class="space-y-2">
	<label class="block text-sm font-semibold" for={id}>
		{label}
		{#if required}<span class="text-danger" aria-hidden="true">*</span>{/if}
	</label>
	<input
		{id}
		class={`input-base bg-white ${message ? 'border-danger' : ''}`}
		value={text}
		oninput={onInput}
		onblur={onBlur}
		{disabled}
		dir="ltr"
		inputmode="numeric"
		maxlength="10"
		placeholder={fa.datePlaceholder}
		aria-invalid={message ? 'true' : undefined}
		aria-describedby={message ? `${id}-error` : undefined}
	/>
	{#if message}
		<p id={`${id}-error`} class="text-xs text-danger" aria-live="polite">{message}</p>
	{/if}
</div>
