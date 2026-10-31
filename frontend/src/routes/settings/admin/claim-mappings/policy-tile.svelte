<script lang="ts">
	import { LucideBraces } from '@lucide/svelte';

	let { size }: { size: 24 | 40 } = $props();
</script>

<span
	aria-hidden="true"
	class="bg-fill-muted text-fg-muted grid shrink-0 place-items-center rounded-md shadow-[inset_0_0_0_1px_var(--border)]"
	style:width="{size}px"
	style:height="{size}px"
>
	<LucideBraces style="width: {Math.round(size * 0.45)}px; height: {Math.round(size * 0.45)}px" />
</span>
