<script lang="ts">
	import { Separator as SeparatorPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';

	type $$Props = SeparatorPrimitive.Props;

	interface Props {
		class?: $$Props['class'];
		orientation?: $$Props['orientation'];
		decorative?: $$Props['decorative'];
		[key: string]: any
	}

	let { class: className = undefined, orientation = 'horizontal', decorative = undefined, ...rest }: Props = $props();
	
</script>

<SeparatorPrimitive.Root
	class={cn(
		'bg-border shrink-0',
		orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
		className
	)}
	{orientation}
	{decorative}
	{...rest}
/>
