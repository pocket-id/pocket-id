<script lang="ts">
	import { cachedBackgroundImage } from '#lib/utils/cached-asset-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { PersistedState } from 'runed';
	import PhoneNav from '../phone-nav.svelte';
	import ClassicSidebar from './classic-sidebar.svelte';
	import type { ShellProps } from './registry.ts';

	let { groups, versionInformation, showUpdate, children }: ShellProps = $props();

	const collapsed = new PersistedState('sidebar-collapsed', false);
	let loaded = $state(false);
	let failed = $state(false);
</script>

<!-- The isolated stacking context keeps the fixed photo above the body background but below the cards -->
<div class="backdrop relative isolate flex min-h-dvh gap-2 sm:p-2">
	{#if !failed}
		<img
			src={cachedBackgroundImage.getUrl()}
			alt=""
			class={cn(
				'pointer-events-none fixed inset-0 -z-10 hidden size-full object-cover opacity-0 transition-opacity duration-(--dur-slow) ease-in-out sm:block dark:brightness-[0.55]',
				loaded && 'opacity-100'
			)}
			onload={() => (loaded = true)}
			onerror={() => (failed = true)}
		/>
	{/if}
	<ClassicSidebar
		{groups}
		{versionInformation}
		{showUpdate}
		bind:collapsed={collapsed.current}
		class="bg-sidebar/60 top-2 h-[calc(100dvh-1rem)] rounded-2xl backdrop-blur-[34px] backdrop-saturate-150"
	/>
	<div class="bg-surface flex min-w-0 flex-1 flex-col sm:rounded-2xl sm:shadow-(--card-shadow)">
		<PhoneNav
			{groups}
			version="Pocket ID {versionInformation.currentVersion}"
			newestVersion={showUpdate ? versionInformation.newestVersion : null}
		/>
		{@render children()}
	</div>
</div>

<style>
	/* The body carries the shell color so the overscroll bounce reveals it instead of the canvas */
	:global(body:has(.backdrop)) {
		background: var(--shell-bg);
	}

	.backdrop {
		--card-shadow: 0 0 0 1px rgb(0 0 0 / 0.06), 0 24px 48px -12px rgb(0 0 0 / 0.25);
		--glass-shadow: 0 0 0 1px rgb(255 255 255 / 0.5), 0 16px 40px -16px rgb(0 0 0 / 0.2);
	}

	:global(.dark) .backdrop {
		--card-shadow: 0 0 0 1px rgb(255 255 255 / 0.08), 0 24px 48px -12px #000000;
		--glass-shadow: 0 0 0 1px rgb(255 255 255 / 0.08), 0 16px 40px -16px #000000;
	}

	/* Outranks the sidebar's hairline, and the fills turn translucent so the glass shows through them */
	/* Muted text and icons move towards the foreground because the photo behind the glass eats their contrast */
	.backdrop :global([data-slot='app-sidebar']) {
		--fg-muted: color-mix(in oklab, var(--fg) 75%, var(--sidebar));
		--fill-hover: rgb(0 0 0 / 0.05);
		--fill-muted: rgb(0 0 0 / 0.05);
		--fill-selected: rgb(0 0 0 / 0.08);
		--border: rgb(0 0 0 / 0.08);
		box-shadow: var(--glass-shadow);
	}

	:global(.dark) .backdrop :global([data-slot='app-sidebar']) {
		--fill-hover: rgb(255 255 255 / 0.07);
		--fill-muted: rgb(255 255 255 / 0.07);
		--fill-selected: rgb(255 255 255 / 0.11);
		--border: rgb(255 255 255 / 0.08);
	}
</style>
