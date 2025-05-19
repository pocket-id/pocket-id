<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CalendarPrimitive.GridProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<CalendarPrimitive.Grid class={cn('w-full border-collapse space-y-1', className)} {...rest}>
	{@render children?.()}
</CalendarPrimitive.Grid>
