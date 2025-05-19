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

<div class={cn('card-header peer flex flex-col space-y-1.5 p-6', className)} {...rest}>
	{@render children?.()}
</div>
