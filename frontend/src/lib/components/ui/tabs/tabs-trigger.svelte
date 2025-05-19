<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = TabsPrimitive.TriggerProps;
	type $$Events = TabsPrimitive.TriggerEvents;

	interface Props {
		class?: $$Props['class'];
		value: $$Props['value'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, value, children, ...rest }: Props = $props();
</script>

<TabsPrimitive.Trigger
	class={cn(
		'ring-offset-background focus-visible:ring-ring data-[state=active]:bg-background data-[state=active]:text-foreground inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:shadow-sm',
		className
	)}
	{value}
	{...rest}
	on:click
>
	{@render children?.()}
</TabsPrimitive.Trigger>
