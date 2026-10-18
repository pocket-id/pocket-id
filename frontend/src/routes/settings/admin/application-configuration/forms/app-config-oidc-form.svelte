<script lang="ts">
	import CopyButton from '#lib/components/copy-button.svelte';
	import DocsLink from '#lib/components/docs-link.svelte';
	import UrlListInput from '#lib/components/form/url-list-input.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import type { OidcDiscoveryConfiguration } from '#lib/types/oidc.type.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges, trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { LucideKey } from '@lucide/svelte';
	import { z } from 'zod/v4';

	let {
		appConfig,
		oidcConfiguration,
		showCimd,
		callback
	}: {
		appConfig: AllAppConfig;
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
	];

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

<div class="flex min-w-0 flex-col gap-10">
	<SettingsSection title={m.config_endpoints()}>
		<dl class="flex flex-col py-2">
			{#each endpoints as endpoint (endpoint.label)}
				<div
					class="grid min-h-11 items-center gap-x-4 gap-y-0.5 py-1.5 sm:grid-cols-[140px_minmax(0,1fr)]"
				>
					<dt class="text-caption text-fg-muted">{endpoint.label}</dt>
					<dd class="flex min-w-0 items-center gap-1">
						<span class="text-mono-s font-mono break-all">{endpoint.value}</span>
						<CopyButton
							value={endpoint.value}
							label={m.config_copy_endpoint({ name: endpoint.label })}
						/>
					</dd>
				</div>
			{/each}
		</dl>
	</SettingsSection>

	<fieldset class="contents" disabled={$appConfigStore.uiConfigDisabled}>
		<SettingsSection title={m.config_new_clients()}>
			<SettingRow
				title={m.config_auto_secret()}
				description={m.config_auto_secret_description()}
				icon={LucideKey}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.autoCreateOidcClientSecret.value} />
				{/snippet}
			</SettingRow>
		</SettingsSection>

		{#if showCimd}
			<SettingsSection title={m.client_id_metadata_documents()}>
				{#snippet description()}
					{m.config_cimd_description()}
					<DocsLink href="https://pocket-id.org/docs/guides/client-id-metadata-documents" />
				{/snippet}
				<Field.Field class="pt-4">
					<Field.Label id="cimd-url-allowlist-label">{m.cimd_url_allowlist()}</Field.Label>
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
			</SettingsSection>
		{/if}
	</fieldset>
</div>
