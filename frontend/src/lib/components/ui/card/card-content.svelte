<script lang="ts">
	import { cn } from '$lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	type $$Props = HTMLAttributes<HTMLDivElement>;

	let className: $$Props['class'] = undefined;
	export { className as class };
</script>

<div class={cn('bg-muted/20 p-6 pt-5 peer-[.card-header]:border-t', className)} {...$$restProps}>
	<slot />
</div>
