<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	let {
		hasAccess,
		granted,
		total
	}: {
		hasAccess: boolean;
		granted: number;
		total: number;
	} = $props();
</script>

{#if !hasAccess}
	<span class="text-muted-foreground text-sm">{m.no_access()}</span>
{:else}
	<span class="text-sm">
		{m.permissions_granted_count({ granted: String(granted), total: String(total) })}
	</span>
{/if}
