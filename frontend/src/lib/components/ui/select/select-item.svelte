<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import { Select as SelectPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = SelectPrimitive.ItemProps;
	type $$Events = SelectPrimitive.ItemEvents;

	interface Props {
		class?: $$Props['class'];
		value: $$Props['value'];
		label?: $$Props['label'];
		disabled?: $$Props['disabled'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let {
		class: className = undefined,
		value,
		label = undefined,
		disabled = undefined,
		children,
		...rest
	}: Props = $props();
</script>

<SelectPrimitive.Item
	{value}
	{disabled}
	{label}
	class={cn(
		'data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground relative flex w-full select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
		className
	)}
	{...rest}
	on:click
	on:keydown
	on:focusin
	on:focusout
	on:pointerleave
	on:pointermove
>
	<span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
		<SelectPrimitive.ItemIndicator>
			<Check class="h-4 w-4" />
		</SelectPrimitive.ItemIndicator>
	</span>
	{#if children}{@render children()}{:else}
		{label || value}
	{/if}
</SelectPrimitive.Item>
