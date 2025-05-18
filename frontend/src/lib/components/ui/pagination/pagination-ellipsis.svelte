<script lang="ts">
	import Ellipsis from 'lucide-svelte/icons/ellipsis';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLAttributes<HTMLSpanElement>;

	interface Props {
		class?: $$Props['class'];
		[key: string]: any
	}

	let { class: className = undefined, ...rest }: Props = $props();
	
</script>

<span
	aria-hidden="true"
	class={cn('flex h-9 w-9 items-center justify-center', className)}
	{...rest}
>
	<Ellipsis class="h-4 w-4" />
	<span class="sr-only">More pages</span>
</span>
