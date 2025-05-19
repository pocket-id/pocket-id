<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';

	type $$Props = SelectPrimitive.SeparatorProps;

	interface Props {
		class?: $$Props['class'];
		[key: string]: any
	}

	let { class: className = undefined, ...rest }: Props = $props();
	
</script>

<SelectPrimitive.Separator class={cn('bg-muted -mx-1 my-1 h-px', className)} {...rest} />
