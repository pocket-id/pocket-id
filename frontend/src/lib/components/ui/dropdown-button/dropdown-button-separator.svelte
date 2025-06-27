<script lang="ts" module>
	import { cn } from '$lib/utils/style.js';

	export type DropdownButtonSeparatorProps = DropdownMenuPrimitive.SeparatorProps;
</script>

<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DropdownMenuPrimitive.SeparatorProps = $props();
</script>

<DropdownMenuPrimitive.Separator
	bind:ref
	class={cn('bg-muted -mx-1 my-1 h-px', className)}
	{...restProps}
/>
