/**
 * Centralised query keys (research R5).
 *
 * Every key is built here rather than inlined, so an invalidation after a
 * mutation cannot silently miss a screen: the keys are a single tree, and
 * `invalidate` calls below use prefixes to sweep whole subtrees.
 *
 * The rule that matters: a key must include every input that changes the
 * RESULT. `['units', buildingId]` alone is a bug — filter and page changes
 * would return cached data for the wrong query.
 */

export const qk = {
	auth: {
		me: () => ['auth', 'me'] as const
	},

	buildings: {
		list: () => ['buildings'] as const,
		detail: (id: string) => ['buildings', 'detail', id] as const,
		managers: (buildingId: string) => ['buildings', buildingId, 'managers'] as const
	},

	units: {
		/** `filters` is a serialized filter object; see units.ts. */
		list: (buildingId: string, filters: string) => ['units', buildingId, filters] as const,
		detail: (unitId: string) => ['units', 'detail', unitId] as const,
		history: (unitId: string) => ['units', 'detail', unitId, 'history'] as const,
		occupancies: (unitId: string) => ['units', unitId, 'occupancies'] as const,
		occupantCounts: (unitId: string) => ['units', unitId, 'occupant-counts'] as const
	},

	persons: {
		list: (buildingId: string, filters: string) => ['persons', buildingId, filters] as const
	},

	periods: {
		list: (buildingId: string) => ['periods', buildingId] as const,
		detail: (periodId: string) => ['periods', 'detail', periodId] as const,
		costItems: (periodId: string) => ['periods', periodId, 'cost-items'] as const,
		/** The pre-issuance review grid (BR-08/BR-09). */
		preview: (periodId: string) => ['periods', periodId, 'preview'] as const
	},

	invoices: {
		list: (buildingId: string, filters: string) => ['invoices', buildingId, filters] as const,
		detail: (invoiceId: string) => ['invoices', 'detail', invoiceId] as const,
		/** Resident's own invoices. */
		mine: (filters: string) => ['invoices', 'me', filters] as const
	},

	payments: {
		ledger: (buildingId: string, filters: string) =>
			['payments', 'ledger', buildingId, filters] as const,
		mine: (filters: string) => ['payments', 'me', filters] as const
	},

	expenses: {
		list: (buildingId: string, filters: string) => ['expenses', buildingId, filters] as const,
		detail: (expenseId: string) => ['expenses', 'detail', expenseId] as const,
		report: (buildingId: string, month: string) =>
			['expenses', buildingId, 'report', month] as const
	},

	maintenance: {
		/** Manager list for a building. */
		list: (buildingId: string, filters: string) => ['maintenance', buildingId, filters] as const,
		detail: (requestId: string) => ['maintenance', 'detail', requestId] as const,
		mine: (filters: string) => ['maintenance', 'me', filters] as const
	},

	announcements: {
		list: (buildingId: string, filters: string) => ['announcements', buildingId, filters] as const,
		mine: (filters: string) => ['announcements', 'me', filters] as const,
		detail: (id: string) => ['announcements', 'detail', id] as const
	},

	notifications: {
		list: (filters: string) => ['notifications', filters] as const
	},

	dashboard: {
		manager: (buildingId: string) => ['dashboard', 'manager', buildingId] as const,
		resident: () => ['dashboard', 'resident'] as const
	},

	balances: {
		unit: (unitId: string) => ['balances', 'unit', unitId] as const
	}
} as const;

/**
 * The invalidation map (research R5), expressed once so a mutation's cache
 * impact is reviewable in one place rather than scattered across features.
 *
 * Note which mutations sweep WIDE: issuing a period's invoices changes invoices,
 * unit balances, the dashboard AND residents' notifications, so all four are
 * invalidated together.
 */
export const invalidations = {
	/** Issuing/reopening/closing a period. */
	periodIssued: (buildingId: string, periodId: string) => [
		qk.periods.detail(periodId),
		qk.periods.preview(periodId),
		qk.invoices.list(buildingId, '*'),
		[qk.balances.unit('*')]
	],
	/** Cancelling an invoice or appending an adjustment note (FR-017/BR-10). */
	invoiceChanged: (buildingId: string, invoiceId: string) => [
		qk.invoices.detail(invoiceId),
		qk.invoices.list(buildingId, '*'),
		[qk.balances.unit('*')],
		qk.dashboard.manager(buildingId)
	],
	/** Recording a payment or a gateway verification. */
	paymentRecorded: (invoiceId: string, buildingId: string) => [
		qk.invoices.detail(invoiceId),
		qk.payments.ledger(buildingId, '*'),
		[qk.payments.mine('*')],
		[qk.balances.unit('*')],
		qk.dashboard.manager(buildingId)
	],
	/** Editing a unit: areas and occupant counts feed the charge engine. */
	unitChanged: (buildingId: string, unitId: string) => [
		qk.units.detail(unitId),
		qk.units.list(buildingId, '*'),
		qk.dashboard.manager(buildingId)
	],
	/** Any expense change. */
	expenseChanged: (buildingId: string) => [
		qk.expenses.list(buildingId, '*'),
		qk.dashboard.manager(buildingId)
	],
	/** A maintenance transition notifies the submitter. */
	maintenanceChanged: (buildingId: string, requestId: string) => [
		qk.maintenance.detail(requestId),
		qk.maintenance.list(buildingId, '*'),
		qk.dashboard.manager(buildingId),
		qk.notifications.list('*')
	],
	/** Publishing an announcement fans out to targeted residents. */
	announcementChanged: (buildingId: string) => [
		qk.announcements.list(buildingId, '*'),
		qk.notifications.list('*'),
		qk.dashboard.manager(buildingId),
		qk.dashboard.resident()
	],
	/** Notification read-state changes the badge everywhere. */
	notificationsChanged: () => [qk.notifications.list('*')]
} as const;
