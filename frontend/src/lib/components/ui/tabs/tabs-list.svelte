<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = TabsPrimitive.ListProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<TabsPrimitive.List
	class={cn(
		'bg-muted text-muted-foreground inline-flex h-10 items-center justify-center rounded-md p-1',
		className
	)}
	{...rest}
>
	{@render children?.()}
</TabsPrimitive.List>
