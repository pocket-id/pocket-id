<script lang="ts">
	import * as FormPrimitive from 'formsnap';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLAttributes<HTMLSpanElement> & { class?: string | null | undefined };
	interface Props {
		class?: string | undefined | null;
		children?: import('svelte').Snippet<[any]>;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	

	const children_render = $derived(children);
</script>

<FormPrimitive.Description
	class={cn('text-muted-foreground text-sm', className)}
	{...rest}
	
>
	{#snippet children({ descriptionAttrs })}
		{@render children_render?.({ descriptionAttrs, })}
	{/snippet}
</FormPrimitive.Description>
