<script lang="ts">
	import { ScrollArea as ScrollAreaPrimitive } from 'bits-ui';
	import { cn, type WithoutChild } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		viewportClass,
		children,
		...restProps
	}: WithoutChild<ScrollAreaPrimitive.RootProps> & { viewportClass?: string } = $props();
</script>

<ScrollAreaPrimitive.Root
	bind:ref
	data-slot="scroll-area"
	class={cn('relative', className)}
	{...restProps}
>
	<!-- A positioned viewport keeps absolutely positioned descendants such as sr-only labels scrolling with the content instead of overflowing the root -->
	<ScrollAreaPrimitive.Viewport
		data-slot="scroll-area-viewport"
		class={cn(
			'focus-visible:outline-ring scroll-fade-y relative size-full rounded-[inherit] outline-none focus-visible:outline-2 focus-visible:-outline-offset-2',
			viewportClass
		)}
	>
		{@render children?.()}
	</ScrollAreaPrimitive.Viewport>
	<ScrollAreaPrimitive.Scrollbar
		data-slot="scroll-area-scrollbar"
		orientation="vertical"
		class="flex h-full w-2.5 touch-none border-l border-l-transparent p-px transition-colors select-none"
	>
		<ScrollAreaPrimitive.Thumb
			data-slot="scroll-area-thumb"
			class="bg-border relative flex-1 rounded-full"
		/>
	</ScrollAreaPrimitive.Scrollbar>
</ScrollAreaPrimitive.Root>
