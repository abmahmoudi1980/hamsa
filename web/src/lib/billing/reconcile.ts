/**
 * BR-08/BR-09 reconciliation arithmetic for the pre-issuance preview grid.
 *
 * The API already computes the sizes; this module only verifies the client-side
 * story the contract requires the UI to display: for every cost item, the sum of
 * the rounded per-unit shares must equal the amount the engine actually
 * distributed (`total_used`), and every amount stays a `Toman` string so a
 * building-scale total never touches a float.
 */

import { toPersianDigits } from '#lib/format/digits';
import { subtractToman, sumToman, toman, type Toman, type TomanLike } from '#lib/format/money';

export interface ShareLike {
	rounded_share: TomanLike;
}

export interface PreviewItemLike {
	total_used: TomanLike;
	shares: readonly ShareLike[];
}

export interface PreviewLike {
	items: readonly PreviewItemLike[];
}

/** Σ rounded shares of one cost item. */
export function sharesSum(shares: readonly ShareLike[]): Toman {
	return sumToman(shares.map((share) => share.rounded_share));
}

export interface ItemReconciliation {
	/** The amount the engine distributed (for `fixed`, per-unit × participants). */
	totalUsed: Toman;
	/** Σ of the rounded per-unit shares. */
	roundedSum: Toman;
	/** `roundedSum − totalUsed`; `0` when the item reconciles. */
	residual: Toman;
	reconciled: boolean;
}

export function reconcileItem(item: PreviewItemLike): ItemReconciliation {
	const totalUsed = toman(item.total_used);
	const roundedSum = sharesSum(item.shares);
	const residual = subtractToman(roundedSum, totalUsed);
	return { totalUsed, roundedSum, residual, reconciled: residual === '0' };
}

export interface PreviewReconciliation {
	items: ItemReconciliation[];
	/** Σ of per-item residuals — the overall footer's "remaining" figure. */
	residual: Toman;
	reconciled: boolean;
}

/**
 * Rolls every item's check into one summary. `reconciled` is true only when
 * every item reconciles, matching the API's period-level flag.
 */
export function reconcilePreview(preview: PreviewLike): PreviewReconciliation {
	const items = preview.items.map(reconcileItem);
	const residual = sumToman(items.map((item) => item.residual));
	return { items, residual, reconciled: items.every((item) => item.reconciled) };
}

/**
 * Renders a wire exact share (`"1600000.0000"`) without its trailing zeros,
 * in Persian digits. An empty suffix is intentional: the value is a label.
 */
export function formatExactShare(value: string): string {
	const trimmed = value.includes('.') ? value.replace(/0+$/, '').replace(/\.$/, '') : value;
	return toPersianDigits(trimmed);
}

/** Looks up one unit's share of one preview item (undefined when not a participant). */
export function shareForUnit<T extends { unit_id: string }>(
	item: { shares: readonly T[] },
	unitId: string
): T | undefined {
	return item.shares.find((share) => share.unit_id === unitId);
}
