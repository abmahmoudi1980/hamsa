import { del, get, post, put } from '#lib/api/http';
import type { Page } from './shared';

export const ANNOUNCEMENT_AUDIENCES = ['all', 'block', 'floor', 'unit'] as const;

export interface Announcement {
	id: string;
	building_id: string;
	title: string;
	body: string;
	audience_type: string;
	audience_value?: string | null;
	publish_at?: string | null;
	expire_at?: string | null;
	attachment_file?: string | null;
	created_by?: string | null;
	created_at: string;
	updated_at: string;
	is_read?: boolean;
	read_at?: string | null;
}

export interface AnnouncementInput {
	title: string;
	body: string;
	audience_type: string;
	audience_value?: string | null;
	publish_at?: string | null;
	expire_at?: string | null;
	attachment_file_id?: string | null;
}

export function listBuildingAnnouncements(
	buildingId: string,
	page = 1,
	pageSize = 20
): Promise<Page<Announcement>> {
	return get<Page<Announcement>>(`/buildings/${buildingId}/announcements`, {
		query: { page, page_size: pageSize }
	});
}

export function listMyAnnouncements(page = 1, pageSize = 20): Promise<Page<Announcement>> {
	return get<Page<Announcement>>('/me/announcements', { query: { page, page_size: pageSize } });
}

export function getAnnouncement(id: string): Promise<Announcement> {
	return get<Announcement>(`/announcements/${id}`);
}

export function getMyAnnouncement(id: string): Promise<Announcement> {
	return get<Announcement>(`/me/announcements/${id}`);
}

export function createAnnouncement(
	buildingId: string,
	input: AnnouncementInput
): Promise<Announcement> {
	return post<Announcement>(`/buildings/${buildingId}/announcements`, input);
}

export function updateAnnouncement(id: string, input: AnnouncementInput): Promise<Announcement> {
	return put<Announcement>(`/announcements/${id}`, input);
}

export function deleteAnnouncement(id: string): Promise<null> {
	return del<null>(`/announcements/${id}`);
}

export function markAnnouncementRead(id: string): Promise<null> {
	return post<null>(`/announcements/${id}/read`);
}
