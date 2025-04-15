<script lang="ts">
	import { page } from '$app/state';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Snippet } from 'svelte';

	let {
		delay = 50,
		stagger = 150,
		children
	}: {
		delay?: number;
		stagger?: number;
		children: Snippet;
	} = $props();

	let containerNode: HTMLElement;

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
					const itemDelay = delay + index * stagger;
					(child as HTMLElement).style.setProperty('animation-delay', `${itemDelay}ms`);
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

<div
	class="fade-wrapper"
	class:no-animations={$appConfigStore.disableAnimations}
	bind:this={containerNode}
>
	{@render children()}
</div>
