<script lang="ts">
	import { appearance, brandImages } from '#lib/components/appearance/appearance.svelte.ts';
	import Logo from '#lib/components/logo.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import * as DropdownMenu from '@pocket-id/ui/dropdown-menu';
	import { cn } from '@pocket-id/ui/utils';
	import GalleryIcon from '~icons/solar/gallery-bold-duotone';
	import DotsIcon from '~icons/solar/menu-dots-bold';
	import RestartIcon from '~icons/solar/restart-bold';
	import TrashIcon from '~icons/solar/trash-bin-minimalistic-bold';
	import UploadIcon from '~icons/solar/upload-minimalistic-bold';
	import UserIcon from '~icons/solar/user-bold-duotone';

	let { image, disabled = false }: { image: (typeof brandImages)[number]; disabled?: boolean } =
		$props();

	let input = $state<HTMLInputElement>();
	let failed = $state<string | null>(null);
	// The width of a pending upload that is too narrow, so its warning can name it
	let width = $state<number | null>(null);

	const url = $derived(appearance.imageUrl(image.key));
	const pending = $derived(appearance.images[image.key]);
	const name = $derived(image.label ? `${image.title()} · ${image.label()}` : image.title());

	function onFileChange(e: Event & { currentTarget: HTMLInputElement }) {
		const file = e.currentTarget.files?.[0];
		e.currentTarget.value = '';
		if (file) appearance.setImage(image.key, file);
	}

	// The size hint only shows when an uploaded file is too narrow, SVGs scale so they never are
	function onLoad(e: Event) {
		const natural = (e.currentTarget as HTMLImageElement).naturalWidth;
		width =
			pending && image.minWidth && pending.type !== 'image/svg+xml' && natural < image.minWidth
				? natural
				: null;
	}
</script>

<div class="flex min-w-0 flex-col gap-2">
	<div
		title={image.spec()}
		class={cn(
			'border-border relative grid aspect-[4/3] place-items-center overflow-hidden rounded-md border',
			image.well === 'dark' ? 'bg-well-dark' : 'bg-well-light'
		)}
	>
		{#if url && failed !== url}
			<img
				src={url}
				alt=""
				onload={onLoad}
				onerror={() => (failed = url)}
				class={cn(
					image.variant === 'background'
						? 'size-full object-cover'
						: image.variant === 'avatar'
							? 'size-16 rounded-full object-cover'
							: 'max-h-12 max-w-[60%] object-contain'
				)}
			/>
		{:else if image.key === 'logoLight' || image.key === 'logoDark'}
			<Logo
				defaultOnly
				animate={false}
				colorScheme={image.key === 'logoDark' ? 'dark' : 'light'}
				class="size-12"
			/>
		{:else if image.variant === 'avatar'}
			<span class="bg-fill-muted text-fg-muted grid size-16 place-items-center rounded-full">
				<UserIcon class="size-7" />
			</span>
		{:else}
			<GalleryIcon class="text-fg-subtle size-8" />
		{/if}
	</div>

	<div class="flex items-start gap-2">
		<span class="text-control text-fg line-clamp-2 min-w-0 flex-1 pt-0.5 font-medium">{name}</span>
		{#if pending !== undefined}
			<span class="text-caption text-fg-muted"
				>{pending ? 'New' : image.canReset ? 'Default' : 'Removed'}</span
			>
		{/if}
		<DropdownMenu.Root>
			<DropdownMenu.Trigger
				{disabled}
				aria-label="Change {image.name()}"
				class="text-fg-muted hover:text-fg hover:bg-fill-hover focus-visible:outline-ring grid size-7 place-items-center rounded-sm transition-colors focus-visible:outline-2 disabled:pointer-events-none disabled:opacity-45"
			>
				<DotsIcon class="size-4" />
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align="end">
				<DropdownMenu.Item onSelect={() => input?.click()}>
					<UploadIcon />
					{m.upload()}
				</DropdownMenu.Item>
				{#if image.canReset && pending !== null}
					<DropdownMenu.Item onSelect={() => appearance.setImage(image.key, null)}>
						<RestartIcon />
						{m.reset_to_default()}
					</DropdownMenu.Item>
				{/if}
				{#if image.canRemove && pending !== null}
					<DropdownMenu.Item onSelect={() => appearance.setImage(image.key, null)}>
						<TrashIcon />
						{m.config_remove()}
					</DropdownMenu.Item>
				{/if}
				{#if pending !== undefined}
					<DropdownMenu.Item onSelect={() => appearance.setImage(image.key, undefined)}>
						<RestartIcon />
						Undo
					</DropdownMenu.Item>
				{/if}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
		<input
			bind:this={input}
			type="file"
			aria-label="Replace {image.name()}"
			class="hidden"
			accept={image.accept}
			onchange={onFileChange}
		/>
	</div>

	{#if width !== null}
		<p class="text-caption text-warning-fg">{m.config_image_too_small({ width })}</p>
	{/if}
</div>
