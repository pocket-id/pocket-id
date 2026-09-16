<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClient, OidcClientFederatedIdentity } from '$lib/types/oidc.type';
	import { createForm } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { slide } from 'svelte/transition';
	import { z } from 'zod/v4';
	import FederatedIdentitiesInput from '../federated-identities-input.svelte';

	let {
		client,
		callback
	}: {
		client: OidcClient;
		callback: (federatedIdentities: OidcClientFederatedIdentity[]) => Promise<void>;
	} = $props();

	const isCIMDClient = $derived(client.clientType === 'cimd');

	const formSchema = z.object({
		credentials: z.object({
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
		})
	});
	const formStore = createForm(formSchema, {
		credentials: {
			federatedIdentities:
				client.credentials?.federatedIdentities?.map((identity) => ({
					...identity,
					publicKeys: identity.publicKeys?.length ? identity.publicKeys : undefined
				})) ?? []
		}
	});
	const { inputs, errors } = formStore;

	const hasFederatedIdentities = $derived($inputs.credentials.value.federatedIdentities.length > 0);

	function getFederatedIdentityErrors(errors: z.ZodError<any> | undefined) {
		return errors?.issues
			.filter((error) =>
				['credentials', 'federatedIdentities'].every(
					(segment, index) => error.path[index] === segment
				)
			)
			.map((error) => ({ ...error, path: error.path.slice(2) }));
	}

	function addFederatedIdentity() {
		$inputs.credentials.value.federatedIdentities = [
			...$inputs.credentials.value.federatedIdentities,
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

	// Metadata document clients manage their own credentials, so there is nothing to save here.
	if (!isCIMDClient) {
		trackFormChanges(
			() => formStore,
			(data) => callback(data.credentials.federatedIdentities)
		);
	}
</script>

<Card.Root data-testid="federated-credentials-card">
	<Card.Header>
		<div class="flex items-center justify-between gap-4">
			<div>
				<Card.Title>{m.federated_client_credentials()}</Card.Title>
				<Card.Description>
					{m.federated_client_credentials_description()}
					<a
						class="underline underline-offset-4"
						href="https://pocket-id.org/docs/guides/oidc-client-authentication"
						target="_blank"
						rel="noreferrer"
					>
						{m.docs()}
					</a>
				</Card.Description>
			</div>
			{#if !hasFederatedIdentities}
				<Button disabled={isCIMDClient} onclick={addFederatedIdentity}>
					{m.create()}
				</Button>
			{/if}
		</div>
	</Card.Header>
	{#if hasFederatedIdentities}
		<div transition:slide>
			<Card.Content>
				<FederatedIdentitiesInput
					bind:federatedIdentities={$inputs.credentials.value.federatedIdentities}
					errors={getFederatedIdentityErrors($errors)}
					disabled={isCIMDClient}
				/>
			</Card.Content>
		</div>
	{/if}
</Card.Root>
