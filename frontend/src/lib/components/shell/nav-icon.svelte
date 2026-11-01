<script lang="ts">
	import { cn } from '#lib/utils/style.ts';
	import type { IconProps } from '@jis3r/icons';
	import type { Component } from 'svelte';
	import { prefersReducedMotion } from 'svelte/motion';

	let {
		icon: Icon,
		active,
		class: className
	}: { icon: Component<IconProps>; active: boolean; class?: string } = $props();

	// Reduced motion and Disable animations keep the icon still; the class is read on every hover because the setting can change at runtime
	const animate = $derived(
		active &&
			!prefersReducedMotion.current &&
			!document.documentElement.classList.contains('no-motion')
	);
</script>

<!-- The library names each icon with role="img" in English, which would leak into the row's accessible name, and plays on its own hover, which ignores reduced motion -->
<span
	aria-hidden="true"
	class={cn(
		'pointer-events-none inline-grid [&_svg_*]:[vector-effect:non-scaling-stroke]',
		className
	)}
>
	<Icon size={17} strokeWidth={1.5} {animate} />
</span>
