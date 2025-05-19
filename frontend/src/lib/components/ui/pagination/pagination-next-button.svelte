<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { cn } from '$lib/utils/style.js';
	import { Pagination as PaginationPrimitive } from 'bits-ui-old';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';

	type $$Props = PaginationPrimitive.NextButtonProps;
	type $$Events = PaginationPrimitive.NextButtonEvents;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();

	const children_render = $derived(children);
</script>

<PaginationPrimitive.NextButton asChild>
	{#snippet children({ builder })}
		<Button
			variant="ghost"
			size="sm"
			class={cn('gap-1 pr-2.5', className)}
			builders={[builder]}
			on:click
			{...rest}
		>
			{#if children_render}{@render children_render()}{:else}
				<ChevronRight class="h-4 w-4" />
			{/if}
		</Button>
	{/snippet}
</PaginationPrimitive.NextButton>
