<script lang="ts">
	import * as FormPrimitive from 'formsnap';
	import { cn } from '$lib/utils/style.js';

	type $$Props = FormPrimitive.LegendProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet<[any]>;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	

	const children_render = $derived(children);
</script>

<FormPrimitive.Legend
	{...rest}
	class={cn('data-fs-error:text-destructive text-sm leading-none font-medium', className)}
	
>
	{#snippet children({ legendAttrs })}
		{@render children_render?.({ legendAttrs, })}
	{/snippet}
</FormPrimitive.Legend>
