<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CalendarPrimitive.GridRowProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
	
</script>

<CalendarPrimitive.GridRow class={cn('flex', className)} {...rest}>
	{@render children?.()}
</CalendarPrimitive.GridRow>
