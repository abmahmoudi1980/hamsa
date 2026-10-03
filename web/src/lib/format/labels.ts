/**
 * Enum → Persian label maps.
 *
 * The API speaks locale-neutral English tokens on the wire (contracts/api.md
 * §Conventions: "Structured fields (codes, enums, ids) remain locale-neutral
 * English tokens") while the UI is Persian-only. These tables are the single
 * translation point, kept out of `fa.ts` because they are keyed by backend
 * enum values rather than by meaning — and an unknown value must fall back
 * predictably instead of rendering `undefined`.
 */

/** Shows the raw token when a value is unrecognised — never a blank. */
function label<T extends string>(
	table: Record<T, string>,
	value: string | null | undefined
): string {
	if (!value) return '—';
	return table[value as T] ?? value;
}

export const unitStatusLabel = (v: string | null | undefined) =>
	label(
		{
			active: 'فعال',
			vacant: 'خالی',
			occupied: 'در حال سکونت',
			inactive: 'غیرفعال'
		} as const,
		v
	);

export const calcMethodLabel = (v: string | null | undefined) =>
	label(
		{
			equal: 'مساوی بین واحدها',
			// The wire token is `per_occupant` (engine.MethodPerOccupant and the
			// cost_method DB enum); `per_person` is not accepted by the API.
			per_occupant: 'بر اساس تعداد نفرات',
			per_area: 'بر اساس متراژ',
			fixed: 'مبلغ ثابت',
			specific_units: 'هزینه اختصاصی',
			combined: 'فرمول ترکیبی'
		} as const,
		v
	);

export const lateFeeTypeLabel = (v: string | null | undefined) =>
	label(
		{
			none: 'بدون جریمه',
			fixed: 'مبلغ ثابت',
			percent: 'درصدی از مبلغ',
			per_day: 'روزانه'
		} as const,
		v
	);

export const periodStatusLabel = (v: string | null | undefined) =>
	label(
		{
			draft: 'پیش‌نویس',
			calculated: 'محاسبه‌شده',
			issued: 'صادرشده',
			closed: 'بسته‌شده'
		} as const,
		v
	);

export const invoiceStatusLabel = (v: string | null | undefined) =>
	label(
		{
			unpaid: 'پرداخت‌نشده',
			partial: 'پرداخت ناقص',
			paid: 'پرداخت‌شده',
			cancelled: 'لغوشده',
			expired: 'منقضی‌شده'
		} as const,
		v
	);

export const paymentMethodLabel = (v: string | null | undefined) =>
	label(
		{
			manual: 'دستی',
			gateway: 'درگاه پرداخت',
			transfer: 'حواله'
		} as const,
		v
	);

export const paymentStatusLabel = (v: string | null | undefined) =>
	label(
		{
			recorded: 'ثبت‌شده',
			verified: 'تأییدشده',
			failed: 'ناموفق'
		} as const,
		v
	);

export const approvalStatusLabel = (v: string | null | undefined) =>
	label(
		{
			pending: 'در انتظار تأیید',
			approved: 'تأییدشده',
			rejected: 'ردشده'
		} as const,
		v
	);

export const maintenanceStatusLabel = (v: string | null | undefined) =>
	label(
		{
			new: 'جدید',
			under_review: 'در حال بررسی',
			in_progress: 'در حال انجام',
			resolved: 'انجام شد',
			closed: 'بسته شد'
		} as const,
		v
	);

export const priorityLabel = (v: string | null | undefined) =>
	label(
		{
			normal: 'عادی',
			important: 'مهم',
			urgent: 'فوری'
		} as const,
		v
	);

export const maintenanceCategoryLabel = (v: string | null | undefined) =>
	label(
		{
			elevator: 'آسانسور',
			facilities: 'تأسیسات',
			electricity: 'برق',
			water: 'آب',
			cleaning: 'نظافت',
			common_areas: 'مشاعات',
			parking: 'پارکینگ',
			other: 'سایر'
		} as const,
		v
	);

export const expenseCategoryLabel = (v: string | null | undefined) =>
	label(
		{
			water: 'آب',
			electricity: 'برق',
			gas: 'گاز',
			elevator: 'آسانسور',
			cleaning: 'نظافت',
			security: 'نگهبانی',
			repairs: 'تعمیرات',
			insurance: 'بیمه',
			equipment: 'تجهیزات',
			other: 'سایر'
		} as const,
		v
	);

export const relationshipLabel = (v: string | null | undefined) =>
	label(
		{
			owner: 'مالک',
			tenant: 'مستأجر',
			non_resident_owner: 'مالک غیرساکن'
		} as const,
		v
	);

export const audienceTypeLabel = (v: string | null | undefined) =>
	label(
		{
			all: 'همه ساختمان',
			block: 'یک بلوک',
			floor: 'یک طبقه',
			unit: 'واحد مشخص'
		} as const,
		v
	);

export const roleLabel = (v: string | null | undefined) =>
	label(
		{
			superadmin: 'مدیر ارشد',
			manager: 'مدیر ساختمان',
			resident: 'ساکن'
		} as const,
		v
	);

/** Tone classes for a status chip, keyed by semantic group. */
export type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info';

export const invoiceStatusTone = (v: string | null | undefined): Tone =>
	v === 'paid'
		? 'success'
		: v === 'partial'
			? 'warning'
			: v === 'cancelled' || v === 'expired'
				? 'danger'
				: 'neutral';

export const periodStatusTone = (v: string | null | undefined): Tone =>
	v === 'closed' ? 'neutral' : v === 'issued' ? 'success' : v === 'calculated' ? 'info' : 'warning';

export const maintenanceStatusTone = (v: string | null | undefined): Tone =>
	v === 'closed'
		? 'neutral'
		: v === 'resolved'
			? 'success'
			: v === 'in_progress' || v === 'under_review'
				? 'info'
				: 'warning';

export const priorityTone = (v: string | null | undefined): Tone =>
	v === 'urgent' ? 'danger' : v === 'important' ? 'warning' : 'neutral';

export const approvalStatusTone = (v: string | null | undefined): Tone =>
	v === 'approved' ? 'success' : v === 'rejected' ? 'danger' : 'warning';

export const paymentStatusTone = (v: string | null | undefined): Tone =>
	v === 'verified' ? 'success' : v === 'failed' ? 'danger' : 'warning';

export const unitStatusTone = (v: string | null | undefined): Tone =>
	v === 'occupied' ? 'success' : v === 'vacant' ? 'warning' : v === 'inactive' ? 'danger' : 'info';
