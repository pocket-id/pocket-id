<script lang="ts">
	import type { HTMLTableAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLTableAttributes;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<div class="relative w-full overflow-auto">
	<table class={cn('w-full table-auto caption-bottom text-sm', className)} {...rest}>
		{@render children?.()}
	</table>
</div>
