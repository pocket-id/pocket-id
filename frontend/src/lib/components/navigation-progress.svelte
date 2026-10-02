<script lang="ts">
	import { navigating } from '$app/state';
	import { fade } from 'svelte/transition';

	// Skip the fade when the bar never became visible, otherwise it would start growing during the fade and flash on fast navigations
	function fadeIfVisible(node: HTMLElement) {
		const visible = new DOMMatrixReadOnly(getComputedStyle(node).transform).a > 0;
		return fade(node, { duration: visible ? 200 : 0 });
	}
</script>

{#if navigating.to && !navigating.shallow}
	<div class="nav-progress" aria-hidden="true" out:fadeIfVisible></div>
{/if}

<style>
	.nav-progress {
		position: fixed;
		inset: 0 0 auto 0;
		z-index: 100;
		height: 2px;
		background: var(--primary);
		transform: scaleX(0);
		transform-origin: left;
		pointer-events: none;
		animation: progress 3s cubic-bezier(0.1, 0.6, 0.3, 1) 100ms forwards;
	}

	@keyframes progress {
		10% {
			transform: scaleX(0.25);
		}

		to {
			transform: scaleX(0.85);
		}
	}
</style>
