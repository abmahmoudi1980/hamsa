<script lang="ts">
	/**
	 * Renders a Bearer-gated attachment (F2) as an object URL.
	 *
	 * `fileRef` is an expense's `receipt_file` path or a bare file id. Images
	 * render inline; PDFs render as an open link. All fetching goes through
	 * `#lib/files/fileCache`, so a bare `/files/...` URL is never used and the
	 * ESLint guard against it stays satisfied.
	 */
	import { isImageRef, fileIdFromRef, getFileObjectUrl } from '#lib/files/fileCache';
	import { fa } from '#i18n/fa';

	interface Props {
		fileRef?: string | null;
		/** Accessible name for the image / link. */
		label?: string;
		/** MIME type when known (a freshly uploaded id carries no extension). */
		contentType?: string;
	}

	let { fileRef = null, label = fa.receipt, contentType = '' }: Props = $props();

	let url = $state('');
	let error = $state('');
	let loading = $state(false);
	/** Plain (non-reactive) run token guarding against out-of-order resolves. */
	let run = 0;

	const hasFile = $derived(fileIdFromRef(fileRef) !== null);
	const isImage = $derived(contentType ? contentType.startsWith('image/') : isImageRef(fileRef));

	$effect(() => {
		const ref = fileRef;
		const id = fileIdFromRef(ref);
		const token = ++run;
		url = '';
		error = '';
		if (!id) {
			loading = false;
			return;
		}
		loading = true;
		getFileObjectUrl(id)
			.then((value) => {
				if (token === run) url = value;
			})
			.catch(() => {
				if (token === run) error = fa.receiptLoadFailed;
			})
			.finally(() => {
				if (token === run) loading = false;
			});
	});
</script>

{#if !hasFile}
	<p class="text-xs text-on-surface-variant">{fa.noReceipt}</p>
{:else if loading}
	<div class="h-36 w-full max-w-xs animate-pulse rounded-xl bg-surface-high"></div>
{:else if error}
	<p class="text-xs text-danger" role="alert">{error}</p>
{:else if url}
	<a
		href={url}
		target="_blank"
		rel="noopener"
		class="inline-block rounded-xl focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
	>
		{#if isImage}
			<img
				src={url}
				alt={label}
				class="max-h-72 w-full max-w-xs rounded-xl border border-outline-variant/60 object-contain"
			/>
		{:else}
			<span
				class="inline-flex items-center gap-2 text-sm font-semibold text-primary hover:underline"
			>
				<svg viewBox="0 0 24 24" fill="none" class="size-5" aria-hidden="true">
					<path
						d="M7 3h7l5 5v13a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Z"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linejoin="round"
					/>
					<path d="M14 3v5h5" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round" />
				</svg>
				{fa.viewReceiptFile}
			</span>
		{/if}
	</a>
{/if}
