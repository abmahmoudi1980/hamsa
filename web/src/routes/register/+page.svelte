<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth, restoreSession, setAuthPending, setAuthenticated } from '#lib/auth/auth.svelte';
	import { ApiError } from '#lib/api/apiError';
	import { register } from '#lib/api/endpoints/auth';
	import { isManagerRole } from '#lib/api/tokenStore';
	import { fa } from '#i18n/fa';
	import {
		isValidInviteCode,
		isValidPassword,
		isValidPhone,
		normalizePhone
	} from '#lib/validation/validators';

	let name = $state('');
	let phone = $state('');
	let code = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');

	onMount(async () => {
		const user = auth.status === 'authenticated' ? auth.user : await restoreSession();
		if (user) await goto(isManagerRole(user.role) ? '/m' : '/r', { replaceState: true });
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		const canonical = normalizePhone(phone);
		if (!canonical || !isValidPhone(phone)) {
			error = fa.invalidPhone;
			return;
		}
		if (!isValidInviteCode(code)) {
			error = fa.invalidInviteCode;
			return;
		}
		if (!isValidPassword(password)) {
			error = fa.invalidPassword;
			return;
		}
		if (password !== confirmPassword) {
			error = fa.passwordMismatch;
			return;
		}

		setAuthPending(true);
		try {
			const user = await register({
				phone: canonical,
				code: code.trim(),
				password,
				name: name.trim() || undefined
			});
			setAuthenticated(user);
			await goto(isManagerRole(user.role) ? '/m' : '/r', { replaceState: true });
		} catch (cause) {
			error = cause instanceof ApiError ? cause.message : fa.errorGeneric;
			setAuthPending(false);
		}
	}
</script>

<main class="grid min-h-dvh place-items-center bg-surface px-4 py-10">
	<section
		class="w-full max-w-lg rounded-3xl border border-outline-variant/60 bg-white p-6 shadow-[0_24px_70px_-36px_rgba(21,58,53,0.25)] sm:p-9"
	>
		<a
			href="/login"
			class="inline-flex items-center gap-2 text-sm font-semibold text-on-surface-variant hover:text-primary"
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
			{fa.login}
		</a>
		<div class="mt-7">
			<p class="text-sm font-semibold text-primary">{fa.appName}</p>
			<h1 class="mt-2 text-2xl font-bold sm:text-3xl">{fa.registerTitle}</h1>
			<p class="mt-2 text-sm leading-6 text-on-surface-variant">{fa.registerSubtitle}</p>
		</div>

		{#if error}
			<p
				class="mt-6 rounded-xl border border-danger/20 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
				role="alert"
			>
				{error}
			</p>
		{/if}

		<form class="mt-6 space-y-4" onsubmit={submit} novalidate>
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="register-name">{fa.name}</label>
				<input
					id="register-name"
					class="input-base bg-white"
					bind:value={name}
					autocomplete="name"
					maxlength="120"
				/>
			</div>
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="register-phone">{fa.phone}</label>
				<input
					id="register-phone"
					class="input-base bg-white"
					type="tel"
					inputmode="numeric"
					autocomplete="tel"
					dir="ltr"
					placeholder={fa.phonePlaceholder}
					bind:value={phone}
				/>
			</div>
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="invite-code">{fa.inviteCode}</label>
				<input
					id="invite-code"
					class="input-base bg-white"
					dir="ltr"
					autocomplete="one-time-code"
					bind:value={code}
				/>
			</div>
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="register-password">{fa.password}</label>
				<input
					id="register-password"
					class="input-base bg-white"
					type="password"
					autocomplete="new-password"
					bind:value={password}
				/>
			</div>
			<div class="space-y-2">
				<label class="block text-sm font-semibold" for="register-confirm"
					>{fa.confirmPassword}</label
				>
				<input
					id="register-confirm"
					class="input-base bg-white"
					type="password"
					autocomplete="new-password"
					bind:value={confirmPassword}
				/>
			</div>
			<button class="btn-primary w-full" type="submit" disabled={auth.pending}>
				{auth.pending ? fa.saving : fa.register}
			</button>
		</form>
	</section>
</main>
