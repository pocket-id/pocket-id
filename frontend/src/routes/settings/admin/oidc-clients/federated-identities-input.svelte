<script lang="ts">
	import SettingRow from '#lib/components/setting-row.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { Label } from '#lib/components/ui/label/index.ts';
	import * as RadioGroup from '#lib/components/ui/radio-group/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { OidcClientFederatedIdentity } from '#lib/types/oidc.type.ts';
	import { LucidePlus, LucideX } from '@lucide/svelte';
	import { z } from 'zod/v4';
	import FederatedIdentityKeysInput from './federated-identity-keys-input.svelte';

	// An identity is verified either against the keys of a JWKS endpoint, or against the public keys configured here
	type KeySource = 'jwks' | 'publicKeys';
	type TextField = 'issuer' | 'subject' | 'audience' | 'jwks';

	let {
		federatedIdentities = $bindable([]),
		errors,
		disabled = false,
		onAdd
	}: {
		federatedIdentities: OidcClientFederatedIdentity[];
		errors?: z.core.$ZodIssue[];
		disabled?: boolean;
		onAdd: () => void;
	} = $props();

	// An empty publicKeys array records that Public keys is selected before the first key is added
	function keySourceFor(identity: OidcClientFederatedIdentity): KeySource {
		return identity.publicKeys === undefined ? 'jwks' : 'publicKeys';
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

	function getFieldError(index: number, field: keyof OidcClientFederatedIdentity) {
		return errors?.find((e) => e.path[0] === index && e.path[1] === field)?.message;
	}
</script>

{#snippet textField(
	i: number,
	identity: OidcClientFederatedIdentity,
	field: TextField,
	label: string | undefined,
	placeholder: string
)}
	{@const error = getFieldError(i, field)}
	<Field.Field>
		{#if label}
			<Field.Label optional={field !== 'issuer'} for="{field}-{i}">{label}</Field.Label>
		{/if}
		<Input
			id="{field}-{i}"
			class="font-mono"
			aria-label={label ? undefined : m.jwks_url()}
			{placeholder}
			value={identity[field] || ''}
			oninput={(e) => updateFederatedIdentity(i, field, e.currentTarget.value)}
			aria-invalid={!!error}
			readonly={disabled}
		/>
		{#if error}
			<Field.Error>{error}</Field.Error>
		{/if}
	</Field.Field>
{/snippet}

{#each federatedIdentities as identity, i (i)}
	<div class="flex flex-col gap-5 rounded-md border p-4">
		<div class="flex items-center justify-between gap-3">
			<h3 class="text-heading">{m.credential_number({ number: i + 1 })}</h3>
			{#if !disabled}
				<Button
					variant="ghost"
					size="icon-xs"
					onclick={() => removeFederatedIdentity(i)}
					aria-label={m.remove_credential_number({ number: i + 1 })}
				>
					<LucideX />
				</Button>
			{/if}
		</div>

		<div class="grid gap-5 md:grid-cols-2">
			<div class="md:col-span-2">
				{@render textField(i, identity, 'issuer', m.issuer(), 'https://ci.example.com')}
			</div>
			{@render textField(i, identity, 'subject', m.subject(), m.defaults_to_the_client_id())}
			{@render textField(
				i,
				identity,
				'audience',
				m.audience(),
				m.defaults_to_the_appname_url({ appName: $appConfigStore.appName })
			)}
		</div>

		<Field.Field>
			<Field.Label>{m.signing_keys()}</Field.Label>
			<RadioGroup.Root
				class="flex flex-wrap gap-x-6 gap-y-3"
				value={keySourceFor(identity)}
				onValueChange={(value) => updateKeySource(i, value as KeySource)}
				{disabled}
			>
				<div class="flex items-center gap-2">
					<RadioGroup.Item value="jwks" id="key-source-jwks-{i}" />
					<Label for="key-source-jwks-{i}" class="font-normal">{m.jwks_url()}</Label>
				</div>
				<div class="flex items-center gap-2">
					<RadioGroup.Item value="publicKeys" id="key-source-public-keys-{i}" />
					<Label for="key-source-public-keys-{i}" class="font-normal">{m.public_keys()}</Label>
				</div>
			</RadioGroup.Root>
		</Field.Field>

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
			{@render textField(
				i,
				identity,
				'jwks',
				undefined,
				m.defaults_to_the_issuer_jwks_url({ issuer: identity.issuer || '<issuer>' })
			)}
		{/if}

		<SettingRow title={m.replay_protection()} description={m.replay_protection_row_description()}>
			{#snippet control(props)}
				<Switch
					{...props}
					{disabled}
					checked={identity.replayProtection}
					onCheckedChange={(checked) => updateFederatedIdentity(i, 'replayProtection', checked)}
				/>
			{/snippet}
		</SettingRow>
	</div>
{:else}
	<div
		class="flex min-h-11 flex-wrap items-center justify-between gap-2 rounded-md border ps-4 pe-2"
	>
		<span class="text-control text-fg-muted">{m.no_federated_credentials()}</span>
		{#if !disabled}
			<Button variant="ghost" size="sm" onclick={onAdd}>
				<LucidePlus />
				{m.add_credential()}
			</Button>
		{/if}
	</div>
{/each}
