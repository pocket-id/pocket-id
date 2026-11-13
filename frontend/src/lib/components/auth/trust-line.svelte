<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import type { InteractionClient } from '#lib/types/oidc.type.ts';
	import { getClientIDHost } from '#lib/utils/client-id-util.ts';
	import * as Alert from '@pocket-id/ui/alert';
	import TriangleAlertIcon from '~icons/solar/danger-triangle-bold-duotone';

	let { client }: { client: InteractionClient } = $props();

	// Metadata-document clients registered themselves, so their name proves nothing
	const host = $derived(getClientIDHost(client));
</script>

{#if host}
	<Alert.Root variant="warning" class="mb-6">
		<TriangleAlertIcon />
		<Alert.Title>{m.auth_not_registered_by_admin()}</Alert.Title>
		<Alert.Description class="wrap-anywhere">
			{m.auth_not_registered_by_admin_description({ host })}
		</Alert.Description>
	</Alert.Root>
{/if}
