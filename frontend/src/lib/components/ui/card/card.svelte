<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style.js';

	type $$Props = HTMLAttributes<HTMLDivElement>;

	let className: $$Props['class'] = undefined;
	export { className as class };
</script>

<div
	class={cn('bg-card text-card-foreground overflow-hidden rounded-lg border shadow-sm', className)}
	{...$$restProps}
>
	<slot />
</div>
