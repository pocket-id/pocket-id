<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Card from '#lib/components/ui/card/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import OidcClaimMappingPolicyForm from './oidc-claim-mapping-policy-form.svelte';
	import { toast } from 'svelte-sonner';
	import OidcClaimMappingPolicyList from './oidc-claim-mapping-policy-list.svelte';
	import type { OidcClaimMappingPolicyDto } from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { LucideMinus, LucideServer, ShieldPlus } from '@lucide/svelte';
	import { slide } from 'svelte/transition';

	let expandAddPolicy = $state(false);

	const oidcClaimMappingPolicyService = new OidcClaimMappingPolicyService();

	async function createClaimMappingPolicy(policy: OidcClaimMappingPolicyDto) {
		await oidcClaimMappingPolicyService
			.createClaimMappingPolicy(policy)
			.then((createdPolicy) => {
				toast.success(m.claim_mapping_policy_created_successfully());
				goto(`/settings/admin/oidc-claim-mapping-policies/${createdPolicy.id}`);
			})
			.catch((e) => {
				axiosErrorToast(e);
			});
	}
</script>

<svelte:head>
	<title>{m.claim_mapping_policies()}</title>
</svelte:head>

<div>
	<Card.Root>
		<Card.Header>
			<div class="flex flex-wrap items-center justify-between gap-4 md:flex-nowrap">
				<div>
					<Card.Title>
						<ShieldPlus class="text-primary/80 size-5" />
						{m.create_claim_mapping_policy()}
					</Card.Title>
					<Card.Description>{m.create_claim_mapping_policy_description()}</Card.Description>
				</div>
				{#if !expandAddPolicy}
					<Button class="w-full md:w-auto" onclick={() => (expandAddPolicy = true)}
						>{m.create_claim_mapping_policy()}</Button
					>
				{:else}
					<Button class="h-8 p-3" variant="ghost" onclick={() => (expandAddPolicy = false)}>
						<LucideMinus class="size-5" />
					</Button>
				{/if}
			</div>
		</Card.Header>
		{#if expandAddPolicy}
			<div transition:slide>
				<Card.Content>
					<OidcClaimMappingPolicyForm callback={createClaimMappingPolicy} mode="create" />
				</Card.Content>
			</div>
		{/if}
	</Card.Root>
</div>

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<LucideServer class="text-primary/80 size-5" />
				{m.manage_claim_mapping_policies()}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<OidcClaimMappingPolicyList />
		</Card.Content>
	</Card.Root>
</div>
