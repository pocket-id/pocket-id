<script lang="ts">
	import { Command as CommandPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';
	import type { ClassValue } from 'svelte/elements';

	type $$Props = CommandPrimitive.EmptyProps;
	interface Props {
		class?: ClassValue | undefined | null;
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<CommandPrimitive.Empty class={cn('py-6 text-center text-sm', className)} {...rest}>
	{@render children?.()}
</CommandPrimitive.Empty>
