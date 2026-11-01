<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import MinusIcon from '@lucide/svelte/icons/minus';
	import CheckIcon from '@lucide/svelte/icons/check';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		checked = $bindable(false),
		indeterminate = $bindable(false),
		class: className,
		children: childrenProp,
		...restProps
	}: WithoutChildrenOrChild<DropdownMenuPrimitive.CheckboxItemProps> & {
		children?: Snippet;
	} = $props();
</script>

<DropdownMenuPrimitive.CheckboxItem
	bind:ref
	bind:checked
	bind:indeterminate
	data-slot="dropdown-menu-checkbox-item"
	class={cn(
		"data-highlighted:bg-fg/6 text-control [&_svg:not([class*='text-'])]:text-fg-muted relative flex min-h-9 cursor-pointer items-center gap-2 rounded-sm px-2.5 pr-8 outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-45 data-inset:pl-8.5 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
		className
	)}
	{...restProps}
>
	{#snippet children({ checked, indeterminate })}
		<span
			class="pointer-events-none absolute right-2.5 flex items-center justify-center"
			data-slot="dropdown-menu-checkbox-item-indicator"
		>
			{#if indeterminate}
				<MinusIcon class="text-fg" />
			{:else if checked}
				<CheckIcon class="text-fg" />
			{/if}
		</span>
		{@render childrenProp?.()}
	{/snippet}
</DropdownMenuPrimitive.CheckboxItem>
