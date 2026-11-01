<script lang="ts">
	import { navigating } from '$app/state';
	import { appReveal } from '#lib/stores/app-reveal-store.svelte.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import { cubicOut } from 'svelte/easing';

	// Run the bar to the end in the first half and fade it out in the second, so a finished navigation reads as complete
	// Skip it when the bar never became visible, otherwise it would start growing during the fade and flash on fast navigations
	function finish(node: HTMLElement) {
		const from = new DOMMatrixReadOnly(getComputedStyle(node).transform).a;
		return {
			duration: from > 0 ? cssMs('--dur-slow') : 0,
			easing: cubicOut,
			css: (t: number, u: number) =>
				`transform: scaleX(${from + (1 - from) * Math.min(1, u * 2)}); opacity: ${Math.min(1, t * 2)}`
		};
	}
</script>

<!-- The app has already loaded when the sign-up card starts unfolding, so the bar finishes then instead of running over the unfold -->
{#if navigating.to && !navigating.shallow && !appReveal.pending}
	<div class="nav-progress" aria-hidden="true" out:finish></div>
{/if}

<style>
	.nav-progress {
		position: fixed;
		inset: 0 0 auto 0;
		z-index: var(--z-progress);
		height: 2px;
		background: linear-gradient(to right, transparent, var(--primary));
		transform: scaleX(0);
		transform-origin: left;
		pointer-events: none;
		animation: progress var(--t-progress) cubic-bezier(0.1, 0.6, 0.3, 1) 100ms forwards;
	}

	@keyframes progress {
		10% {
			transform: scaleX(0.25);
		}

		to {
			transform: scaleX(0.9);
		}
	}
</style>
