<script lang="ts">
	import Logo from '#lib/components/logo.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import ConnectArrow from '#lib/icons/connect-arrow.svelte';
	import type { InteractionClient } from '#lib/types/oidc.type.ts';
	import { untrack } from 'svelte';
	import { cubicIn, cubicOut } from 'svelte/easing';
	import { prefersReducedMotion, Tween } from 'svelte/motion';
	import CheckIcon from '~icons/solar/check-bold';
	import CrossIcon from '~icons/solar/close-bold';

	type Outcome = 'success' | 'error';

	const {
		success,
		error,
		client,
		onsuccessshown
	}: {
		success?: boolean;
		error?: boolean;
		client?: InteractionClient;
		onsuccessshown?: () => void;
	} = $props();

	// The 56px tiles start this far apart, with the 100px arrow and 12px on each side of it
	const GAP = 124;
	// The client tile travels onto the operator tile, so the merged tile sits on the stack's left edge like every other auth tile
	const TRAVEL = 56 + GAP;
	// The tiles touch once the client tile has covered this share of its travel
	const CONTACT = GAP / TRAVEL;
	// Kept short so the animation never delays the redirect
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

	// The client tile accelerates into the operator tile while the arrow snaps taut during the first third of the pull
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
		const animate = hasMounted && !prefersReducedMotion.current;
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
			try {
				await resultTile?.animate(outcome === 'success' ? SQUASH : SHAKE, {
					duration: outcome === 'success' ? 380 : 620,
					easing: 'ease-out'
				}).finished;
			} catch {
				// A newer outcome cancelled the animation, so this one never counts as shown
				return;
			}
		}
		if (outcome === 'success') onsuccessshown?.();
	}
</script>

<div class="mb-6 flex items-center gap-3" aria-hidden="true">
	<div
		class="bg-surface-raised inset-ring-border grid size-14 shrink-0 place-items-center rounded-md inset-ring"
		style:transform="scale({1 - 0.15 * travelled})"
		style:opacity={Math.min(1, (1 - travelled) / 0.15)}
	>
		<Logo class="size-8" animate={false} />
	</div>

	<ConnectArrow
		class="w-25 shrink-0 {travelled >= CONTACT ? 'invisible' : ''}"
		{taut}
		squeeze={Math.min(1, travelled / CONTACT)}
	/>

	<div class="shrink-0" style:transform="translateX({-TRAVEL * travelled}px)">
		<div bind:this={resultTile} class="relative isolate size-14 rounded-md" class:animated>
			<div class="client" class:faded={result}>
				{#if client}
					<OidcClientAvatar
						id={client.id}
						name={client.name}
						hasLogo={client.hasLogo || client.hasDarkLogo}
						size={56}
					/>
				{:else}
					<div class="bg-fill-muted inset-ring-border size-14 rounded-md inset-ring"></div>
				{/if}
			</div>

			<!-- The outcome color floods in from the edge where the tiles collided -->
			<!-- The tile itself does not clip, because clipping at its own radius eats into the client tile's 1px ring on fractional positions -->
			<div
				class="fill absolute inset-0 rounded-[inherit] {result === 'error'
					? 'bg-danger-soft'
					: 'bg-success-soft'}"
				class:flooded={result}
			></div>

			{#if result}
				<div class="glyph absolute inset-0 grid place-items-center">
					{#if result === 'success'}
						<CheckIcon class="text-success-solid size-7" />
					{:else}
						<CrossIcon class="text-danger-solid size-5.5" />
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

	/* The icon pops in once the flood has passed the middle of the tile */
	.animated .glyph {
		animation: glyph-in var(--dur-base) var(--ease-out) 120ms backwards;
	}

	@keyframes glyph-in {
		from {
			opacity: 0;
			scale: 0.6;
		}
	}
</style>
