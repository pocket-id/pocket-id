<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CalendarPrimitive.HeadingProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet<[any]>;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	

	const children_render = $derived(children);
</script>

<CalendarPrimitive.Heading
	
	class={cn('text-sm font-medium', className)}
	{...rest}
>
	{#snippet children({ headingValue })}
		{#if children_render}{@render children_render({ headingValue, })}{:else}
			{headingValue}
		{/if}
	{/snippet}
</CalendarPrimitive.Heading>
