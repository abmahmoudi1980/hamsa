export const MAINTENANCE_WORKFLOW = [
	'new',
	'under_review',
	'in_progress',
	'done',
	'closed'
] as const;

const NEXT_STATUSES: Record<string, readonly string[]> = {
	new: ['under_review'],
	under_review: ['in_progress', 'closed'],
	in_progress: ['done', 'closed'],
	done: ['closed'],
	closed: []
};

export function nextMaintenanceStatuses(status: string): readonly string[] {
	return NEXT_STATUSES[status] ?? [];
}
