<script lang="ts">
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';

	let {
		delay = 0,
		stagger = 50,
		children
	}: {
		delay?: number;
		stagger?: number;
		children: Snippet;
	} = $props();

	let containerNode: HTMLElement;

	// Get all direct children of the container
	function getChildren(node: HTMLElement | null): HTMLElement[] {
		let allChildren: HTMLElement[] = [];
		if (node) {
			const childNodes = Array.from(node.children);
			childNodes.forEach((child) => {
				// Skip comment nodes and text nodes
				if (child.nodeType === 1) {
					// Element node
					allChildren.push(child as HTMLElement);
				}
			});
		}
		return allChildren;
	}

	// Apply initial styles and animation delays
	function applyAnimationDelays(node: HTMLElement) {
		const children = getChildren(node);

		children.forEach((el, index) => {
			const itemDelay = delay + index * stagger;
			el.style.setProperty('animation-delay', `${itemDelay}ms`);
		});

		return {};
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
		.fade-wrapper > * {
			animation-fill-mode: both;
			opacity: 0;
			transform: translateY(10px);
			animation-delay: calc(var(--animation-delay, 0ms) + 0.1s);
			animation: fadeIn 0.8s ease-out forwards;
			will-change: opacity, transform;
		}
	</style>
</svelte:head>

<div class="fade-wrapper" bind:this={containerNode} use:applyAnimationDelays>
	{@render children()}
</div>
