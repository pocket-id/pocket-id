<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLAttributes<HTMLDivElement>;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<div
	class={cn('mt-4 flex flex-col space-y-4 sm:flex-row sm:space-y-0 sm:space-x-4', className)}
	{...rest}
>
	{@render children?.()}
</div>
