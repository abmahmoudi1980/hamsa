import { fromIsoDate, toIsoDate } from '#lib/format/jalali';

export interface LocalSchedule {
	date: string;
	time: string;
}

export function localScheduleFromApi(value: string | null | undefined): LocalSchedule {
	if (!value) return { date: '', time: '' };
	const date = new Date(value);
	if (!Number.isFinite(date.getTime())) return { date: '', time: '' };
	const hour = String(date.getHours()).padStart(2, '0');
	const minute = String(date.getMinutes()).padStart(2, '0');
	return { date: toIsoDate(date), time: `${hour}:${minute}` };
}

export function apiScheduleFromLocal(dateValue: string, timeValue: string): string | null {
	if (!dateValue) return null;
	const date = fromIsoDate(dateValue);
	const time = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(timeValue || '00:00');
	if (!date || !time) throw new RangeError('تاریخ یا ساعت واردشده معتبر نیست.');
	date.setHours(Number(time[1]), Number(time[2]), 0, 0);
	return date.toISOString();
}
