<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<!-- On phones the decision sticks to the bottom of a sheet taller than the window, reaching the sheet's edges through the padding variables login-wrapper sets on its column -->
<!-- Its hairline shows only while the column reports content below the fold, and data-action-panel gives the column the scroll padding that keeps keyboard focus from hiding under it -->
<div
	class="max-sm:bg-surface max-sm:group-data-content-below/sheet:border-border max-sm:sticky max-sm:-bottom-(--sheet-b) max-sm:-mx-(--sheet-x) max-sm:-mt-3 max-sm:-mb-(--sheet-b) max-sm:border-t max-sm:border-transparent max-sm:px-(--sheet-x) max-sm:pt-3 max-sm:pb-(--sheet-b) max-sm:transition-colors max-sm:duration-(--dur-fast)"
	data-action-panel
>
	{@render children()}
</div>
