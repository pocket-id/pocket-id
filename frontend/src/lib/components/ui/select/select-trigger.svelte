<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui-old';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import { cn } from '$lib/utils/style.js';

	type $$Props = SelectPrimitive.TriggerProps;
	type $$Events = SelectPrimitive.TriggerEvents;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet<[any]>;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();

	const children_render = $derived(children);
</script>

<SelectPrimitive.Trigger
	class={cn(
		'border-input bg-background ring-offset-background focus-visible:ring-ring aria-[invalid]:border-destructive data-[placeholder]:[&>span]:text-muted-foreground flex h-10 w-full items-center justify-between rounded-md border px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1',
		className
	)}
	{...rest}
	on:click
	on:keydown
>
	{#snippet children({ builder })}
		{@render children_render?.({ builder })}
		<div>
			<ChevronDown class="h-4 w-4 opacity-50" />
		</div>
	{/snippet}
</SelectPrimitive.Trigger>
