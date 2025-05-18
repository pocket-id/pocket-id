<script lang="ts">
	import { Command as CommandPrimitive } from 'cmdk-sv';
	import { cn } from '$lib/utils/style.js';
	import type { ClassValue } from 'svelte/elements';
	type $$Props = CommandPrimitive.GroupProps;

	interface Props {
		class?: ClassValue | undefined | null;
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<CommandPrimitive.Group
	class={cn(
		'text-foreground **:data-cmdk-group-heading:text-muted-foreground overflow-hidden p-1 **:data-cmdk-group-heading:px-2 **:data-cmdk-group-heading:py-1.5 **:data-cmdk-group-heading:text-xs **:data-cmdk-group-heading:font-medium',
		className
	)}
	{...rest}
>
	{@render children?.()}
</CommandPrimitive.Group>
