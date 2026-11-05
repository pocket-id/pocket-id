<script lang="ts">
	import AppearancePreview from '#lib/components/appearance/appearance-preview.svelte';
	import {
		previewMode,
		appearance,
		brandImages
	} from '#lib/components/appearance/appearance.svelte.ts';
	import Logo from '#lib/components/logo.svelte';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import CloseIcon from '~icons/solar/close-linear';
	import type { Snippet } from 'svelte';
	import ImageTile from './image-tile.svelte';
	import PreviewSection from './preview-section.svelte';

	const appName = $derived($appConfigStore.appName);

	// A saved image may not exist, so a preview that fails to load shows its fallback instead
	let broken = $state<Record<string, boolean>>({});
</script>

{#snippet image(url: string | null, className: string, fallback: Snippet)}
	{#if url && !broken[url]}
		<img src={url} alt="" class={className} onerror={() => (broken[url] = true)} />
	{:else}
		{@render fallback()}
	{/if}
{/snippet}

{#snippet signinPreview()}
	<AppearancePreview screen="signin" />
{/snippet}

{#snippet smallPreview()}
	<div class="flex w-full max-w-md flex-col gap-4">
		<div class="border-border bg-surface overflow-hidden rounded-lg border">
			<div class="bg-fill-muted flex h-10 items-end px-2">
				<span
					class="bg-surface text-control text-fg flex h-8 w-52 items-center gap-2 rounded-t-md px-3"
				>
					{@render image(appearance.imageUrl('favicon'), 'size-4 object-contain', faviconFallback)}
					<span class="min-w-0 flex-1 truncate">{appName}</span>
					<CloseIcon class="text-fg-muted size-4" />
				</span>
			</div>
			<div class="px-2 py-2">
				<span
					class="bg-fill-muted text-caption text-fg-muted flex h-7 items-center rounded-sm px-3"
				>
					{location.host}/settings
				</span>
			</div>
		</div>

		<!-- Emails always render on white, whatever mode the app is in -->
		<div
			class="flex flex-col gap-3 rounded-lg bg-white p-5 text-black/60 shadow-[inset_0_0_0_1px_rgb(0_0_0/0.08)]"
		>
			{@render image(
				appearance.imageUrl('emailLogo'),
				'h-8 max-w-32 self-start object-contain',
				emailFallback
			)}
			<span class="text-heading text-black">Your sign-in code</span>
			<span class="text-caption"
				>Enter this code to sign in to {appName}. It expires in 15 minutes.</span
			>
			<span
				class="w-fit rounded-sm bg-black/5 px-3 py-1.5 font-mono text-[17px] tracking-[0.2em] text-black"
			>
				482 913
			</span>
		</div>

		<div class="bg-surface border-border flex items-center gap-3 rounded-lg border px-4 py-3">
			<span
				class="bg-fill-muted text-fg-muted text-caption grid size-9 shrink-0 place-items-center overflow-hidden rounded-full font-medium"
			>
				{@render image(
					appearance.imageUrl('defaultProfilePicture'),
					'size-full object-cover',
					initials
				)}
			</span>
			<span class="flex min-w-0 flex-col">
				<span class="text-control text-fg font-medium">Alex Morgan</span>
				<span class="text-caption text-fg-muted truncate">alex@example.com</span>
			</span>
		</div>
	</div>
{/snippet}

{#snippet brandingPreview()}
	<div class="flex w-full flex-col gap-6">
		{@render signinPreview()}
		{@render smallPreview()}
	</div>
{/snippet}

{#key appearance.imagesKey}
	<PreviewSection title="Branding" preview={brandingPreview}>
		<fieldset class="min-w-0" disabled={$appConfigStore.uiConfigDisabled}>
			<div class="grid grid-cols-2 gap-x-3 gap-y-5 sm:grid-cols-3">
				{#each brandImages as image (image.key)}
					<ImageTile {image} disabled={$appConfigStore.uiConfigDisabled} />
				{/each}
			</div>
		</fieldset>
	</PreviewSection>
{/key}

{#snippet faviconFallback()}
	<Logo class="size-4" animate={false} colorScheme={previewMode.current} />
{/snippet}

{#snippet emailFallback()}
	<span class="text-heading text-black">{appName}</span>
{/snippet}

{#snippet initials()}
	AM
{/snippet}
