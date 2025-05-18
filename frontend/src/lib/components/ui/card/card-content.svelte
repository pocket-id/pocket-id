<script lang="ts">
	import { cn } from '$lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	type $$Props = HTMLAttributes<HTMLDivElement>;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<div class={cn('bg-muted/20 p-6 pt-5 peer-[.card-header]:border-t', className)} {...rest}>
	{@render children?.()}
</div>
