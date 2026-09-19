<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as RadioGroup from '$lib/components/ui/radio-group';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { OidcClientFederatedIdentity } from '$lib/types/oidc.type';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { z } from 'zod/v4';
	import FederatedIdentityKeysInput from './federated-identity-keys-input.svelte';

	// An identity is verified either against the keys of a JWKS endpoint, or against the public keys configured here
	type KeySource = 'jwks' | 'publicKeys';

	let {
		federatedIdentities = $bindable([]),
		errors,
		disabled = false,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		federatedIdentities: OidcClientFederatedIdentity[];
		errors?: z.core.$ZodIssue[];
		disabled?: boolean;
		children?: Snippet;
	} = $props();

	// An empty publicKeys array records that Public keys is selected before the first key is added
	function keySourceFor(identity: OidcClientFederatedIdentity): KeySource {
		return identity.publicKeys === undefined ? 'jwks' : 'publicKeys';
	}

	function addFederatedIdentity() {
		federatedIdentities = [
			...federatedIdentities,
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

	function removeFederatedIdentity(index: number) {
		federatedIdentities = federatedIdentities.filter((_, i) => i !== index);
	}

	function updateFederatedIdentity<K extends keyof OidcClientFederatedIdentity>(
		index: number,
		field: K,
		value: OidcClientFederatedIdentity[K]
	) {
		federatedIdentities[index] = {
			...federatedIdentities[index],
			[field]: value
		};
	}

	// Only one of the two sources is ever submitted, so the one that is not selected is cleared
	function updateKeySource(index: number, source: KeySource) {
		const identity = federatedIdentities[index];
		federatedIdentities[index] = {
			...identity,
			jwks: source === 'jwks' ? identity.jwks : '',
			publicKeys: source === 'publicKeys' ? (identity.publicKeys ?? []) : undefined
		};
	}

	function getFieldError(index: number, field: keyof OidcClientFederatedIdentity): string | null {
		if (!errors) return null;
		const path = [index, field];
		return errors?.filter((e) => e.path[0] == path[0] && e.path[1] == path[1])[0]?.message;
	}
</script>

<div {...restProps}>
	<FormInput {disabled}>
		<div class="flex flex-col gap-4">
			{#each federatedIdentities as identity, i (i)}
				<div class="flex flex-col gap-3">
					<div class="flex items-center justify-between">
						<Field.Label>{m.federated_identity_number({ number: i + 1 })}</Field.Label>
						{#if federatedIdentities.length > 0}
							<Button
								variant="outline"
								size="sm"
								onclick={() => removeFederatedIdentity(i)}
								aria-label={m.remove_federated_identity()}
								{disabled}
							>
								<LucideMinus data-icon="inline-start" />
							</Button>
						{/if}
					</div>

					<div class="grid grid-cols-1 gap-5 md:grid-cols-2">
						<Field.Field>
							<Field.Label required for="issuer-{i}">{m.issuer()}</Field.Label>
							<Input
								id="issuer-{i}"
								placeholder="https://example.com/"
								value={identity.issuer}
								oninput={(e) => updateFederatedIdentity(i, 'issuer', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'issuer')}
								{disabled}
							/>
							{#if getFieldError(i, 'issuer')}
								<Field.Error>{getFieldError(i, 'issuer')}</Field.Error>
							{/if}
						</Field.Field>

						<Field.Field>
							<Field.Label for="subject-{i}">{m.subject()}</Field.Label>
							<Input
								id="subject-{i}"
								placeholder={m.defaults_to_the_client_id()}
								value={identity.subject || ''}
								oninput={(e) => updateFederatedIdentity(i, 'subject', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'subject')}
								{disabled}
							/>
							{#if getFieldError(i, 'subject')}
								<Field.Error>{getFieldError(i, 'subject')}</Field.Error>
							{/if}
						</Field.Field>

						<Field.Field>
							<Field.Label for="audience-{i}">{m.audience()}</Field.Label>
							<Input
								id="audience-{i}"
								placeholder={m.defaults_to_the_appname_url({ appName: $appConfigStore.appName })}
								value={identity.audience || ''}
								oninput={(e) => updateFederatedIdentity(i, 'audience', e.currentTarget.value)}
								aria-invalid={!!getFieldError(i, 'audience')}
								{disabled}
							/>
							{#if getFieldError(i, 'audience')}
								<Field.Error>{getFieldError(i, 'audience')}</Field.Error>
							{/if}
						</Field.Field>

						<Field.Field class="md:col-span-2">
							<Field.Label>{m.signing_keys()}</Field.Label>
							<RadioGroup.Root
								class="flex flex-wrap gap-x-6 gap-y-3"
								value={keySourceFor(identity)}
								onValueChange={(value) => updateKeySource(i, value as KeySource)}
								{disabled}
							>
								<div class="flex items-center gap-2">
									<RadioGroup.Item value="jwks" id="key-source-jwks-{i}" />
									<Label for="key-source-jwks-{i}" class="mb-0 font-normal">{m.jwks_url()}</Label>
								</div>
								<div class="flex items-center gap-2">
									<RadioGroup.Item value="publicKeys" id="key-source-public-keys-{i}" />
									<Label for="key-source-public-keys-{i}" class="mb-0 font-normal">
										{m.public_keys()}
									</Label>
								</div>
							</RadioGroup.Root>

							{#if keySourceFor(identity) === 'publicKeys'}
								<FederatedIdentityKeysInput
									id="public-keys-{i}"
									publicKeys={identity.publicKeys ?? []}
									onChange={(publicKeys) => updateFederatedIdentity(i, 'publicKeys', publicKeys)}
									{disabled}
								/>
								{#if getFieldError(i, 'publicKeys')}
									<Field.Error>{getFieldError(i, 'publicKeys')}</Field.Error>
								{/if}
							{:else}
								<Input
									id="jwks-{i}"
									aria-label={m.jwks_url()}
									placeholder={m.defaults_to_the_issuer_jwks_url({
										issuer: identity.issuer || '<issuer>'
									})}
									value={identity.jwks || ''}
									oninput={(e) => updateFederatedIdentity(i, 'jwks', e.currentTarget.value)}
									aria-invalid={!!getFieldError(i, 'jwks')}
									{disabled}
								/>
								{#if getFieldError(i, 'jwks')}
									<Field.Error>{getFieldError(i, 'jwks')}</Field.Error>
								{/if}
							{/if}
						</Field.Field>

						<SwitchWithLabel
							id="replay-protection-{i}"
							label={m.replay_protection()}
							description={m.replay_protection_description()}
							checked={identity.replayProtection}
							onCheckedChange={(checked) => updateFederatedIdentity(i, 'replayProtection', checked)}
							{disabled}
						/>
					</div>
				</div>
			{/each}
		</div>
	</FormInput>

	<Button
		class="mt-7"
		variant="secondary"
		size="sm"
		onclick={addFederatedIdentity}
		type="button"
		{disabled}
	>
		<LucidePlus data-icon="inline-start" />
		{federatedIdentities.length === 0
			? m.add_federated_client_credential()
			: m.add_another_federated_client_credential()}
	</Button>
</div>
