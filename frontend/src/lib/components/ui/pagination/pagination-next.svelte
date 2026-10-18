<script lang="ts">
	import { Pagination as PaginationPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: PaginationPrimitive.NextButtonProps = $props();
</script>

<PaginationPrimitive.NextButton
	bind:ref
	aria-label="Go to next page"
	class={cn(buttonVariants({ variant: 'secondary', size: 'sm' }), 'max-sm:px-2', className)}
	{...restProps}
>
	<span class="max-sm:sr-only">Next</span>
	<ChevronRightIcon data-icon="inline-end" />
</PaginationPrimitive.NextButton>
