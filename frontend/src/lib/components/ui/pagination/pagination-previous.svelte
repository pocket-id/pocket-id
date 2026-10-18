<script lang="ts">
	import { Pagination as PaginationPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: PaginationPrimitive.PrevButtonProps = $props();
</script>

<PaginationPrimitive.PrevButton
	bind:ref
	aria-label="Go to previous page"
	class={cn(buttonVariants({ variant: 'secondary', size: 'sm' }), 'max-sm:px-2', className)}
	{...restProps}
>
	<ChevronLeftIcon data-icon="inline-start" />
	<span class="max-sm:sr-only">Previous</span>
</PaginationPrimitive.PrevButton>
