<script lang="ts">
	import { Command as CommandPrimitive } from 'cmdk-sv';
	import { cn } from '$lib/utils/style.js';
	import type { ClassValue } from 'svelte/elements';
	type $$Props = CommandPrimitive.GroupProps;

	let className: ClassValue | undefined | null = undefined;
	export { className as class };
</script>

<CommandPrimitive.Group
	class={cn(
		'text-foreground **:data-cmdk-group-heading:text-muted-foreground overflow-hidden p-1 **:data-cmdk-group-heading:px-2 **:data-cmdk-group-heading:py-1.5 **:data-cmdk-group-heading:text-xs **:data-cmdk-group-heading:font-medium',
		className
	)}
	{...$$restProps}
>
	<slot />
</CommandPrimitive.Group>
