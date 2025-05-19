<script lang="ts">
	import { AlertDialog as AlertDialogPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = AlertDialogPrimitive.DescriptionProps;

	interface Props {
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { class: className = undefined, children, ...rest }: Props = $props();
</script>

<AlertDialogPrimitive.Description class={cn('text-muted-foreground text-sm', className)} {...rest}>
	{@render children?.()}
</AlertDialogPrimitive.Description>
