<script lang="ts">
	import { cn } from '#lib/utils/style.js';
	import type { SVGAttributes } from 'svelte/elements';

	let { class: className, ...restProps }: SVGAttributes<SVGSVGElement> = $props();
</script>

<!-- The animate-spin class is what the reduced-motion safety net in app.css exempts -->
<svg
	viewBox="0 0 16 16"
	aria-hidden="true"
	focusable="false"
	class={cn('size-4 shrink-0 animate-spin [animation-duration:var(--t-spin)]', className)}
	{...restProps}
>
	<circle
		cx="8"
		cy="8"
		r="6.25"
		fill="none"
		stroke="currentColor"
		stroke-width="1.5"
		stroke-linecap="round"
		stroke-dasharray="29.45 39.27"
	/>
</svg>
