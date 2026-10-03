<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { auth } from '#lib/auth/auth.svelte';
	import { ApiError } from '#lib/api/apiError';
	import { listBuildings } from '#lib/api/endpoints/buildings';
	import { getManagerDashboard, type DashboardAlert } from '#lib/api/endpoints/dashboard';
	import { buildingSelection, selectBuilding } from '#lib/building/selection.svelte';
	import { formatTomanWithUnit, subtractToman } from '#lib/format/money';
	import { fromIsoDate, jalaliMonthLabel, toJalali } from '#lib/format/jalali';
	import { fa } from '#i18n/fa';
	import { qk } from '#lib/query/keys';

	const buildingsQuery = createQuery(() => ({
		queryKey: qk.buildings.list(),
		queryFn: listBuildings,
		initialData: auth.user?.role === 'superadmin' ? [] : undefined
	}));

	const buildings = $derived(buildingsQuery.data ?? []);
	const selectedBuilding = $derived(
		buildings.find((building) => building.id === buildingSelection.id) ?? buildings[0] ?? null
	);

	$effect(() => {
		const items = buildingsQuery.data;
		const first = items?.at(0);
		if (first && !items?.some((building) => building.id === buildingSelection.id)) {
			selectBuilding(first.id);
		}
	});

	const dashboardQuery = createQuery(() => ({
		queryKey: selectedBuilding
			? qk.dashboard.manager(selectedBuilding.id)
			: ['dashboard', 'manager', 'none'],
		queryFn: () => getManagerDashboard(selectedBuilding!.id),
		enabled: selectedBuilding !== null
	}));

	const occupancyPercent = $derived.by(() => {
		const dashboard = dashboardQuery.data;
		if (!dashboard || dashboard.unit_count === 0) return 0;
		return Math.min(100, Math.round((dashboard.occupied_unit_count / dashboard.unit_count) * 100));
	});

	function alertLabel(alert: DashboardAlert): string {
		if (alert.title) return alert.title;
		switch (alert.kind) {
			case 'debtor_unit':
				return alert.unit_number
					? `${fa.unit} ${alert.unit_number} · ${fa.alertDebtor}`
					: fa.alertDebtor;
			case 'past_due_invoice':
				return fa.alertPastDue;
			case 'open_request':
				return fa.alertMaintenance;
			case 'pending_expense':
				return fa.alertExpense;
			default:
				return fa.attentionNeeded;
		}
	}

	function alertTone(severity: number): string {
		if (severity >= 2) return 'border-danger/20 bg-danger-soft text-danger';
		if (severity === 1) return 'border-warning/20 bg-warning-soft text-warning';
		return 'border-info/20 bg-info-soft text-info';
	}

	function jalaliMonth(value: string): string {
		const parts = value.split('-');
		if (parts.length !== 2) return '';
		const year = Number(parts[0]);
		const month = Number(parts[1]);
		if (!Number.isInteger(year) || month < 1 || month > 12) return '';
		const date = fromIsoDate(`${value}-01`);
		if (!date) return '';
		const jalali = toJalali(date);
		return jalaliMonthLabel(jalali.year, jalali.month);
	}
</script>

