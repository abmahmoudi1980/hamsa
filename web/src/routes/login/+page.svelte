<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import {
		auth,
		restoreSession,
		setAuthError,
		setAuthPending,
		setAuthenticated
	} from '#lib/auth/auth.svelte';
	import { isManagerRole } from '#lib/api/tokenStore';
	import { ApiError } from '#lib/api/apiError';
	import { login } from '#lib/api/endpoints/auth';
	import { fa } from '#i18n/fa';
	import { isValidPassword, isValidPhone, normalizePhone } from '#lib/validation/validators';

	let phone = $state('');
	let password = $state('');
	let showPassword = $state(false);
	/** Set after a failed attempt, so the first render is not full of errors. */
	let submitted = $state(false);

	onMount(async () => {
		const user = auth.status === 'authenticated' ? auth.user : await restoreSession();
		if (user) await goto(isManagerRole(user.role) ? '/m' : '/r', { replace: true });
	});

	const phoneError = $derived.by(() => {
		if (!submitted) return null;
		if (!phone.trim()) return fa.required;
		if (!isValidPhone(phone)) return fa.invalidPhone;
		return null;
	});

	const passwordError = $derived.by(() => {
		if (!submitted) return null;
		if (!password) return fa.required;
		return isValidPassword(password) ? null : fa.invalidPassword;
	});

	const canSubmit = $derived(!auth.pending);

	async function onSubmit(event: SubmitEvent) {
		event.preventDefault();
		submitted = true;
		auth.error = null;

		const canonical = normalizePhone(phone);
		if (!canonical || !isValidPhone(phone) || !isValidPassword(password)) return;

		setAuthPending(true);
		try {
			const user = await login({ phone: canonical, password });
			setAuthenticated(user);
			await goto(isManagerRole(user.role) ? '/m' : '/r', { replace: true });
		} catch (err) {
			const message =
				err instanceof ApiError ? err.message : 'خطایی رخ داد. لطفاً دوباره تلاش کنید.';
			setAuthError(message);
		}
	}
</script>

