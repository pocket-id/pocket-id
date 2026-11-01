<script lang="ts">
	import { Checkbox as CheckboxPrimitive } from 'bits-ui';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import CheckIcon from '@lucide/svelte/icons/check';
	import MinusIcon from '@lucide/svelte/icons/minus';

	let {
		ref = $bindable(null),
		checked = $bindable(false),
		indeterminate = $bindable(false),
		class: className,
		...restProps
	}: WithoutChildrenOrChild<CheckboxPrimitive.RootProps> = $props();

	const markClass =
		'col-start-1 row-start-1 size-3 transition-[opacity,scale] duration-(--dur-base) ease-out';
</script>

<CheckboxPrimitive.Root
	bind:ref
	data-slot="checkbox"
	class={cn(
		'peer bg-surface text-primary-control-foreground before:bg-primary-control focus-visible:outline-ring relative isolate grid size-4.5 shrink-0 place-items-center rounded-xs shadow-[inset_0_0_0_1px_var(--border-control)] group-has-disabled/field:opacity-45 before:absolute before:inset-0 before:-z-1 before:rounded-[inherit] before:opacity-0 before:transition-opacity before:duration-(--dur-base) before:ease-out after:absolute after:-inset-x-3 after:-inset-y-2 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-45 aria-invalid:shadow-[inset_0_0_0_1px_var(--danger-solid)] data-checked:before:opacity-100 data-[state=indeterminate]:before:opacity-100',
		className
	)}
	bind:checked
	bind:indeterminate
	{...restProps}
>
	{#snippet children({ checked, indeterminate })}
		<span data-slot="checkbox-indicator" class="grid">
			<CheckIcon class={cn(markClass, !checked && 'scale-60 opacity-0')} />
			<MinusIcon class={cn(markClass, (checked || !indeterminate) && 'scale-60 opacity-0')} />
		</span>
	{/snippet}
</CheckboxPrimitive.Root>
