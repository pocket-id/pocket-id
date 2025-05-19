<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui-old';
	import { fade } from 'svelte/transition';
	import { cn } from '$lib/utils/style.js';

	type $$Props = DialogPrimitive.OverlayProps;

	interface Props {
		class?: $$Props['class'];
		transition?: $$Props['transition'];
		transitionConfig?: $$Props['transitionConfig'];
		[key: string]: any;
	}

	let {
		class: className = undefined,
		transition = fade,
		transitionConfig = {
			duration: 150
		},
		...rest
	}: Props = $props();
</script>

<DialogPrimitive.Overlay
	{transition}
	{transitionConfig}
	class={cn('bg-background/80 fixed inset-0 z-50 backdrop-blur-sm', className)}
	{...rest}
/>
