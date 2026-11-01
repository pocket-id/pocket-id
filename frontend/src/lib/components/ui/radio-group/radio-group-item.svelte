<script lang="ts">
	import { RadioGroup as RadioGroupPrimitive } from 'bits-ui';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithoutChildrenOrChild<RadioGroupPrimitive.ItemProps> = $props();
</script>

<RadioGroupPrimitive.Item
	bind:ref
	data-slot="radio-group-item"
	class={cn(
		'group/radio-group-item peer bg-surface focus-visible:outline-ring relative grid size-4.5 shrink-0 place-items-center rounded-full shadow-[inset_0_0_0_1px_var(--border-control)] before:absolute before:inset-0 before:rounded-full before:opacity-0 before:shadow-[inset_0_0_0_var(--selection-border)_var(--primary-control)] before:transition-opacity before:duration-(--dur-base) before:ease-out after:absolute after:-inset-x-3 after:-inset-y-2 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-45 aria-invalid:shadow-[inset_0_0_0_1px_var(--danger-solid)] data-checked:before:opacity-100',
		className
	)}
	{...restProps}
>
	{#snippet children({ checked })}
		<span
			data-slot="radio-group-indicator"
			class={cn(
				'bg-primary-control size-1.5 rounded-full transition-[opacity,scale] duration-(--dur-base) ease-out',
				!checked && 'scale-60 opacity-0'
			)}
		></span>
	{/snippet}
</RadioGroupPrimitive.Item>
