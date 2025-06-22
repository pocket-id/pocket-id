<script lang="ts" module>
	import { cn, type WithElementRef } from '$lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	export type DropdownButtonProps = WithElementRef<HTMLAttributes<HTMLDivElement>>;
</script>

<script lang="ts">
	let { class: className, ref = $bindable(null), children, ...restProps }: DropdownButtonProps = $props();
</script>

<div bind:this={ref} data-slot="dropdown-button" class={cn('flex', className)} {...restProps}>
	{@render children?.()}
</div>
