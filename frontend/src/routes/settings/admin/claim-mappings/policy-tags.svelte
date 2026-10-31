<script lang="ts">
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { builtInClaimMappingPolicyId } from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { LucideLock } from '@lucide/svelte';

	let { policy }: { policy: { id: string; isDefault: boolean } } = $props();
</script>

{#if policy.isDefault}<Badge>{m.default_policy_tag()}</Badge>{/if}
{#if policy.id === builtInClaimMappingPolicyId}
	<Badge><LucideLock class="size-3" />{m.built_in()}</Badge>
{/if}
