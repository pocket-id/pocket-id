<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		tabindex,
		children,
		...restProps
	}: TabsPrimitive.ContentProps = $props();
</script>

<!-- bits-ui always gives the panel a tabindex, which adds a Tab stop before its first control and lets clicks on its non-focusable parts focus it -->
<!-- Pass tabindex={0} for panels without focusable content -->
<TabsPrimitive.Content
	bind:ref
	data-slot="tabs-content"
	class={cn('flex-1', className)}
	{...restProps}
>
	{#snippet child({ props })}
		<div role="tabpanel" {...props} {tabindex}>
			{@render children?.()}
		</div>
	{/snippet}
</TabsPrimitive.Content>