{#if buildingsQuery.isPending}
	<div class="space-y-5" aria-label={fa.loading}>
		<div class="h-10 w-56 animate-pulse rounded-xl bg-surface-high"></div>
		<div class="h-56 animate-pulse rounded-3xl bg-primary/10"></div>
		<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
			{#each [1, 2, 3, 4] as item (item)}
				<div class="h-32 animate-pulse rounded-2xl bg-white"></div>
			{/each}
		</div>
	</div>
{:else if buildingsQuery.isError}
	<section class="mx-auto max-w-xl card p-8 text-center">
		<span class="mx-auto grid size-14 place-items-center rounded-2xl bg-danger-soft text-danger">
			<svg viewBox="0 0 24 24" fill="none" class="size-7" aria-hidden="true"
				><path
					d="M12 8v4m0 4h.01M10.3 3.9 2.6 17.2a1.8 1.8 0 0 0 1.6 2.7h15.6a1.8 1.8 0 0 0 1.6-2.7L13.7 3.9a2 2 0 0 0-3.4 0Z"
					stroke="currentColor"
					stroke-width="1.7"
					stroke-linecap="round"
					stroke-linejoin="round"
				/></svg
			>
		</span>
		<h1 class="mt-4 text-xl font-bold">{fa.errorGeneric}</h1>
		<p class="mt-2 text-sm leading-6 text-on-surface-variant">
			{buildingsQuery.error instanceof ApiError ? buildingsQuery.error.message : fa.errorGeneric}
		</p>
		<button class="mt-6 btn-primary" type="button" onclick={() => void buildingsQuery.refetch()}
			>{fa.retry}</button
		>
	</section>
{:else if auth.user?.role === 'superadmin'}
	<section class="relative overflow-hidden rounded-3xl bg-primary p-7 text-on-primary sm:p-10">
		<div class="absolute -end-12 -top-16 size-64 rounded-full border border-white/10"></div>
		<div class="relative max-w-2xl">
			<p class="text-sm font-semibold text-white/75">{fa.appName}</p>
			<h1 class="mt-3 text-3xl leading-tight font-bold sm:text-4xl">{fa.superadminHomeTitle}</h1>
			<p class="mt-4 max-w-xl text-sm leading-7 text-white/80">{fa.superadminHomeHint}</p>
		</div>
	</section>
{:else if buildings.length === 0}
	<section class="relative overflow-hidden rounded-3xl bg-primary p-7 text-on-primary sm:p-10">
		<div class="absolute -end-12 -top-16 size-64 rounded-full border border-white/10"></div>
		<div class="relative max-w-2xl">
			<p class="text-sm font-semibold text-white/75">{fa.welcome}، {auth.user?.name}</p>
			<h1 class="mt-3 text-3xl leading-tight font-bold sm:text-4xl">{fa.emptyBuildingsTitle}</h1>
			<p class="mt-4 max-w-xl text-sm leading-7 text-white/80">{fa.emptyBuildingsBody}</p>
			<a
				class="mt-7 inline-flex min-h-12 items-center gap-2 rounded-xl bg-white px-5 font-bold text-primary transition-colors hover:bg-white/90"
				href="/m/buildings/new"
			>
				<span aria-hidden="true">＋</span>{fa.createFirstBuilding}
			</a>
		</div>
	</section>
{:else if dashboardQuery.isPending}
	<div class="space-y-5" aria-label={fa.loading}>
		<div class="h-10 w-64 animate-pulse rounded-xl bg-surface-high"></div>
		<div class="h-56 animate-pulse rounded-3xl bg-primary/10"></div>
		<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
			{#each [1, 2, 3, 4] as item (item)}
				<div class="h-32 animate-pulse rounded-2xl bg-white"></div>
			{/each}
		</div>
	</div>
{:else if dashboardQuery.isError}
	<section class="card p-8 text-center">
		<h1 class="text-xl font-bold">{fa.dashboardTitle}</h1>
		<p class="mt-2 text-sm text-on-surface-variant">
			{dashboardQuery.error instanceof ApiError ? dashboardQuery.error.message : fa.errorGeneric}
		</p>
		<button class="mt-5 btn-primary" type="button" onclick={() => void dashboardQuery.refetch()}
			>{fa.retry}</button
		>
	</section>
{:else if dashboardQuery.data}
	{@const dashboard = dashboardQuery.data}
	<div class="space-y-6 sm:space-y-8">
		<header class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<p class="text-sm font-semibold text-primary">{fa.welcome}، {auth.user?.name}</p>
				<h1 class="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{selectedBuilding?.name}</h1>
				<p class="mt-1 text-sm text-on-surface-variant">{fa.buildingOverview}</p>
			</div>
			<a class="btn-secondary min-h-11" href="/m/buildings">{fa.openBuildings}</a>
		</header>

		<section
			class="relative overflow-hidden rounded-3xl bg-primary p-6 text-on-primary shadow-[0_24px_70px_-35px_rgba(0,105,92,0.65)] sm:p-9"
		>
			<div
				aria-hidden="true"
				class="absolute -end-12 -top-24 size-72 rounded-full border border-white/10"
			></div>
			<div
				aria-hidden="true"
				class="absolute -end-2 -top-14 size-52 rounded-full border border-white/10"
			></div>
			<div class="relative grid gap-8 lg:grid-cols-[1fr_auto] lg:items-end">
				<div>
					<p class="text-sm font-semibold text-white/75">{fa.totalDebt}</p>
					<p class="mt-2 text-3xl font-bold tracking-tight sm:text-5xl">
						{formatTomanWithUnit(dashboard.total_debt)}
					</p>
					<p class="mt-3 text-sm text-white/75">
						{fa.debtorUnits}: {dashboard.debtor_unit_count.toLocaleString('fa-IR')}
					</p>
				</div>
				<div
					class="w-full max-w-sm rounded-2xl border border-white/15 bg-white/10 p-4 backdrop-blur"
				>
					<div class="flex items-center justify-between text-sm">
						<span class="text-white/80">{fa.occupiedUnits}</span>
						<span class="font-semibold"
							>{dashboard.occupied_unit_count.toLocaleString('fa-IR')} / {dashboard.unit_count.toLocaleString(
								'fa-IR'
							)}</span
						>
					</div>
					<div class="mt-3 h-2 overflow-hidden rounded-full bg-black/15">
						<div
							class="h-full rounded-full bg-emerald-200 transition-[width]"
							style={`width:${occupancyPercent}%`}
						></div>
					</div>
					<p class="mt-2 text-xs text-white/70">
						{fa.occupancyRate} · {occupancyPercent.toLocaleString('fa-IR')}٪
					</p>
				</div>
			</div>
		</section>

		<section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-label={fa.buildingOverview}>
			<article class="metric-card">
				<span class="metric-icon bg-info-soft text-info"><span aria-hidden="true">⌂</span></span>
				<p class="metric-label">{fa.unitsCount}</p>
				<p class="metric-value">{dashboard.unit_count.toLocaleString('fa-IR')}</p>
			</article>
			<article class="metric-card">
				<span class="metric-icon bg-warning-soft text-warning"
					><span aria-hidden="true">◷</span></span
				>
				<p class="metric-label">{fa.openRequests}</p>
				<p class="metric-value">{dashboard.open_requests.toLocaleString('fa-IR')}</p>
			</article>
			<article class="metric-card">
				<span class="metric-icon bg-success-soft text-success"
					><span aria-hidden="true">↙</span></span
				>
				<p class="metric-label">{fa.collectedThisMonth}</p>
				<p class="metric-value text-success">{formatTomanWithUnit(dashboard.month_income)}</p>
			</article>
			<article class="metric-card">
				<span class="metric-icon bg-danger-soft text-danger"><span aria-hidden="true">↗</span></span
				>
				<p class="metric-label">{fa.monthlyExpense}</p>
				<p class="metric-value text-danger">{formatTomanWithUnit(dashboard.month_expense)}</p>
			</article>
		</section>

		<div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
			<section class="card p-5 sm:p-6">
				<div class="flex items-center justify-between gap-4">
					<div>
						<p class="text-xs font-semibold text-primary">{fa.monthToDate}</p>
						<h2 class="mt-1 text-lg font-bold">{fa.monthlyNet}</h2>
					</div>
					<span
						class="rounded-full bg-surface px-3 py-1 text-xs font-semibold text-on-surface-variant"
						>{jalaliMonth(dashboard.month)}</span
					>
				</div>
				<div class="mt-5 grid gap-4 sm:grid-cols-3">
					<div class="rounded-2xl bg-success-soft/60 p-4">
						<p class="text-xs text-on-surface-variant">{fa.monthlyIncome}</p>
						<p class="mt-2 text-lg font-bold text-success">
							{formatTomanWithUnit(dashboard.month_income)}
						</p>
					</div>
					<div class="rounded-2xl bg-danger-soft/60 p-4">
						<p class="text-xs text-on-surface-variant">{fa.monthlyExpense}</p>
						<p class="mt-2 text-lg font-bold text-danger">
							{formatTomanWithUnit(dashboard.month_expense)}
						</p>
					</div>
					<div class="rounded-2xl bg-surface p-4">
						<p class="text-xs text-on-surface-variant">{fa.monthlyNet}</p>
						<p class="mt-2 text-lg font-bold">
							{formatTomanWithUnit(subtractToman(dashboard.month_income, dashboard.month_expense))}
						</p>
					</div>
				</div>
			</section>

			<section class="card p-5 sm:p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-xs font-semibold text-primary">{fa.attentionNeeded}</p>
						<h2 class="mt-1 text-lg font-bold">{fa.alerts}</h2>
					</div>
					<span
						class="grid size-10 place-items-center rounded-xl bg-warning-soft font-bold text-warning"
						>{dashboard.alerts.length.toLocaleString('fa-IR')}</span
					>
				</div>
				{#if dashboard.alerts.length}
					<ul class="mt-4 space-y-2">
						{#each dashboard.alerts.slice(0, 5) as alert, index (`${alert.kind}-${alert.ref_id ?? alert.unit_id ?? index}`)}
							<li
								class={`flex items-center gap-3 rounded-xl border px-3 py-3 text-sm ${alertTone(alert.severity)}`}
							>
								<span class="size-2 shrink-0 rounded-full bg-current"></span>
								<span class="min-w-0 flex-1 truncate">{alertLabel(alert)}</span>
								{#if alert.amount}
									<span class="shrink-0 text-xs font-bold">{formatTomanWithUnit(alert.amount)}</span
									>
								{/if}
							</li>
						{/each}
					</ul>
				{:else}
					<div
						class="mt-5 rounded-2xl bg-surface px-4 py-6 text-center text-sm text-on-surface-variant"
					>
						{fa.noAlerts}
					</div>
				{/if}
			</section>
		</div>
	</div>
{/if}
