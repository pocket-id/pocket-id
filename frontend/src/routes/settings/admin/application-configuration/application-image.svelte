<script lang="ts">
	import { Button, buttonVariants } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.ts';
	import { LucideTriangleAlert, LucideUpload, LucideUser } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		name,
		label,
		image = $bindable(),
		imageURL,
		accept,
		well = 'light',
		variant = 'swatch',
		fallback,
		canReset = false,
		canRemove = false,
		minWidth
	}: {
		// The image's name inside the actions' accessible names, such as "light logo"
		name: string;
		// The visible variant name beside the swatch, such as "Light"
		label?: string;
		// A file to upload, null to reset or remove the image on save, undefined while unchanged
		image: File | null | undefined;
		imageURL: string;
		accept: string;
		well?: 'light' | 'dark' | 'surface';
		variant?: 'swatch' | 'background' | 'avatar';
		// Shown while no custom image is set, such as the bundled logo
		fallback?: Snippet;
		canReset?: boolean;
		canRemove?: boolean;
		// Images narrower than this get a soft warning, except SVGs, which scale
		minWidth?: number;
	} = $props();

	let src = $state(imageURL);
	let isImageSet = $state(true);
	let tooSmallWidth = $state<number | null>(null);

	function onFileChange(e: Event & { currentTarget: HTMLInputElement }) {
		const file = e.currentTarget.files?.[0];
		e.currentTarget.value = '';
		if (!file) return;
		image = file;
		src = URL.createObjectURL(file);
		isImageSet = true;
	}

	function onImageLoad(img: HTMLImageElement) {
		const width = img.naturalWidth;
		tooSmallWidth =
			image && minWidth && image.type !== 'image/svg+xml' && width < minWidth ? width : null;
	}

	function clear() {
		image = null;
		isImageSet = false;
		tooSmallWidth = null;
	}
</script>

<div
	class={cn(
		'grid min-w-0 items-center gap-x-4 gap-y-2',
		variant === 'background' ? 'max-w-[400px] grid-cols-1' : 'grid-cols-[auto_minmax(0,1fr)]'
	)}
>
	<div
		class={cn(
			'grid place-items-center overflow-hidden *:col-start-1 *:row-start-1',
			variant === 'avatar'
				? 'bg-fill-muted text-fg-muted size-18 rounded-full'
				: 'size-22 rounded-md shadow-[inset_0_0_0_1px_var(--border-strong)]',
			variant === 'background' && 'aspect-[16/10] size-auto w-full',
			variant !== 'avatar' &&
				{
					light: 'bg-well-light text-well-light-fg',
					dark: 'bg-well-dark text-well-dark-fg',
					surface: 'bg-surface-sunken text-fg-muted'
				}[well]
		)}
	>
		{#if isImageSet}
			<img
				{src}
				alt=""
				class={cn(
					variant === 'swatch' && 'max-h-14 max-w-14 object-contain',
					variant !== 'swatch' && 'size-full object-cover'
				)}
				onload={(e) => onImageLoad(e.currentTarget as HTMLImageElement)}
				onerror={() => (isImageSet = false)}
			/>
		{:else if fallback}
			<div class="size-14">{@render fallback()}</div>
		{:else if variant === 'avatar'}
			<LucideUser class="size-7" />
		{:else}
			<span class="text-caption px-6 text-center">{m.config_no_background()}</span>
		{/if}
	</div>
	<div class="flex min-w-0 flex-col gap-2">
		{#if label}
			<p class="text-control font-semibold">{label}</p>
		{/if}
		<div class="flex flex-wrap items-center gap-2">
			<!-- The label is the file input's accessible name, and the visually hidden input keeps it reachable by keyboard -->
			<label
				class={cn(
					buttonVariants({ variant: 'secondary', size: 'sm' }),
					'has-focus-visible:outline-ring cursor-pointer has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-disabled:cursor-not-allowed has-disabled:opacity-45'
				)}
			>
				<input type="file" class="sr-only" {accept} onchange={onFileChange} />
				<LucideUpload />
				{m.config_replace()} <span class="sr-only">{name}</span>
			</label>
			{#if canReset}
				<Button
					size="sm"
					variant="ghost"
					disabled={!isImageSet}
					aria-label={m.config_reset_image({ image: name })}
					onclick={clear}
				>
					{m.reset_to_default()}
				</Button>
			{:else if canRemove}
				<Button
					size="sm"
					variant="ghost"
					disabled={!isImageSet}
					aria-label={m.config_remove_image({ image: name })}
					onclick={clear}
				>
					{m.config_remove()}
				</Button>
			{/if}
		</div>
	</div>
	{#if tooSmallWidth}
		<p
			class={cn(
				'text-caption text-fg-muted flex items-start gap-1.5',
				variant !== 'background' && 'col-start-2'
			)}
		>
			<LucideTriangleAlert class="text-warning-fg mt-[3px] size-3 shrink-0" />
			{m.config_image_too_small({ width: tooSmallWidth })}
		</p>
	{/if}
</div>
