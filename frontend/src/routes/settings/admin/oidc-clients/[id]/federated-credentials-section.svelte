<script lang="ts">
	import DocsLink from '#lib/components/docs-link.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { OidcClient, OidcClientFederatedIdentity } from '#lib/types/oidc.type.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { LucideLock, LucidePlus } from '@lucide/svelte';
	import { z } from 'zod/v4';
	import FederatedIdentitiesInput from '../federated-identities-input.svelte';

	let {
		client,
		callback
	}: {
		client: OidcClient;
		callback: (federatedIdentities: OidcClientFederatedIdentity[]) => Promise<void>;
	} = $props();

	const isCIMDClient = client.clientType === 'cimd';

	const formSchema = z.object({
		federatedIdentities: z.array(
			z.object({
				issuer: z.url(),
				subject: z.string().optional(),
				audience: z.string().optional(),
				jwks: z.url().optional().or(z.literal('')),
				publicKeys: z.array(z.record(z.string(), z.unknown())).optional(),
				replayProtection: z.boolean().default(true)
			})
		)
	});
	const formStore = createForm(formSchema, {
		federatedIdentities:
			client.credentials?.federatedIdentities?.map((identity) => ({
				...identity,
				publicKeys: identity.publicKeys?.length ? identity.publicKeys : undefined
			})) ?? []
	});
	const { inputs, errors } = formStore;

	function addFederatedIdentity() {
		$inputs.federatedIdentities.value = [
			...$inputs.federatedIdentities.value,
			{
				issuer: '',
				subject: '',
				audience: '',
				jwks: '',
				publicKeys: undefined,
				replayProtection: true
			}
		];
	}

	// Metadata document clients manage their own credentials, so there is nothing to save here
	if (!isCIMDClient) {
		trackFormChanges(
			() => formStore,
			(data) => callback(data.federatedIdentities)
		);
	}
</script>

<SettingsSection title={m.federated_client_credentials()}>
	{#snippet description()}
		{m.federated_credentials_section_description({ name: client.name })}
		<DocsLink href="https://pocket-id.org/docs/guides/oidc-client-authentication" />
	{/snippet}
	{#snippet action()}
		{#if !isCIMDClient && $inputs.federatedIdentities.value.length > 0}
			<Button variant="secondary" size="sm" onclick={addFederatedIdentity}>
				<LucidePlus />
				{m.add_credential()}
			</Button>
		{/if}
	{/snippet}
	<div class="mt-2 flex flex-col gap-4" data-testid="federated-credentials">
		{#if isCIMDClient}
			<p class="text-caption text-fg-muted flex items-center gap-1.5">
				<LucideLock class="size-3" />
				{m.managed_by_the_clients_metadata_document()}
			</p>
		{/if}
		<FederatedIdentitiesInput
			bind:federatedIdentities={$inputs.federatedIdentities.value}
			errors={$errors?.issues
				.filter((issue) => issue.path[0] === 'federatedIdentities')
				.map((issue) => ({ ...issue, path: issue.path.slice(1) }))}
			disabled={isCIMDClient}
			onAdd={addFederatedIdentity}
		/>
	</div>
</SettingsSection>
