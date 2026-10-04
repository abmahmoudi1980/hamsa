<script lang="ts">
	import { ApiError } from '#lib/api/apiError';
	import { uploadFile, MAX_UPLOAD_BYTES } from '#lib/api/endpoints/files';
	import AttachmentPreview from '#lib/components/AttachmentPreview.svelte';
	import { fa } from '#i18n/fa';

	interface Props {
		value?: string;
		label: string;
		accept?: string;
		imageOnly?: boolean;
		previewLabel?: string;
	}

	let {
		value = $bindable(''),
		label,
		accept = 'image/*,application/pdf',
		imageOnly = false,
		previewLabel = label
	}: Props = $props();

	let uploading = $state(false);
	let error = $state('');
	let contentType = $state('');

	async function selectFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		error = '';
		if (file.size > MAX_UPLOAD_BYTES) {
			error = fa.receiptTooLarge;
			return;
		}
		if (
			imageOnly
				? !file.type.startsWith('image/')
				: !file.type.startsWith('image/') && file.type !== 'application/pdf'
		) {
			error = imageOnly ? fa.imageOnlyError : fa.receiptInvalidType;
			return;
		}
		uploading = true;
		try {
			const uploaded = await uploadFile(file);
			value = uploaded.id;
			contentType = file.type;
		} catch (cause) {
			error = cause instanceof ApiError ? cause.message : fa.uploadFailed;
		} finally {
			uploading = false;
		}
	}
</script>

<div class="space-y-3">
	<label class="block text-sm font-semibold" for={`file-${label}`}>{label}</label>
	<div class="flex flex-wrap items-center gap-3">
		<label class="btn-secondary min-h-10 cursor-pointer text-sm">
			{uploading ? fa.uploadingFile : fa.chooseFile}
			<input
				id={`file-${label}`}
				class="sr-only"
				type="file"
				accept={imageOnly ? 'image/*' : accept}
				disabled={uploading}
				onchange={selectFile}
			/>
		</label>
		{#if value}
			<button
				class="text-sm font-semibold text-danger hover:underline"
				type="button"
				onclick={() => (value = '')}
			>
				{imageOnly ? fa.removePhoto : fa.clearAttachment}
			</button>
		{/if}
	</div>
	{#if error}<p class="text-sm text-danger" role="alert">{error}</p>{/if}
	{#if value}<AttachmentPreview fileRef={value} label={previewLabel} {contentType} />{/if}
</div>
