<script lang="ts">
	import { Command as CommandPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';
	import type { ClassValue } from 'svelte/elements';

	type $$Props = CommandPrimitive.ListProps;
	interface Props {
		class?: ClassValue | undefined | null;
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<CommandPrimitive.List
	class={cn('max-h-[300px] overflow-y-auto overflow-x-hidden', className)}
	{...rest}
>
	{@render children?.()}
</CommandPrimitive.List>
