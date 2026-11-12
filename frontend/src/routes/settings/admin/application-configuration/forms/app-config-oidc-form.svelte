<script lang="ts">
	import CopyRows from '#lib/components/copy-rows.svelte';
	import DocsLink from '#lib/components/docs-link.svelte';
	import EnvOverrideLock from '#lib/components/env-override-lock.svelte';
	import UrlListInput from '#lib/components/form/url-list-input.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsCard from '#lib/components/settings-card.svelte';
	import * as Field from '@pocket-id/ui/field';
	import { Switch } from '@pocket-id/ui/switch';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import type { OidcDiscoveryConfiguration } from '#lib/types/oidc.type.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges, trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import KeyIcon from '~icons/solar/key-bold-duotone';
	import { z } from 'zod/v4';

	let {
		appConfig,
		envOverrides,
		oidcConfiguration,
		showCimd,
		callback
	}: {
		appConfig: AllAppConfig;
		envOverrides: Set<keyof AllAppConfig>;
		oidcConfiguration: OidcDiscoveryConfiguration;
		showCimd: boolean;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const endpoints = [
		{ label: m.issuer(), value: oidcConfiguration.issuer },
		{
			label: m.config_discovery_url(),
			value: `${oidcConfiguration.issuer}/.well-known/openid-configuration`
		},
		{ label: m.config_jwks(), value: oidcConfiguration.jwks_uri }
	].map((endpoint) => ({
		...endpoint,
		copyLabel: m.config_copy_endpoint({ name: endpoint.label })
	}));

	const formStore = createForm(z.object({ autoCreateOidcClientSecret: z.boolean() }), {
		autoCreateOidcClientSecret: appConfig.autoCreateOidcClientSecret
	});
	const inputs = formStore.inputs;
	trackFormChanges(() => formStore, callback);

	let cimdUrlAllowlist: string[] = $state(appConfig.cimdUrlAllowlist ?? []);
	if (cimdUrlAllowlist.length === 0) cimdUrlAllowlist.push('');

	trackUnsavedValue(
		() => cimdUrlAllowlist.filter((u) => u.trim() !== ''),
		(value) => {
			cimdUrlAllowlist = value;
			if (cimdUrlAllowlist.length === 0) cimdUrlAllowlist.push('');
		},
		(value) => callback({ cimdUrlAllowlist: value })
	);
</script>

<div class="flex min-w-0 flex-col gap-4">
	<SettingsCard title={m.config_endpoints()}>
		<div class="py-2">
			<CopyRows rows={endpoints} />
		</div>
	</SettingsCard>

	<fieldset class="contents" disabled={$appConfigStore.uiConfigDisabled}>
		<SettingsCard title={m.config_new_clients()}>
			<SettingRow
				title={m.config_auto_secret()}
				description={m.config_auto_secret_description()}
				icon={KeyIcon}
				envOverride={envOverrides.has('autoCreateOidcClientSecret')}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.autoCreateOidcClientSecret.value} />
				{/snippet}
			</SettingRow>
		</SettingsCard>

		{#if showCimd}
			<SettingsCard title={m.client_id_metadata_documents()}>
				{#snippet description()}
					{m.config_cimd_description()}
					<DocsLink href="https://pocket-id.org/docs/guides/client-id-metadata-documents" />
				{/snippet}
				<fieldset class="contents" disabled={envOverrides.has('cimdUrlAllowlist')}>
					<Field.Field class="py-4">
						<div class="flex items-center gap-2">
							<Field.Label id="cimd-url-allowlist-label">{m.cimd_url_allowlist()}</Field.Label>
							{#if envOverrides.has('cimdUrlAllowlist')}
								<EnvOverrideLock />
							{/if}
						</div>
						<UrlListInput
							bind:urls={cimdUrlAllowlist}
							labelId="cimd-url-allowlist-label"
							testIdPrefix="cimd-url-allowlist"
							keepAtLeastOne
						/>
						<Field.Description>
							<FormattedMessage message={m.config_cimd_allowlist_description} />
						</Field.Description>
					</Field.Field>
				</fieldset>
			</SettingsCard>
		{/if}
	</fieldset>
</div>
