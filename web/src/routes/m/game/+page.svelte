<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '#lib/auth/auth.svelte';
	import { toPersianDigits } from '#lib/format/digits';
	import { fa } from '#i18n/fa';
	import {
		GAME_ACTIONS,
		GAME_EVENTS,
		canAffordAction,
		canAffordEventChoice,
		chooseAction,
		createCampaign,
		resolveEventChoice,
		scoreCampaign
	} from '#lib/game/engine';
	import { clearCampaign, gameSaveKey, loadCampaign, saveCampaign } from '#lib/game/persistence';
	import type {
		CampaignState,
		GameActionId,
		GameEventChoiceId,
		GameEventId
	} from '#lib/game/model';
	import { CAMPAIGN_MONTHS, MONTHLY_INCOME } from '#lib/game/model';

	let campaign = $state<CampaignState | null>(null);
	let savedCampaign = $state<CampaignState | null>(null);
	let storageKey = $state<string | null>(null);
	let initialized = $state(false);
	let storageWarning = $state(false);
	let confirmation = $state<'fresh' | 'reset' | null>(null);
	let announcement = $state('');

	const actionCopy = {
		maintenance: {
			title: fa.gameActionMaintenanceTitle,
			description: fa.gameActionMaintenanceDescription
		},
		insulation: {
			title: fa.gameActionInsulationTitle,
			description: fa.gameActionInsulationDescription
		},
		garden: { title: fa.gameActionGardenTitle, description: fa.gameActionGardenDescription },
		elevator: { title: fa.gameActionElevatorTitle, description: fa.gameActionElevatorDescription }
	} satisfies Record<GameActionId, { title: string; description: string }>;

	const eventCopy = {
		waterLeak: { title: fa.gameEventWaterLeakTitle, description: fa.gameEventWaterLeakDescription },
		energySpike: {
			title: fa.gameEventEnergySpikeTitle,
			description: fa.gameEventEnergySpikeDescription
		},
		liftFailure: {
			title: fa.gameEventLiftFailureTitle,
			description: fa.gameEventLiftFailureDescription
		},
		gardenRequest: {
			title: fa.gameEventGardenRequestTitle,
			description: fa.gameEventGardenRequestDescription
		}
	} satisfies Record<GameEventId, { title: string; description: string }>;

	const choiceCopy = {
		repairLeak: {
			title: fa.gameChoiceRepairLeakTitle,
			description: fa.gameChoiceRepairLeakDescription
		},
		deferLeak: {
			title: fa.gameChoiceDeferLeakTitle,
			description: fa.gameChoiceDeferLeakDescription
		},
		installLighting: {
			title: fa.gameChoiceInstallLightingTitle,
			description: fa.gameChoiceInstallLightingDescription
		},
		reduceLighting: {
			title: fa.gameChoiceReduceLightingTitle,
			description: fa.gameChoiceReduceLightingDescription
		},
		urgentLiftRepair: {
			title: fa.gameChoiceUrgentLiftRepairTitle,
			description: fa.gameChoiceUrgentLiftRepairDescription
		},
		delayLiftRepair: {
			title: fa.gameChoiceDelayLiftRepairTitle,
			description: fa.gameChoiceDelayLiftRepairDescription
		},
		buildGarden: {
			title: fa.gameChoiceBuildGardenTitle,
			description: fa.gameChoiceBuildGardenDescription
		},
		postponeGarden: {
			title: fa.gameChoicePostponeGardenTitle,
			description: fa.gameChoicePostponeGardenDescription
		}
	} satisfies Record<GameEventChoiceId, { title: string; description: string }>;

	const metricLabels = {
		condition: fa.gameCondition,
		satisfaction: fa.gameSatisfaction,
		efficiency: fa.gameEfficiency
	} as const;

	const currentEvent = $derived(
		campaign?.currentEventId
			? (GAME_EVENTS.find((event) => event.id === campaign?.currentEventId) ?? null)
			: null
	);
	const finalScore = $derived(campaign?.phase === 'complete' ? scoreCampaign(campaign) : null);
	const recentHistory = $derived(campaign ? [...campaign.history].reverse().slice(0, 4) : []);

	onMount(() => {
		if (auth.status !== 'authenticated' || auth.user?.role !== 'manager') {
			void goto('/m', { replace: true });
			return;
		}

		storageKey = gameSaveKey(auth.user.id);
		const storage = browserStorage();
		if (storage) savedCampaign = loadCampaign(storage, storageKey);
		else storageWarning = true;
		initialized = true;
	});

	$effect(() => {
		const current = campaign;
		const key = storageKey;
		if (!initialized || !current || !key) return;

		const storage = browserStorage();
		storageWarning = !storage || !saveCampaign(storage, key, current);
	});

	function browserStorage(): Storage | null {
		try {
			return window.localStorage;
		} catch {
			return null;
		}
	}

	function startCampaign() {
		announcement = '';
		if (savedCampaign) {
			confirmation = 'fresh';
			return;
		}
		campaign = createFreshCampaign();
	}

	function createFreshCampaign(): CampaignState {
		return createCampaign(Math.floor(Math.random() * 0x100000000));
	}

	function resumeCampaign() {
		if (!savedCampaign) return;
		campaign = savedCampaign;
		savedCampaign = null;
		confirmation = null;
	}

	function chooseBuildingAction(actionId: GameActionId) {
		if (!campaign) return;
		const result = chooseAction(campaign, actionId);
		if (!result.ok) {
			announcement = result.reason === 'unaffordable' ? fa.gameUnaffordable : fa.errorGeneric;
			return;
		}
		campaign = result.state;
		announcement = fa.gameActionApplied;
	}

	function chooseEventResponse(choiceId: GameEventChoiceId) {
		if (!campaign) return;
		const result = resolveEventChoice(campaign, choiceId);
		if (!result.ok) {
			announcement = result.reason === 'unaffordable' ? fa.gameUnaffordable : fa.errorGeneric;
			return;
		}
		campaign = result.state;
		announcement =
			result.state.phase === 'complete' ? fa.gameCampaignComplete : fa.gameEventResolved;
	}

	function requestReset() {
		confirmation = 'reset';
	}

	function confirmRestart() {
		const shouldStartFresh = confirmation === 'fresh';
		if (storageKey) {
			const storage = browserStorage();
			if (!storage || !clearCampaign(storage, storageKey)) storageWarning = true;
		}
		campaign = createFreshCampaign();
		savedCampaign = null;
		confirmation = null;
		announcement = shouldStartFresh ? '' : fa.gameResetDone;
	}

	function formatNumber(value: number): string {
		return toPersianDigits(String(value));
	}

	function formatSigned(value: number): string {
		return `${value > 0 ? '+' : '−'}${formatNumber(Math.abs(value))}`;
	}

	function metricEffects(condition: number, satisfaction: number, efficiency: number): string {
		return [
			condition ? `${formatSigned(condition)} ${fa.gameConditionShort}` : '',
			satisfaction ? `${formatSigned(satisfaction)} ${fa.gameSatisfactionShort}` : '',
			efficiency ? `${formatSigned(efficiency)} ${fa.gameEfficiencyShort}` : ''
		]
			.filter(Boolean)
			.join(' · ');
	}

	function eventTitle(eventId: GameEventId): string {
		return eventCopy[eventId].title;
	}
