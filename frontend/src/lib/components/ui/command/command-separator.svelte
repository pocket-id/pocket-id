<script lang="ts">
	import { Command as CommandPrimitive } from 'cmdk-sv';
	import { cn } from '$lib/utils/style.js';
	import type { ClassValue } from 'svelte/elements';

	type $$Props = CommandPrimitive.SeparatorProps;
	interface Props {
		class?: ClassValue | undefined | null;
		[key: string]: any
	}

	let { class: className = undefined, ...rest }: Props = $props();
	
</script>

<CommandPrimitive.Separator class={cn('bg-border -mx-1 h-px', className)} {...rest} />
