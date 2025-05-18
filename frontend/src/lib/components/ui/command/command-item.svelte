<script lang="ts">
	import { Command as CommandPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';
	import type { ClassValue } from 'svelte/elements';

	type $$Props = CommandPrimitive.ItemProps;

	interface Props {
		asChild?: boolean;
		class?: ClassValue | undefined | null;
		children?: import('svelte').Snippet<[any]>;
		[key: string]: any;
	}

	let { asChild = false, class: className = undefined, children, ...rest }: Props = $props();

	const children_render = $derived(children);
</script>

<CommandPrimitive.Item
	{asChild}
	class={cn(
		'aria-selected:bg-accent aria-selected:text-accent-foreground data-disabled:pointer-events-none data-disabled:opacity-50 relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none',
		className
	)}
	{...rest}
>
	{#snippet children({ action, attrs })}
		{@render children_render?.({ action, attrs })}
	{/snippet}
</CommandPrimitive.Item>