</script>

<svelte:head>
	<title>{fa.gameTitle} | {fa.appName}</title>
</svelte:head>

<div class="space-y-6">
	<header class="flex flex-wrap items-start justify-between gap-4">
		<div>
			<p class="text-sm font-semibold text-primary">{fa.navStrategyGame}</p>
			<h1 class="mt-1 text-2xl font-bold text-on-surface sm:text-3xl">{fa.gameTitle}</h1>
			<p class="mt-2 max-w-2xl text-sm text-on-surface-variant">{fa.gameSubtitle}</p>
		</div>
		<a class="btn-secondary" href="/m">{fa.back}</a>
	</header>

	{#if storageWarning}
		<div
			class="rounded-xl border border-warning bg-warning-soft px-4 py-3 text-sm text-on-surface"
			role="status"
		>
			{fa.gameStorageWarning}
		</div>
	{/if}

	<div class="sr-only" aria-live="polite" aria-atomic="true">{announcement}</div>

	{#if confirmation}
		<section
			class="rounded-2xl border border-warning bg-warning-soft p-5 sm:p-6"
			aria-labelledby="game-reset-title"
		>
			<h2 id="game-reset-title" class="text-lg font-bold">{fa.gameResetTitle}</h2>
			<p class="mt-2 text-sm text-on-surface-variant">{fa.gameResetDescription}</p>
			<div class="mt-4 flex flex-wrap gap-3">
				<button class="btn-primary" type="button" onclick={confirmRestart}
					>{fa.gameResetConfirm}</button
				>
				<button class="btn-secondary" type="button" onclick={() => (confirmation = null)}
					>{fa.gameResetCancel}</button
				>
			</div>
		</section>
	{:else if campaign === null}
		<section
			class="grid gap-6 overflow-hidden rounded-3xl border border-outline-variant bg-white p-5 sm:p-8 lg:grid-cols-[1.3fr_0.7fr] lg:items-center"
		>
			<div>
				<p
					class="inline-flex rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold text-primary"
				>
					{fa.gameCredits}
				</p>
				<h2 class="mt-4 max-w-xl text-2xl leading-tight font-bold sm:text-3xl">
					{fa.gameSubtitle}
				</h2>
				<p class="mt-3 max-w-xl text-sm leading-7 text-on-surface-variant">
					{fa.gameSandboxNotice}
				</p>
				<p class="mt-2 max-w-xl text-sm leading-7 text-on-surface-variant">{fa.gameLoopHint}</p>
				<div class="mt-6 flex flex-wrap gap-3">
					{#if savedCampaign}
						<button class="btn-primary" type="button" onclick={resumeCampaign}
							>{fa.gameResume}</button
						>
					{:else}
						<button class="btn-primary" type="button" onclick={startCampaign}>{fa.gameStart}</button
						>
					{/if}
					{#if savedCampaign}
						<button class="btn-secondary" type="button" onclick={startCampaign}
							>{fa.gameStartFresh}</button
						>
					{/if}
				</div>
				{#if savedCampaign}
					<p class="mt-4 text-sm text-on-surface-variant">
						{fa.gameSavedCampaign}
						{fa.gameMonth}
						{formatNumber(savedCampaign.month)}
						{fa.gameOf}
						{formatNumber(CAMPAIGN_MONTHS)}
					</p>
				{/if}
			</div>
			<div
				class="relative mx-auto flex min-h-60 w-full max-w-sm items-end justify-center overflow-hidden rounded-2xl bg-primary/5 p-5"
				aria-hidden="true"
			>
				<div class="absolute inset-x-0 bottom-0 h-10 bg-primary/10"></div>
				<svg viewBox="0 0 220 230" fill="none" class="relative z-10 w-full max-w-52 text-primary">
					<path d="M28 218V55L110 15l82 40v163H28Z" fill="currentColor" opacity=".14" />
					<path
						d="M28 218V55L110 15l82 40v163H28Z"
						stroke="currentColor"
						stroke-width="5"
						stroke-linejoin="round"
					/>
					<path
						d="M48 75h28v31H48zM97 75h28v31H97zM146 75h28v31h-28zM48 128h28v31H48zM97 128h28v31H97zM146 128h28v31h-28z"
						fill="currentColor"
						opacity=".32"
					/>
					<path d="M91 181h38v37H91z" fill="currentColor" opacity=".55" />
					<path d="M18 218h184" stroke="currentColor" stroke-width="5" stroke-linecap="round" />
				</svg>
			</div>
		</section>
	{:else}
		<section class="card p-5 sm:p-6" aria-labelledby="campaign-progress-title">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<p class="text-sm font-semibold text-primary">
						{fa.gameMonth}
						{formatNumber(campaign.month)}
						{fa.gameOf}
						{formatNumber(12)}
					</p>
					<h2 id="campaign-progress-title" class="mt-1 text-xl font-bold">
						{campaign.phase === 'action'
							? fa.gameChooseAction
							: campaign.phase === 'event'
								? fa.gameChooseEvent
								: fa.gameResults}
					</h2>
				</div>
				{#if campaign.phase !== 'complete'}
					<button class="btn-secondary" type="button" onclick={requestReset}>{fa.gameReset}</button>
				{/if}
			</div>
			<div
				class="mt-5 h-3 overflow-hidden rounded-full bg-surface-high"
				role="progressbar"
				aria-label={fa.gameMonth}
				aria-valuemin="0"
				aria-valuemax={CAMPAIGN_MONTHS}
				aria-valuenow={campaign.phase === 'complete' ? CAMPAIGN_MONTHS : campaign.month - 1}
			>
				<div
					class="h-full rounded-full bg-primary transition-[width] duration-300"
					style={`width: ${campaign.phase === 'complete' ? 100 : ((campaign.month - 1) / CAMPAIGN_MONTHS) * 100}%`}
				></div>
			</div>
		</section>

		<section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label={fa.gameCredits}>
			<div class="rounded-2xl bg-primary p-5 text-on-primary sm:col-span-2 xl:col-span-1">
				<p class="text-sm opacity-90">{fa.gameCredits}</p>
				<p class="mt-2 text-3xl font-bold tabular-nums">{formatNumber(campaign.credits)}</p>
				<p class="mt-1 text-xs opacity-80">
					{fa.gameMonthlyIncome}: +{formatNumber(MONTHLY_INCOME)}
				</p>
			</div>
			{#each ['condition', 'satisfaction', 'efficiency'] as const as metric (metric)}
				<div class="card p-5">
					<div class="flex items-center justify-between gap-3">
						<p class="text-sm font-semibold">{metricLabels[metric]}</p>
						<p class="text-lg font-bold tabular-nums">{formatNumber(campaign[metric])}</p>
					</div>
					<div
						class="mt-4 h-2.5 overflow-hidden rounded-full bg-surface-high"
						role="progressbar"
						aria-label={metricLabels[metric]}
						aria-valuemin="0"
						aria-valuemax="100"
						aria-valuenow={campaign[metric]}
					>
						<div
							class="h-full rounded-full bg-primary transition-[width] duration-300"
							style={`width: ${campaign[metric]}%`}
						></div>
					</div>
				</div>
			{/each}
		</section>

		{#if campaign.phase === 'action'}
			<section class="space-y-4" aria-labelledby="game-action-heading">
				<div>
					<h2 id="game-action-heading" class="text-xl font-bold">{fa.gameChooseAction}</h2>
					<p class="mt-1 text-sm text-on-surface-variant">{fa.gameChooseActionHint}</p>
				</div>
				<div class="grid gap-4 md:grid-cols-2">
					{#each GAME_ACTIONS as action (action.id)}
						{@const copy = actionCopy[action.id]}
						<button
							class="min-h-40 card p-5 text-start transition-colors hover:border-primary/60 hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-55 disabled:hover:border-outline-variant disabled:hover:bg-white"
							type="button"
							disabled={!canAffordAction(campaign, action.id)}
							data-testid={`game-action-${action.id}`}
							onclick={() => chooseBuildingAction(action.id)}
						>
							<span class="flex items-start justify-between gap-3">
								<span class="text-base font-bold">{copy.title}</span>
								<span class="shrink-0 rounded-full bg-surface-high px-3 py-1 text-xs font-semibold">
									{fa.gameCost}: {formatNumber(action.cost)}
									{fa.gameCreditsUnit}
								</span>
							</span>
							<span class="mt-2 block text-sm leading-6 text-on-surface-variant"
								>{copy.description}</span
							>
							<span class="mt-3 block text-xs font-semibold text-primary">
								{metricEffects(action.condition, action.satisfaction, action.efficiency)}
							</span>
							{#if !canAffordAction(campaign, action.id)}
								<span class="mt-2 block text-xs text-danger">{fa.gameUnaffordable}</span>
							{/if}
						</button>
					{/each}
				</div>
			</section>
		{:else if campaign.phase === 'event' && currentEvent}
			<section class="space-y-4" aria-labelledby="game-event-heading">
				<div class="rounded-2xl border border-warning/50 bg-warning-soft p-5 sm:p-6">
					<p class="text-xs font-bold text-warning">{fa.gameChooseEvent}</p>
					<h2 id="game-event-heading" class="mt-2 text-xl font-bold">
						{eventCopy[currentEvent.id].title}
					</h2>
					<p class="mt-2 text-sm leading-7 text-on-surface-variant">
						{eventCopy[currentEvent.id].description}
					</p>
				</div>
				<div class="grid gap-4 md:grid-cols-2">
					{#each currentEvent.choices as choice (choice.id)}
						{@const copy = choiceCopy[choice.id]}
						<button
							class="min-h-40 card p-5 text-start transition-colors hover:border-primary/60 hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-55 disabled:hover:border-outline-variant disabled:hover:bg-white"
							type="button"
							disabled={!canAffordEventChoice(campaign, choice.id)}
							data-testid={`game-choice-${choice.id}`}
							onclick={() => chooseEventResponse(choice.id)}
						>
							<span class="flex items-start justify-between gap-3">
								<span class="text-base font-bold">{copy.title}</span>
								<span class="shrink-0 rounded-full bg-surface-high px-3 py-1 text-xs font-semibold">
									{choice.cost
										? `${fa.gameCost}: ${formatNumber(choice.cost)} ${fa.gameCreditsUnit}`
										: fa.gameFree}
								</span>
							</span>
							<span class="mt-2 block text-sm leading-6 text-on-surface-variant"
								>{copy.description}</span
							>
							<span class="mt-3 block text-xs font-semibold text-primary">
								{metricEffects(choice.condition, choice.satisfaction, choice.efficiency)}
								{#if choice.credits > 0}{` · +${formatNumber(choice.credits)} ${fa.gameCreditGain}`}{/if}
							</span>
							{#if !canAffordEventChoice(campaign, choice.id)}
								<span class="mt-2 block text-xs text-danger">{fa.gameUnaffordable}</span>
							{/if}
						</button>
					{/each}
				</div>
			</section>
		{:else if finalScore}
			<section class="grid gap-5 lg:grid-cols-[0.8fr_1.2fr]">
				<div class="rounded-2xl bg-primary p-6 text-on-primary sm:p-8">
					<p class="text-sm opacity-90">{fa.gameScore}</p>
					<p class="mt-3 text-6xl font-bold tabular-nums" data-testid="game-score">
						{formatNumber(finalScore.score)}
					</p>
					<p class="mt-4 text-lg font-bold">
						{finalScore.outcome === 'excellent'
							? fa.gameOutcomeExcellent
							: finalScore.outcome === 'stable'
								? fa.gameOutcomeStable
								: fa.gameOutcomeStruggling}
					</p>
					<button
						class="mt-6 rounded-xl bg-white px-5 py-3 font-semibold text-primary transition-colors hover:bg-surface-high focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
						type="button"
						onclick={requestReset}
					>
						{fa.gameStartFresh}
					</button>
				</div>
				<div class="card p-5 sm:p-6">
					<h2 class="text-lg font-bold">{fa.gameResults}</h2>
					<div class="mt-4 space-y-4">
						{#each ['condition', 'satisfaction', 'efficiency'] as const as metric (metric)}
							<div
								class="flex items-center justify-between gap-4 border-b border-outline-variant/60 pb-3 last:border-0 last:pb-0"
							>
								<span class="text-sm text-on-surface-variant">{metricLabels[metric]}</span>
								<span class="font-bold tabular-nums"
									>{formatNumber(campaign[metric])} / {formatNumber(100)}</span
								>
							</div>
						{/each}
					</div>
				</div>
			</section>
		{/if}

		{#if recentHistory.length > 0}
			<section class="card p-5 sm:p-6" aria-labelledby="game-history-heading">
				<h2 id="game-history-heading" class="text-lg font-bold">{fa.gameHistory}</h2>
				<ol class="mt-4 divide-y divide-outline-variant/60">
					{#each recentHistory as record (record.month)}
						<li class="flex flex-wrap items-center justify-between gap-2 py-3 first:pt-0 last:pb-0">
							<span class="text-sm font-semibold">{fa.gameMonth} {formatNumber(record.month)}</span>
							<span class="text-sm text-on-surface-variant">
								{actionCopy[record.actionId].title} · {eventTitle(record.eventId)} · {choiceCopy[
									record.choiceId
								].title}
							</span>
						</li>
					{/each}
				</ol>
			</section>
		{/if}
	{/if}
</div>
