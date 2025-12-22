<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClient, OidcClientFederatedIdentity } from '$lib/types/oidc.type';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { z } from 'zod/v4';

	let {
		client,
		federatedIdentities = $bindable([]),
		errors,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		client?: OidcClient;
		federatedIdentities: OidcClientFederatedIdentity[];
		errors?: z.core.$ZodIssue[];

		children?: Snippet;
	} = $props();

	function addFederatedIdentity() {
		federatedIdentities = [
			...federatedIdentities,
			{
				issuer: '',
				subject: '',
				audience: '',
				jwks: ''
			}
		];
	}

	function removeFederatedIdentity(index: number) {
		federatedIdentities = federatedIdentities.filter((_, i) => i !== index);
	}

	function updateFederatedIdentity(
		index: number,
		field: keyof OidcClientFederatedIdentity,
		value: string
	) {
		federatedIdentities[index] = {
			...federatedIdentities[index],
			[field]: value
		};
	}

	function getFieldError(index: number, field: keyof OidcClientFederatedIdentity): string | null {
		if (!errors) return null;
		const path = [index, field];
		return errors?.filter((e) => e.path[0] == path[0] && e.path[1] == path[1])[0]?.message;
	}
</script>

<div {...restProps}>
	<FormInput
		label={m.federated_client_credentials()}
		description={m.federated_client_credentials_description()}
		docsLink="https://pocket-id.org/docs/guides/oidc-client-authentication"
	>
		<div class="space-y-4">
			{#each federatedIdentities as identity, i}
				<div class="space-y-3 rounded-lg border p-4">
					<div class="flex items-center justify-between">
						<Field.Label>Identity {i + 1}</Field.Label>
						{#if federatedIdentities.length > 0}
							<Button
								variant="outline"
								size="sm"
								onclick={() => removeFederatedIdentity(i)}
								aria-label="Remove federated identity"
							>
								<LucideMinus class="size-4" />
							</Button>
						{/if}
					</div>

					<div class="grid grid-cols-1 gap-3 md:grid-cols-2">
						<Field.Field>
							<Field.Label required for="issuer-{i}">Issuer</Field.Label>
							<Input
								id="issuer-{i}"
								placeholder="https://example.com/"
								value={identity.issuer}
								oninput={(e) => updateFederatedIdentity(i, 'issuer', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'issuer')}
							/>
							{#if getFieldError(i, 'issuer')}
								<Field.Error>{getFieldError(i, 'issuer')}</Field.Error>
							{/if}
						</Field.Field>

						<Field.Field>
							<Field.Label for="subject-{i}">Subject</Field.Label>
							<Input
								id="subject-{i}"
								placeholder="Defaults to the client ID"
								value={identity.subject || ''}
								oninput={(e) => updateFederatedIdentity(i, 'subject', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'subject')}
							/>
							{#if getFieldError(i, 'subject')}
								<Field.Error>{getFieldError(i, 'subject')}</Field.Error>
							{/if}
						</Field.Field>

						<Field.Field>
							<Field.Label for="audience-{i}">Audience</Field.Label>
							<Input
								id="audience-{i}"
								placeholder="Defaults to the Pocket ID URL"
								value={identity.audience || ''}
								oninput={(e) => updateFederatedIdentity(i, 'audience', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'audience')}
							/>
							{#if getFieldError(i, 'audience')}
								<Field.Error>{getFieldError(i, 'audience')}</Field.Error>
							{/if}
						</Field.Field>

						<Field.Field>
							<Field.Label for="jwks-{i}">JWKS URL</Field.Label>
							<Input
								id="jwks-{i}"
								placeholder="Defaults to {identity.issuer || '<issuer>'}/.well-known/jwks.json"
								value={identity.jwks || ''}
								oninput={(e) => updateFederatedIdentity(i, 'jwks', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'jwks')}
							/>
							{#if getFieldError(i, 'jwks')}
								<Field.Error>{getFieldError(i, 'jwks')}</Field.Error>
							{/if}
						</Field.Field>
					</div>
				</div>
			{/each}
		</div>
	</FormInput>

	<Button class="mt-3" variant="secondary" size="sm" onclick={addFederatedIdentity} type="button">
		<LucidePlus class="mr-1 size-4" />
		{federatedIdentities.length === 0
			? m.add_federated_client_credential()
			: m.add_another_federated_client_credential()}
	</Button>
</div>
