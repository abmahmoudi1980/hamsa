/**
 * File uploads (contracts/api.md "Files").
 *
 * `POST /files` is multipart (`file` field) and answers `{ id, url }`. The `url`
 * is Bearer-gated and unusable as an `<img src>`; render the bound file through
 * `#lib/files/fileCache` instead.
 */

import { post } from '#lib/api/http';

export interface UploadedFile {
	id: string;
	url: string;
}

/** Maximum accepted upload size (backend storage.DefaultMaxSize, 5 MiB). */
export const MAX_UPLOAD_BYTES = 5 << 20;

export const ACCEPTED_UPLOAD_TYPES = 'image/*,application/pdf';

/**
 * Uploads one attachment and returns its registry id.
 *
 * `body` must be left undefined: the FormData travels untouched so the browser
 * sets the multipart boundary itself.
 */
export function uploadFile(file: File): Promise<UploadedFile> {
	const form = new FormData();
	form.append('file', file);
	return post<UploadedFile>('/files', undefined, { formData: form });
}
