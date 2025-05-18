<script lang="ts">
	import { Label as LabelPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';

	type $$Props = LabelPrimitive.Props;
	type $$Events = LabelPrimitive.Events;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<LabelPrimitive.Root
	class={cn(
		'mb-3 block text-sm leading-none font-medium peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
		className
	)}
	{...rest}
	on:mousedown
>
	{@render children?.()}
</LabelPrimitive.Root>
