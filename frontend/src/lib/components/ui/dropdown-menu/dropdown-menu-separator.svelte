<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = DropdownMenuPrimitive.SeparatorProps;

	interface Props {
		class?: $$Props['class'];
		[key: string]: any;
	}

	let { class: className = undefined, ...rest }: Props = $props();
</script>

<DropdownMenuPrimitive.Separator class={cn('bg-muted -mx-1 my-1 h-px', className)} {...rest} />
