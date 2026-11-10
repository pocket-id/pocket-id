<script lang="ts">
	import Logo from '#lib/components/logo.svelte';
	import { cn } from '@pocket-id/ui/utils';
	import { appearance, previewClass, previewMode, type Layout } from './appearance.svelte.ts';

	let {
		layout,
		screen = 'app',
		flush = false,
		class: className
	}: {
		// Defaults to the draft's layout, a layout picker passes each option instead
		layout?: Layout;
		screen?: 'app' | 'signin';
		// Fills a card edge to edge, so the card's own border and corners frame it
		flush?: boolean;
		class?: string;
	} = $props();

	const shown = $derived(layout ?? appearance.value.layout);
	const logoKey = $derived(previewMode.current === 'dark' ? 'logoDark' : 'logoLight');
	// Only a pending upload is drawn from its file, the Logo component already shows the saved logo or the bundled one
	const pendingLogo = $derived(appearance.images[logoKey] ? appearance.imageUrl(logoKey) : null);
	const photo = $derived(appearance.imageUrl('background'));
	let failedPhoto = $state<string | null>(null);
	const showPhoto = $derived(
		!!photo && failedPhoto !== photo && (screen === 'signin' || shown === 'backdrop')
	);
</script>

{#snippet mark()}
	{#if pendingLogo}
		<img src={pendingLogo} alt="" class="mark" />
	{:else}
		<Logo
			class="mark"
			animate={false}
			colorScheme={previewMode.current}
			defaultOnly={appearance.images[logoKey] === null}
		/>
	{/if}
{/snippet}

<!-- The outer element is the size container, so the frame inside can size its corners and lines from it -->
<div aria-hidden="true" class={cn(previewClass(), 'appearance-preview w-full', className)}>
	<div class={cn('frame', screen === 'app' && shown, flush && 'flush')}>
		{#if showPhoto}
			<img
				src={photo}
				alt=""
				class={cn('photo', previewMode.current === 'dark' && 'dim')}
				onerror={() => (failedPhoto = photo)}
			/>
		{/if}

		{#if screen === 'signin'}
			<div class={cn('signin', !showPhoto && 'plain')}>
				<div class="card">
					{@render mark()}
					<i class="bar title"></i>
					<i class="bar" style:width="70%"></i>
					<i class="button"></i>
				</div>
			</div>
		{:else}
			<div class="side">
				{@render mark()}
				<div class="navs">
					{#each [62, 48, 56, 40] as width, i (i)}
						<i class={cn('nav', i === 0 && 'active')}><u></u><b style:width="{width}%"></b></i>
					{/each}
				</div>
			</div>
			<div class="page">
				<div class="head">
					<i class="bar title"></i>
					<i class="button"></i>
				</div>
				{#each [54, 40, 48] as width, i (i)}
					<i class="row"><b style:width="{width}%"></b></i>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.appearance-preview {
		position: relative;
		container-type: inline-size;
		aspect-ratio: 16 / 10;
		user-select: none;
	}

	/* Corners scale with the preview's width, so a thumbnail keeps the proportions of a large preview */
	.frame {
		--k: var(--appearance-radius, 1);
		position: absolute;
		inset: 0;
		display: flex;
		overflow: hidden;
		border-radius: calc(2.4cqw * var(--k));
		background: var(--shell-bg);
		box-shadow: inset 0 0 0 1px var(--border);
	}

	.frame.flush {
		border-radius: 0;
		box-shadow: none;
	}

	/* The gap between the cards follows the shells' 8px on a window about 1440px wide, the edge gets a little more so the cards do not crowd the frame */
	.backdrop {
		gap: max(2px, 0.6cqw);
		padding: max(3px, 1.4cqw);
	}

	.photo {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.photo.dim {
		filter: brightness(0.55);
	}

	i,
	b,
	u {
		display: block;
	}

	.side {
		position: relative;
		display: flex;
		width: 24%;
		flex-direction: column;
		gap: 1.6cqw;
		padding: 3cqw 2.4cqw;
		background: var(--sidebar);
	}

	.classic .side {
		box-shadow: inset -1px 0 0 var(--border);
	}

	.navs {
		display: flex;
		flex-direction: column;
		gap: inherit;
	}

	/* The top bar lays the sidebar's parts out as a band, with the page's column below it */
	.top-bar {
		flex-direction: column;
	}

	.top-bar .side {
		width: auto;
		gap: 0;
		padding: 2.6cqw 12cqw 0;
		box-shadow: inset 0 -1px 0 var(--border);
	}

	.top-bar .navs {
		flex-direction: row;
		gap: 3.4cqw;
	}

	.top-bar .nav {
		width: 7cqw;
		height: 5cqw;
		padding: 0;
		border-radius: 0;
	}

	.top-bar .nav u {
		display: none;
	}

	.top-bar .nav b {
		flex: 1;
	}

	.top-bar .nav.active {
		background: none;
		box-shadow: inset 0 calc(max(1px, 0.5cqw) * -1) 0 var(--primary-control);
	}

	.top-bar .page {
		border-radius: 0;
		padding-inline: 12cqw;
	}

	.backdrop .side,
	.page {
		border-radius: calc(1.8cqw * var(--k));
	}

	.backdrop .side {
		background: color-mix(in srgb, var(--sidebar) 60%, transparent);
		backdrop-filter: blur(3cqw) saturate(1.5);
	}

	/* A custom logo can be wide, so it keeps its height and grows sideways up to the sidebar's width */
	.frame :global(.mark) {
		width: auto;
		max-width: 100%;
		height: 4cqw;
		margin-bottom: 1.6cqw;
		align-self: flex-start;
		color: var(--fg);
		object-fit: contain;
	}

	.nav {
		display: flex;
		align-items: center;
		gap: 1cqw;
		height: 3.6cqw;
		padding: 0 1.2cqw;
		border-radius: calc(1cqw * var(--k));
	}

	.nav.active {
		background: var(--fill-selected);
	}

	/* The nav icon, which takes the primary color while its item is active */
	.nav u {
		width: 1.6cqw;
		height: 1.6cqw;
		flex-shrink: 0;
		border-radius: 0.5cqw;
		background: var(--fg-muted);
		opacity: 0.5;
	}

	.nav.active u {
		background: var(--primary-control);
		opacity: 1;
	}

	.nav b,
	.row b,
	.bar {
		height: 1.2cqw;
		border-radius: 1cqw;
		background: var(--fg-muted);
		opacity: 0.45;
	}

	.page {
		position: relative;
		display: flex;
		min-width: 0;
		flex: 1;
		flex-direction: column;
		padding: 4.8cqw 5cqw;
		background: var(--surface);
	}

	.classic .page {
		border-radius: 0;
	}

	.backdrop .page {
		box-shadow: 0 2cqw 4cqw -1.5cqw rgb(0 0 0 / 0.35);
	}

	.head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 3cqw;
	}

	.bar.title {
		width: 30%;
		height: 2.6cqw;
		background: var(--fg);
		opacity: 0.85;
	}

	.button {
		width: 13cqw;
		height: 3.8cqw;
		border-radius: calc(1cqw * var(--k));
		background: var(--primary);
	}

	.row {
		display: flex;
		align-items: center;
		height: 6cqw;
		box-shadow: inset 0 -1px 0 var(--border);
	}

	.signin {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
	}

	.signin.plain {
		background: var(--sidebar);
	}

	.card {
		display: flex;
		width: 38%;
		flex-direction: column;
		gap: 2cqw;
		padding: 4cqw 3.6cqw;
		border-radius: calc(2.4cqw * var(--k));
		background: var(--surface);
		box-shadow: 0 2cqw 4cqw -1.5cqw rgb(0 0 0 / 0.35);
	}

	.card :global(.mark) {
		margin-bottom: 0;
	}

	.card .bar.title {
		width: 64%;
	}

	.card .button {
		width: 100%;
		margin-top: 1cqw;
	}
</style>
