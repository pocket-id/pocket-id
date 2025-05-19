<script lang="ts">
	import { createBubbler } from 'svelte/legacy';

	const bubble = createBubbler();
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLAttributes<HTMLTableRowElement> & {
		'data-state'?: unknown;
	};

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<tr
	class={cn('data-[state=selected]:bg-muted border-b transition-colors', className)}
	{...rest}
	onclick={bubble('click')}
	onkeydown={bubble('keydown')}
>
	{@render children?.()}
</tr>
