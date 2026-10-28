<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import type { InteractionClient } from '#lib/types/oidc.type.ts';
	import { getClientIDHost } from '#lib/utils/client-id-util.ts';
	import { LucideCheck, LucideTriangleAlert } from '@lucide/svelte';

	let { client }: { client: InteractionClient } = $props();

	// Metadata-document clients registered themselves, so their name proves nothing
	const host = $derived(getClientIDHost(client));
</script>

<div class="text-caption text-fg-muted -mt-4 mb-6 flex items-start gap-1.5">
	{#if host}
		<LucideTriangleAlert class="text-warning-solid mt-0.5 size-3 shrink-0" />
		<div>
			<p class="text-fg font-medium">{m.auth_not_registered_by_admin()}</p>
			<p class="wrap-anywhere">{m.auth_not_registered_by_admin_description({ host })}</p>
		</div>
	{:else}
		<LucideCheck class="mt-0.5 size-3 shrink-0" />
		<p>{m.auth_registered_by_admin()}</p>
	{/if}
</div>
