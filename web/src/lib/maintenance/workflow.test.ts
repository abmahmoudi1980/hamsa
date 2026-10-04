import { describe, expect, it } from 'vitest';
import { nextMaintenanceStatuses, MAINTENANCE_WORKFLOW } from './workflow';

describe('maintenance workflow', () => {
	it('lists the backend status tokens in process order', () => {
		expect(MAINTENANCE_WORKFLOW).toEqual(['new', 'under_review', 'in_progress', 'done', 'closed']);
	});

	it('returns only legal next states', () => {
		expect(nextMaintenanceStatuses('new')).toEqual(['under_review']);
		expect(nextMaintenanceStatuses('under_review')).toEqual(['in_progress', 'closed']);
		expect(nextMaintenanceStatuses('in_progress')).toEqual(['done', 'closed']);
		expect(nextMaintenanceStatuses('done')).toEqual(['closed']);
		expect(nextMaintenanceStatuses('closed')).toEqual([]);
		expect(nextMaintenanceStatuses('unknown')).toEqual([]);
	});
});
