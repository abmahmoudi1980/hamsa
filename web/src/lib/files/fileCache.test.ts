import { describe, expect, it } from 'vitest';
import { fileIdFromRef, isImageRef } from './fileCache';

const ID = '3f2504e0-4f89-11d3-9a0c-0305e82c3301';

describe('fileIdFromRef', () => {
	it('recovers the id from a stored receipt path', () => {
		expect(fileIdFromRef(`${ID}.jpg`)).toBe(ID);
		expect(fileIdFromRef(`${ID}.pdf`)).toBe(ID);
	});

	it('accepts a bare id', () => {
		expect(fileIdFromRef(ID)).toBe(ID);
	});

	it('rejects empty, missing and non-uuid values', () => {
		expect(fileIdFromRef('')).toBeNull();
		expect(fileIdFromRef('   ')).toBeNull();
		expect(fileIdFromRef(null)).toBeNull();
		expect(fileIdFromRef(undefined)).toBeNull();
		expect(fileIdFromRef('not-a-uuid.jpg')).toBeNull();
		expect(fileIdFromRef('..hidden')).toBeNull();
	});
});

describe('isImageRef', () => {
	it('detects renderable image extensions', () => {
		expect(isImageRef(`${ID}.jpg`)).toBe(true);
		expect(isImageRef(`${ID}.JPEG`)).toBe(true);
		expect(isImageRef(`${ID}.png`)).toBe(true);
		expect(isImageRef(`${ID}.webp`)).toBe(true);
	});

	it('treats pdf and unknown extensions as non-images', () => {
		expect(isImageRef(`${ID}.pdf`)).toBe(false);
		expect(isImageRef('receipt.pdf')).toBe(false);
		expect(isImageRef(null)).toBe(false);
	});
});
