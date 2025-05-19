<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = DialogPrimitive.DescriptionProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<DialogPrimitive.Description class={cn('text-muted-foreground text-sm', className)} {...rest}>
	{@render children?.()}
</DialogPrimitive.Description>
