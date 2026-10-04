import { get, post } from '#lib/api/http';
import type { Page } from './shared';

export interface Notification {
	id: string;
	type: string;
	title: string;
	body?: string;
	ref_type?: string;
	ref_id?: string;
	is_read: boolean;
	read_at?: string;
	created_at: string;
}

export interface NotificationFilters {
	unreadOnly?: boolean;
	page?: number;
	page_size?: number;
}

export function listNotifications(filters: NotificationFilters = {}): Promise<Page<Notification>> {
	return get<Page<Notification>>('/me/notifications', { query: { ...filters } });
}

export function markNotificationRead(id: string): Promise<null> {
	return post<null>(`/me/notifications/${id}/read`);
}

export function markAllNotificationsRead(): Promise<null> {
	return post<null>('/me/notifications/read-all');
}
