<script lang="ts">
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import appConfigStore from '$lib/stores/application-configuration-store';

	let {
		delay = 50,
		stagger = 150,
		animationsDisabled = false, // Start with a default value
		children
	}: {
		delay?: number;
		stagger?: number;
		animationsDisabled?: boolean;
		children: Snippet;
	} = $props();

	let containerNode: HTMLElement;

	// Pull the animations disabled value from the store
	$effect(() => {
		// Check if the store value exists and use it
		if ($appConfigStore && $appConfigStore.disableAnimations !== undefined) {
			console.log('Animations disabled:', $appConfigStore.disableAnimations);
			animationsDisabled = $appConfigStore.disableAnimations;
			applyAnimationDelays();
		}
	});

	$effect(() => {
		page.route;
		applyAnimationDelays();
	});

	function applyAnimationDelays() {
		if (containerNode) {
			const childNodes = Array.from(containerNode.children);
			childNodes.forEach((child, index) => {
				// Skip comment nodes and text nodes
				if (child.nodeType === 1) {
					if (animationsDisabled) {
						// Remove animation-delay if animations are disabled
						(child as HTMLElement).style.removeProperty('animation-delay');
					} else {
						const itemDelay = delay + index * stagger;
						(child as HTMLElement).style.setProperty('animation-delay', `${itemDelay}ms`);
					}
				}
			});
		}
	}
</script>

<svelte:head>
	<style>
		/* Base styles */
		.fade-wrapper {
			display: contents;
			overflow: hidden;
		}

		/* Apply these styles to all children */
		.fade-wrapper > *:not(.no-fade) {
			animation-fill-mode: both;
			opacity: 0;
			transform: translateY(10px);
			animation-delay: calc(var(--animation-delay, 0ms) + 0.1s);
			animation: fadeIn 0.8s ease-out forwards;
			will-change: opacity, transform;
		}

		/* Disable animations completely - add locally to ensure it works */
		.fade-wrapper.no-animations > * {
			animation: none !important;
			opacity: 1 !important;
			transform: none !important;
		}
	</style>
</svelte:head>

<div class="fade-wrapper" class:no-animations={animationsDisabled} bind:this={containerNode}>
	{@render children()}
</div>
