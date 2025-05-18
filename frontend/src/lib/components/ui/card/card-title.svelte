<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import type { HeadingLevel } from './index.js';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLAttributes<HTMLHeadingElement> & {
		tag?: HeadingLevel;
	};

	interface Props {
		class?: $$Props['class'];
		tag?: $$Props['tag'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, tag = 'h3', children, ...rest }: Props = $props();
	
</script>

<svelte:element
	this={tag}
	class={cn('flex items-center gap-2 text-xl leading-none font-semibold tracking-tight', className)}
	{...rest}
>
	{@render children?.()}
</svelte:element>
