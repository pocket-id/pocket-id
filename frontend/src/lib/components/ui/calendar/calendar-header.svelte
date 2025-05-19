<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CalendarPrimitive.HeaderProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<CalendarPrimitive.Header
	class={cn('relative flex w-full items-center justify-between pt-1', className)}
	{...rest}
>
	{@render children?.()}
</CalendarPrimitive.Header>
