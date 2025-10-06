<script lang="ts">
	import { cn } from '$lib/utils/style.js';
	import { Label as LabelPrimitive } from 'bits-ui';

	let {
		ref = $bindable(null),
		class: className,
		required = false,
		children,
		...restProps
	}: LabelPrimitive.RootProps & {
		required?: boolean;
	} = $props();
</script>

<LabelPrimitive.Root
	bind:ref
	data-slot="label"
	class={cn(
		'mb-3 flex items-center gap-1.5 text-sm leading-none font-medium select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50',
		required && "after:text-[14px] after:text-red-500 after:content-['*']",
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</LabelPrimitive.Root>
