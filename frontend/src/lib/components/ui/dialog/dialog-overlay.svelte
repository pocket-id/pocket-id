<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DialogPrimitive.OverlayProps = $props();
</script>

<DialogPrimitive.Overlay
	bind:ref
	data-slot="dialog-overlay"
	class={cn(
		'bg-scrim transition-motion fixed inset-0 isolate z-(--z-overlay) backdrop-blur-[4px] duration-(--dur-slow) data-closed:opacity-0 data-closed:duration-(--dur-base) data-closed:ease-in starting:opacity-0 [@media(prefers-reduced-transparency:reduce)]:backdrop-filter-none!',
		className
	)}
	{...restProps}
/>
