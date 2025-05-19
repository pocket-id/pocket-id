<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CalendarPrimitive.GridBodyProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<CalendarPrimitive.GridBody class={cn(className)} {...rest}>
	{@render children?.()}
</CalendarPrimitive.GridBody>
