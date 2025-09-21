<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';

	let { children, ...restProps }: HTMLAttributes<HTMLDivElement> = $props();
</script>

<div {...restProps}>
	{@render children?.()}
</div>
