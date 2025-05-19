<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLAttributes<HTMLTableSectionElement>;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<tfoot class={cn('bg-primary text-primary-foreground font-medium', className)} {...rest}>
	{@render children?.()}
</tfoot>
