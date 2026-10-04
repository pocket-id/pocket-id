<script lang="ts">
	import Logo from '#lib/components/logo.svelte';
	import CheckmarkAnimated from '#lib/icons/checkmark-animated.svelte';
	import ConnectArrow from '#lib/icons/connect-arrow.svelte';
	import CrossAnimated from '#lib/icons/cross-animated.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { OidcClientMetaData } from '#lib/types/oidc.type.ts';
	import { cachedOidcClientLogo } from '#lib/utils/cached-image-util.ts';
	import { mode } from 'mode-watcher';
	import { untrack } from 'svelte';
	import { cubicIn, cubicOut } from 'svelte/easing';
	import { prefersReducedMotion, Tween } from 'svelte/motion';

	type Outcome = 'success' | 'error';

	const {
		success,
		error,
		client
	}: {
		success?: boolean;
		error?: boolean;
		client?: OidcClientMetaData;
	} = $props();

	// Each tile travels this far so they meet in the middle
	const TRAVEL = 108;
	// The tiles start 152px apart, so they touch once each has covered this share of its travel
	const CONTACT = 152 / (2 * TRAVEL);
	// Kept below the previous 500ms slide so the animation never delays the redirect
	const PULL_MS = 350;

	const SQUASH = [
		{ transform: 'scale(1.18, .84)' },
		{ transform: 'scale(.94, 1.06)', offset: 0.45 },
		{ transform: 'scale(1)' }
	];
	const SHAKE = [
		{ transform: 'scale(1.18, .84)' },
		{ transform: 'scale(.94, 1.06)', offset: 0.2 },
		{ transform: 'scale(1)', offset: 0.3 },
		{ transform: 'translateX(-7px)', offset: 0.44 },
		{ transform: 'translateX(6px)', offset: 0.58 },
		{ transform: 'translateX(-4px)', offset: 0.72 },
		{ transform: 'translateX(2px)', offset: 0.86 },
		{ transform: 'translateX(0)' }
	];

	const isLightMode = $derived(mode.current === 'light');

	// The tiles accelerate into each other while the arrow snaps taut during the first third of the pull
	const pull = new Tween(0, { duration: PULL_MS });
	const travelled = $derived(cubicIn(pull.current));
	const taut = $derived(cubicOut(Math.min(1, pull.current / 0.32)));

	let result = $state<Outcome | null>(null);
	let animated = $state(false);
	let resultTile = $state<HTMLDivElement>();
	let hasMounted = false;

	$effect(() => {
		const outcome = success ? 'success' : error ? 'error' : null;
		untrack(() => show(outcome));
	});

	async function show(outcome: Outcome | null) {
		// An outcome that is already set when the page loads is shown without replaying the animation
		const animate =
			hasMounted && !$appConfigStore.disableAnimations && !prefersReducedMotion.current;
		hasMounted = true;

		resultTile?.getAnimations().forEach((animation) => animation.cancel());
		result = null;

		if (!outcome) {
			pull.set(0, { duration: 0 });
			return;
		}

		// The pull only starts from rest, so an outcome that arrives mid-pull jumps straight to the merged tile
		// Any newer outcome interrupts the pull, and an interrupted tween never resolves, so a stale outcome is never shown
		if (animate && pull.current === 0) await pull.set(1);
		else pull.set(1, { duration: 0 });

		animated = animate;
		result = outcome;
		if (animate) {
			resultTile?.animate(outcome === 'success' ? SQUASH : SHAKE, {
				duration: outcome === 'success' ? 380 : 620,
				easing: 'ease-out'
			});
		}
	}
</script>

<div class="flex justify-center gap-3">
	<div
		class="bg-muted rounded-2xl p-3"
		style:transform="translateX({TRAVEL * travelled}px) scale({1 - 0.15 * travelled})"
		style:opacity={Math.min(1, (1 - travelled) / 0.15)}
	>
		<Logo class="size-10" animate={false} />
	</div>

	<ConnectArrow
		class="w-32 {travelled >= CONTACT ? 'invisible' : ''}"
		{taut}
		squeeze={Math.min(1, travelled / CONTACT)}
	/>

	<div style:transform="translateX({-TRAVEL * travelled}px)">
		<div
			bind:this={resultTile}
			class="bg-muted relative overflow-hidden rounded-2xl p-3"
			class:animated
		>
			<!-- The outcome colour floods in from the edge where the tiles collided -->
			<div
				class="fill absolute inset-0 {result === 'error' ? 'bg-red-200' : 'bg-green-200'}"
				class:flooded={result}
			></div>

			<div class="client relative size-10" class:faded={result}>
				{#if client?.hasLogo || client?.hasDarkLogo}
					<img
						class="aspect-square size-10 object-contain"
						src={cachedOidcClientLogo.getUrl(client.id, isLightMode)}
						draggable={false}
						alt={m.client_logo()}
					/>
				{:else if client?.name}
					<div class="flex size-10 items-center justify-center text-3xl font-bold">
						{client.name.charAt(0).toUpperCase()}
					</div>
				{/if}
			</div>

			{#if result}
				<div class="absolute inset-0 flex items-center justify-center">
					{#if result === 'success'}
						<CheckmarkAnimated class="size-7" />
					{:else}
						<CrossAnimated class="size-5" />
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>

<style>
	.fill {
		clip-path: circle(0% at 0% 50%);
	}

	.fill.flooded {
		clip-path: circle(150% at 0% 50%);
	}

	.client.faded {
		opacity: 0;
		scale: 0.6;
	}

	/* Only an outcome reached through the animation transitions in, one present on page load appears at once */
	.animated .fill.flooded {
		transition: clip-path 340ms cubic-bezier(0.3, 0, 0.2, 1);
	}

	.animated .client {
		transition:
			opacity 160ms 40ms,
			scale 160ms 40ms;
	}
</style>