<main class="grid min-h-dvh bg-surface lg:grid-cols-[1.04fr_0.96fr]">
	<!-- Brand panel: hidden on small screens, where it would only push the form
	     below the fold. -->
	<section
		class="relative hidden min-h-dvh flex-col justify-between overflow-hidden bg-primary px-12 py-12 text-on-primary lg:flex xl:px-16 xl:py-14"
	>
		<div aria-hidden="true" class="pointer-events-none absolute inset-0 overflow-hidden">
			<div
				class="absolute -start-36 -top-44 size-[28rem] rounded-full border border-white/10"
			></div>
			<div
				class="absolute -start-20 -top-28 size-[21rem] rounded-full border border-white/10"
			></div>
			<div class="absolute end-0 bottom-0 h-[22rem] w-[34rem] opacity-15">
				<svg viewBox="0 0 544 352" fill="none" xmlns="http://www.w3.org/2000/svg" class="size-full">
					<path
						d="M35 351V112L153 42L271 112V351M271 351V147L365 91L459 147V351M1 351H543"
						stroke="white"
						stroke-width="3"
					/>
					<path
						d="M74 146H113V187H74zM153 146H192V187H153zM74 222H113V263H74zM153 222H192V263H153zM310 174H342V208H310zM383 174H415V208H383zM310 238H342V272H310zM383 238H415V272H383z"
						stroke="white"
						stroke-width="2"
					/>
					<path d="M126 351V291H174V351M350 351V300H391V351" stroke="white" stroke-width="2" />
				</svg>
			</div>
		</div>

		<div class="relative z-10 flex items-center gap-3">
			<div class="grid size-12 place-items-center rounded-2xl bg-white/12 ring-1 ring-white/20">
				<svg viewBox="0 0 32 32" fill="none" class="size-7" aria-hidden="true">
					<path
						d="M5 27V13.5L16 5l11 8.5V27h-8v-8h-6v8H5Z"
						stroke="currentColor"
						stroke-width="2.2"
						stroke-linejoin="round"
					/>
					<path
						d="M2.5 14.5 16 4l13.5 10.5"
						stroke="currentColor"
						stroke-width="2.2"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
				</svg>
			</div>
			<div>
				<p class="text-2xl leading-tight font-bold">{fa.appName}</p>
				<p class="mt-1 text-sm text-white/75">{fa.appTagline}</p>
			</div>
		</div>

		<div class="relative z-10 w-full max-w-[36rem] pb-12">
			<p
				class="mb-5 inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/10 px-4 py-2 text-sm text-white/90"
			>
				<span class="size-2 rounded-full bg-emerald-200"></span>
				{fa.loginEyebrow}
			</p>
			<h2 class="max-w-lg text-4xl leading-[1.45] font-bold xl:text-5xl">{fa.loginHeroTitle}</h2>
			<p class="mt-5 max-w-lg text-base leading-8 text-white/80 xl:text-lg">
				{fa.loginHeroDescription}
			</p>

			<ul class="mt-9 grid gap-4 text-sm text-white/90 xl:text-base">
				<li class="flex items-center gap-3">
					<span class="grid size-8 shrink-0 place-items-center rounded-full bg-white/12">
						<svg viewBox="0 0 20 20" fill="none" class="size-4" aria-hidden="true"
							><path
								d="m4 10 4 4 8-8"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							/></svg
						>
					</span>
					{fa.loginFeatureBilling}
				</li>
				<li class="flex items-center gap-3">
					<span class="grid size-8 shrink-0 place-items-center rounded-full bg-white/12">
						<svg viewBox="0 0 20 20" fill="none" class="size-4" aria-hidden="true"
							><path
								d="M3 16V7l7-4 7 4v9M7 16v-5h6v5M2 16h16"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/></svg
						>
					</span>
					{fa.loginFeatureResidents}
				</li>
				<li class="flex items-center gap-3">
					<span class="grid size-8 shrink-0 place-items-center rounded-full bg-white/12">
						<svg viewBox="0 0 20 20" fill="none" class="size-4" aria-hidden="true"
							><path
								d="M10 3v2m0 10v2M3 10h2m10 0h2M5.05 5.05l1.42 1.42m7.06 7.06 1.42 1.42m0-9.9-1.42 1.42m-7.06 7.06-1.42 1.42M7 10a3 3 0 1 0 6 0 3 3 0 0 0-6 0Z"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
							/></svg
						>
					</span>
					{fa.loginFeatureSupport}
				</li>
			</ul>
		</div>

		<p class="relative z-10 text-xs text-white/60">{fa.appName} · {fa.loginSubtitle}</p>
	</section>

	<section class="flex min-h-dvh items-center justify-center px-5 py-10 sm:px-8 lg:px-12">
		<div class="w-full max-w-md">
			<div class="mb-7 flex items-center gap-3 lg:hidden">
				<div
					class="grid size-11 place-items-center rounded-2xl bg-primary text-on-primary shadow-sm"
				>
					<svg viewBox="0 0 32 32" fill="none" class="size-6" aria-hidden="true">
						<path
							d="M5 27V13.5L16 5l11 8.5V27h-8v-8h-6v8H5Z"
							stroke="currentColor"
							stroke-width="2.2"
							stroke-linejoin="round"
						/>
						<path
							d="M2.5 14.5 16 4l13.5 10.5"
							stroke="currentColor"
							stroke-width="2.2"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
				</div>
				<div>
					<p class="text-xl font-bold">{fa.appName}</p>
					<p class="text-xs text-on-surface-variant">{fa.appTagline}</p>
				</div>
			</div>

			<div
				class="rounded-3xl border border-outline-variant/70 bg-white p-6 shadow-[0_24px_70px_-36px_rgba(21,58,53,0.28)] sm:p-9"
			>
				<div class="mb-7">
					<p class="text-sm font-semibold text-primary">{fa.appName}</p>
					<h1 class="mt-2 text-2xl font-bold tracking-tight sm:text-3xl">{fa.loginTitle}</h1>
					<p class="mt-2 text-sm text-on-surface-variant">{fa.loginSubtitle}</p>
				</div>

				{#if auth.error}
					<p
						class="mb-6 flex items-start gap-3 rounded-xl border border-danger/25 bg-danger-soft px-4 py-3 text-sm leading-6 text-danger"
						role="alert"
						aria-live="assertive"
					>
						<svg viewBox="0 0 20 20" fill="none" class="mt-1 size-4 shrink-0" aria-hidden="true"
							><path
								d="M10 6v4m0 3h.01M8.27 3.8 1.9 14.83A1.35 1.35 0 0 0 3.07 16.8h13.86a1.35 1.35 0 0 0 1.17-1.97L11.73 3.8a2 2 0 0 0-3.46 0Z"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							/></svg
						>
						{auth.error}
					</p>
				{/if}

				<form class="space-y-5" onsubmit={onSubmit} novalidate>
					<div class="space-y-2">
						<label class="block text-sm font-semibold" for="phone">{fa.phone}</label>
						<input
							id="phone"
							class="input-base bg-white transition-colors hover:border-primary/50"
							type="tel"
							inputmode="numeric"
							autocomplete="tel"
							dir="ltr"
							placeholder={fa.phonePlaceholder}
							bind:value={phone}
							oninput={() => (auth.error = null)}
							aria-invalid={phoneError ? 'true' : undefined}
							aria-describedby={phoneError ? 'phone-error' : undefined}
						/>
						{#if phoneError}
							<p id="phone-error" class="text-xs text-danger" aria-live="polite">{phoneError}</p>
						{/if}
					</div>

					<div class="space-y-2">
						<label class="block text-sm font-semibold" for="password">{fa.password}</label>
						<div class="relative">
							<input
								id="password"
								class="input-base bg-white pe-14 transition-colors hover:border-primary/50"
								type={showPassword ? 'text' : 'password'}
								autocomplete="current-password"
								bind:value={password}
								oninput={() => (auth.error = null)}
								aria-invalid={passwordError ? 'true' : undefined}
								aria-describedby={passwordError ? 'password-error' : undefined}
							/>
							<button
								type="button"
								class="absolute inset-y-0 end-0 grid min-w-12 place-items-center px-3 text-on-surface-variant transition-colors hover:text-primary"
								onclick={() => (showPassword = !showPassword)}
								aria-label={showPassword ? fa.hidePassword : fa.showPassword}
								aria-pressed={showPassword}
							>
								{#if showPassword}
									<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
										><path
											d="M3 3l18 18M10.6 10.7a2 2 0 0 0 2.7 2.7M9.9 5.2A11.7 11.7 0 0 1 12 5c5.5 0 9 7 9 7a15.6 15.6 0 0 1-3 3.7M6.2 6.3C3.9 7.8 3 12 3 12s3.5 7 9 7c1 0 1.9-.2 2.7-.5"
											stroke="currentColor"
											stroke-width="1.8"
											stroke-linecap="round"
											stroke-linejoin="round"
										/></svg
									>
								{:else}
									<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true"
										><path
											d="M3 12s3.5-7 9-7 9 7 9 7-3.5 7-9 7-9-7-9-7Z"
											stroke="currentColor"
											stroke-width="1.8"
											stroke-linejoin="round"
										/><circle
											cx="12"
											cy="12"
											r="2.5"
											stroke="currentColor"
											stroke-width="1.8"
										/></svg
									>
								{/if}
							</button>
						</div>
						{#if passwordError}
							<p id="password-error" class="text-xs text-danger" aria-live="polite">
								{passwordError}
							</p>
						{/if}
					</div>

					<button type="submit" class="btn-primary w-full" disabled={!canSubmit}>
						{#if auth.pending}
							<svg viewBox="0 0 24 24" fill="none" class="size-5 animate-spin" aria-hidden="true"
								><circle
									cx="12"
									cy="12"
									r="9"
									stroke="currentColor"
									stroke-width="3"
									opacity="0.3"
								/><path
									d="M21 12a9 9 0 0 0-9-9"
									stroke="currentColor"
									stroke-width="3"
									stroke-linecap="round"
								/></svg
							>
							<span>{fa.saving}</span>
						{:else}
							<span>{fa.login}</span>
							<svg viewBox="0 0 20 20" fill="none" class="size-4 -scale-x-100" aria-hidden="true"
								><path
									d="M3.5 10h12m0 0-4.5-4.5M15.5 10 11 14.5"
									stroke="currentColor"
									stroke-width="1.8"
									stroke-linecap="round"
									stroke-linejoin="round"
								/></svg
							>
						{/if}
					</button>
				</form>

				<div
					class="mt-6 flex items-start gap-3 rounded-xl bg-surface px-4 py-3 text-sm leading-6 text-on-surface-variant"
				>
					<svg
						viewBox="0 0 20 20"
						fill="none"
						class="mt-1 size-4 shrink-0 text-primary"
						aria-hidden="true"
						><circle cx="10" cy="10" r="7.5" stroke="currentColor" stroke-width="1.6" /><path
							d="M10 9v4m0-6h.01"
							stroke="currentColor"
							stroke-width="1.8"
							stroke-linecap="round"
						/></svg
					>
					<p>{fa.loginAccessHint}</p>
				</div>

				<div class="mt-6 flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-sm">
					<a class="font-semibold text-primary hover:underline" href="/register">{fa.register}</a>
					<a class="font-semibold text-on-surface-variant hover:text-primary" href="/setup"
						>{fa.setup}</a
					>
				</div>
			</div>

			<p class="mt-6 text-center text-xs text-on-surface-variant lg:hidden">{fa.loginSubtitle}</p>
		</div>
	</section>
</main>
