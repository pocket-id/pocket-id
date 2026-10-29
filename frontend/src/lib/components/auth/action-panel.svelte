<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<!-- On phones the decision sticks to the bottom of a sheet taller than the window, reaching the sheet's edges through the padding variables login-wrapper sets on its column -->
<!-- Its hairline shows only while the column reports content below the fold, and data-action-panel gives the column the scroll padding that keeps keyboard focus from hiding under it -->
<div
	class="max-lg:bg-surface max-lg:group-data-content-below/sheet:border-border max-lg:sticky max-lg:-bottom-(--sheet-b) max-lg:-mx-(--sheet-x) max-lg:-mt-3 max-lg:-mb-(--sheet-b) max-lg:border-t max-lg:border-transparent max-lg:px-(--sheet-x) max-lg:pt-3 max-lg:pb-(--sheet-b) max-lg:transition-colors max-lg:duration-(--dur-fast)"
	data-action-panel
>
	{@render children()}
</div>
