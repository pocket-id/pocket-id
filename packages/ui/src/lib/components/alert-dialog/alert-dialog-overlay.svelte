<script lang="ts">
	import { AlertDialog as AlertDialogPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: AlertDialogPrimitive.OverlayProps = $props();
</script>

<AlertDialogPrimitive.Overlay
	bind:ref
	data-slot="alert-dialog-overlay"
	class={cn(
		'bg-scrim transition-motion fixed inset-0 isolate z-(--z-overlay) backdrop-blur-[4px] duration-(--dur-slow) data-closed:opacity-0 data-closed:duration-(--dur-base) data-closed:ease-in starting:opacity-0 [@media(prefers-reduced-transparency:reduce)]:backdrop-filter-none!',
		className
	)}
	{...restProps}
/>
