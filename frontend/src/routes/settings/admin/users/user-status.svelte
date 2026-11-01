<script lang="ts">
	import { Badge } from '@pocket-id/ui/badge';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { User } from '#lib/types/user.type.ts';

	let { user }: { user: Pick<User, 'disabled' | 'emailVerified'> } = $props();
</script>

{#if user.disabled}
	<Badge variant="destructive">{m.disabled()}</Badge>
{:else if !user.emailVerified && $appConfigStore.emailVerification !== 'disabled'}
	<Badge variant="warning">{m.unverified()}</Badge>
{/if}
