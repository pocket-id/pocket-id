<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Popover from '$lib/components/ui/popover';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils/style';
	import { LucideLink, LucideUpload, LucideX } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		logoDataURL,
		clientName,
		resetLogo,
		onLogoChange,
		light,
		tabTriggers
	}: {
		logoDataURL: string | null;
		clientName: string;
		resetLogo: () => void;
		onLogoChange: (file: File | string | null) => void;
		tabTriggers?: Snippet;
		light: boolean;
	} = $props();

	const id = `oidc-client-logo-${light ? 'light' : 'dark'}`;

	let url = $state('');
	let hasUrlError = $state(false);
	let isDraggingOver = $state(false);

	function selectFile(file: File | null) {
		url = '';
		hasUrlError = false;
		onLogoChange(file);
	}

	function onFileChange(e: Event) {
		selectFile((e.target as HTMLInputElement).files?.[0] || null);
	}

	// Dropping an image on the preview behaves like picking it in the file dialog
	function onDrop(e: DragEvent) {
		e.preventDefault();
		isDraggingOver = false;
		const file = e.dataTransfer?.files[0];
		if (file?.type.startsWith('image/')) {
			selectFile(file);
		}
	}

	function onUrlChange(e: Event) {
		const value = (e.target as HTMLInputElement).value.trim();
		if (!value) return;

		try {
			new URL(value);
			hasUrlError = false;
		} catch {
			hasUrlError = true;
			return;
		}

		onLogoChange(value);
	}
</script>

<div class="flex w-28 flex-col gap-2">
	<div
		class="relative"
		role="group"
		aria-label={m.logo()}
		ondragover={(e) => {
			e.preventDefault();
			isDraggingOver = true;
		}}
		ondragleave={() => (isDraggingOver = false)}
		ondrop={onDrop}
	>
		<FileInput
			{id}
			accept="image/*"
			onchange={onFileChange}
			onclick={(e: any) => (e.target.value = '')}
			class={cn(
				'text-muted-foreground hover:bg-muted/50 flex size-28 cursor-pointer flex-col items-center justify-center gap-1.5 overflow-hidden rounded-2xl border border-dashed text-xs transition-colors',
				logoDataURL && (light ? 'bg-[#F5F5F5]' : 'bg-[#262626]'),
				isDraggingOver && 'border-primary bg-muted/50'
			)}
		>
			{#if logoDataURL}
				<ImageBox
					class="size-full bg-transparent"
					src={logoDataURL}
					alt={m.name_logo({ name: clientName })}
				/>
				<span class="sr-only">{m.upload_logo()}</span>
			{:else}
				<LucideUpload class="size-5" />
				{m.logo()}
			{/if}
		</FileInput>
		{#if logoDataURL}
			<Button
				size="icon"
				onclick={resetLogo}
				aria-label={m.remove_logo()}
				class="absolute -top-2 -right-2 size-6 rounded-full shadow-md"
			>
				<LucideX class="size-3" />
			</Button>
		{/if}
	</div>

	<div class="flex items-center justify-between gap-1">
		{@render tabTriggers?.()}
		<Popover.Root>
			<Popover.Trigger
				class={cn(buttonVariants({ variant: 'ghost', size: 'icon-sm' }), 'text-muted-foreground')}
				aria-label={m.use_image_url()}
			>
				<LucideLink class="size-4" />
			</Popover.Trigger>
			<Popover.Content class="w-80">
				<Label for="{id}-url" class="text-xs">URL</Label>
				<Input
					id="{id}-url"
					value={url}
					oninput={(e) => (url = e.currentTarget.value)}
					onfocusout={onUrlChange}
					aria-invalid={hasUrlError}
					type="url"
				/>
				{#if hasUrlError}
					<p class="text-destructive mt-1 text-start text-xs">{m.invalid_url()}</p>
				{/if}
				<p class="text-muted-foreground mt-2 text-xs">
					<FormattedMessage message={m.logo_from_url_description} />
				</p>
			</Popover.Content>
		</Popover.Root>
	</div>
</div>
