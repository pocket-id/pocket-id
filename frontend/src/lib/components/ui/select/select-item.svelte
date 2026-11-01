<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import { cn, type WithoutChild } from '#lib/utils/style.js';
	import CheckIcon from '@lucide/svelte/icons/check';

	let {
		ref = $bindable(null),
		class: className,
		value,
		label,
		children: childrenProp,
		...restProps
	}: WithoutChild<SelectPrimitive.ItemProps> = $props();
</script>

<SelectPrimitive.Item
	bind:ref
	{value}
	data-slot="select-item"
	class={cn(
		"text-control text-fg data-highlighted:bg-fill-hover relative flex min-h-(--row-dense) w-full cursor-default items-center gap-2 rounded-sm px-2.5 outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-45 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
		className
	)}
	{...restProps}
>
	{#snippet children({ selected, highlighted })}
		<span class="flex size-4 items-center justify-center">
			{#if selected}
				<CheckIcon />
			{/if}
		</span>
		<span class="flex min-w-0 flex-1 items-center gap-2 py-1.5 whitespace-nowrap">
			{#if childrenProp}
				{@render childrenProp({ selected, highlighted })}
			{:else}
				<span class="truncate">{label || value}</span>
			{/if}
		</span>
	{/snippet}
</SelectPrimitive.Item>
