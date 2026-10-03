<script lang="ts">
	import { goto } from '$app/navigation';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError } from '#lib/api/apiError';
	import {
		createPerson,
		updatePerson,
		type Person,
		type PersonInput
	} from '#lib/api/endpoints/persons';
	import { fa } from '#i18n/fa';

	interface Props {
		buildingId: string;
		/** Present in edit mode. */
		personId?: string;
		initial?: Person;
	}

	let { buildingId, personId, initial }: Props = $props();

	const queryClient = useQueryClient();

	let fullName = $state('');
	let phone = $state('');
	let nationalId = $state('');
	let nameError = $state('');
	let phoneError = $state('');
	let nationalIdError = $state('');
	let formError = $state('');

	let seeded = false;
	$effect(() => {
		if (initial && !seeded) {
			fullName = initial.full_name;
			phone = initial.phone ?? '';
			nationalId = initial.national_id ?? '';
			seeded = true;
		}
	});

	const mutation = createMutation(() => ({
		mutationFn: (input: PersonInput) =>
			personId ? updatePerson(personId, input) : createPerson(buildingId, input),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['persons'] });
			await goto('/m/persons', { replaceState: true });
		}
	}));

	/** Routes a server error to the field it named, else the form banner. */
	function applyApiError(error: ApiError, fields: Record<string, (message: string) => void>) {
		const detail = error.details.find((d) => d.field && fields[d.field]);
		if (detail?.field) fields[detail.field]?.(error.message);
		else formError = error.message;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		nameError = '';
		phoneError = '';
		nationalIdError = '';
		formError = '';

		if (fullName.trim() === '') {
			nameError = fa.required;
			return;
		}

		const input: PersonInput = {
			full_name: fullName.trim(),
			phone: phone || undefined,
			national_id: nationalId || undefined
		};

		try {
			await mutation.mutateAsync(input);
		} catch (error) {
			if (error instanceof ApiError) {
				applyApiError(error, {
					full_name: (message) => (nameError = message),
					phone: (message) => (phoneError = message),
					national_id: (message) => (nationalIdError = message)
				});
			} else {
				formError = fa.errorGeneric;
			}
		}
	}
</script>

<form
	class="overflow-hidden rounded-3xl border border-outline-variant/60 bg-white shadow-sm"
	onsubmit={submit}
	novalidate
>
	<div class="grid gap-5 p-5 sm:p-8">
		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="person-name"
				>{fa.fullName}<span class="text-danger" aria-hidden="true">*</span></label
			>
			<input
				id="person-name"
				class={`input-base bg-white ${nameError ? 'border-danger' : ''}`}
				bind:value={fullName}
				placeholder={fa.personNamePlaceholder}
				aria-invalid={nameError ? 'true' : undefined}
			/>
			{#if nameError}<p class="text-xs text-danger" aria-live="polite">{nameError}</p>{/if}
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="person-phone">{fa.phone}</label>
			<input
				id="person-phone"
				class={`input-base bg-white ${phoneError ? 'border-danger' : ''}`}
				bind:value={phone}
				inputmode="tel"
				dir="ltr"
				placeholder={fa.phonePlaceholder}
			/>
			{#if phoneError}<p class="text-xs text-danger" aria-live="polite">{phoneError}</p>{/if}
		</div>

		<div class="space-y-2">
			<label class="block text-sm font-semibold" for="person-national-id">{fa.nationalId}</label>
			<input
				id="person-national-id"
				class={`input-base bg-white ${nationalIdError ? 'border-danger' : ''}`}
				bind:value={nationalId}
				inputmode="numeric"
				dir="ltr"
			/>
			{#if nationalIdError}<p class="text-xs text-danger" aria-live="polite">
					{nationalIdError}
				</p>{/if}
		</div>

		{#if formError}
			<p
				class="rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
				role="alert"
			>
				{formError}
			</p>
		{/if}
	</div>

	<div
		class="flex flex-col-reverse gap-3 border-t border-outline-variant/50 bg-surface/70 p-5 sm:flex-row sm:justify-end sm:px-8"
	>
		<a class="btn-secondary" href="/m/persons">{fa.cancel}</a>
		<button class="btn-primary" type="submit" disabled={mutation.isPending}>
			{mutation.isPending ? fa.saving : fa.save}
		</button>
	</div>
</form>
