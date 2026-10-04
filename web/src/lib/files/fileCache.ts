/**
 * Bearer-authenticated file display (research F2).
 *
 * `GET /files/{id}` is Bearer-gated, so a bare `<img src="/files/...">` sends no
 * Authorization header and silently fails. This module fetches the bytes
 * through the shared HTTP client (which attaches the token and refreshes on a
 * 401) and returns an object URL the browser can render.
 *
 * Object URLs are cached in a small LRU so a list of receipts does not refetch
 * on every navigation, and evicted URLs are revoked so blobs are not leaked.
 * `clearFileObjectUrls` must run on sign-out.
 *
 * The stored expense `receipt_file` is `<fileId>.<ext>` (storage.Save), so
 * `fileIdFromRef` recovers the id from either a path or a bare id.
 */

import { getBlob } from '#lib/api/http';
import { isUuid } from '#lib/config';

/** Keep at most this many object URLs alive before revoking the oldest. */
const MAX_CACHED = 24;

const cache = new Map<string, Promise<string>>();

/**
 * Recovers the file id from a stored receipt path (`<id>.<ext>`) or a bare id.
 * Returns null for anything that is not a UUID, so a junk value cannot build an
 * API path.
 */
export function fileIdFromRef(ref: string | null | undefined): string | null {
	if (!ref) return null;
	const value = ref.trim();
	if (value === '') return null;
	const dot = value.indexOf('.');
	const candidate = dot === -1 ? value : value.slice(0, dot);
	return isUuid(candidate) ? candidate : null;
}

/** True when the reference looks like a renderable image. */
export function isImageRef(ref: string | null | undefined): boolean {
	return /\.(jpe?g|png|gif|webp)$/i.test(ref ?? '');
}

/**
 * Resolves a file reference to a renderable object URL, reusing the LRU.
 * Rejects when the reference is not a valid file id or the download fails.
 */
export async function getFileObjectUrl(ref: string): Promise<string> {
	const id = fileIdFromRef(ref);
	if (!id) throw new Error('شناسه فایل نامعتبر است.');
	const existing = cache.get(id);
	if (existing) {
		// Refresh recency (Map preserves insertion order).
		cache.delete(id);
		cache.set(id, existing);
		return existing;
	}
	const pending = fetchAsObjectUrl(id);
	cache.set(id, pending);
	evictOverflow();
	return pending;
}

async function fetchAsObjectUrl(id: string): Promise<string> {
	try {
		const blob = await getBlob(`/files/${id}`);
		return URL.createObjectURL(blob);
	} catch (error) {
		cache.delete(id);
		throw error;
	}
}

function evictOverflow(): void {
	while (cache.size > MAX_CACHED) {
		const oldest = cache.keys().next();
		if (oldest.done) break;
		const key = oldest.value;
		const pending = cache.get(key);
		cache.delete(key);
		if (pending) void pending.then(revoke).catch(() => {});
	}
}

/** Revokes every cached object URL. Call on sign-out. */
export function clearFileObjectUrls(): void {
	for (const pending of cache.values()) {
		void pending.then(revoke).catch(() => {});
	}
	cache.clear();
}

function revoke(url: string): void {
	URL.revokeObjectURL(url);
}

/** Cache size — exposed for tests. */
export function fileCacheSize(): number {
	return cache.size;
}
