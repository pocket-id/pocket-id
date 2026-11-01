<script lang="ts">
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { createForm, pickSchemaValues } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { LucideFingerprint, LucideKeyRound, LucideRefreshCw } from '@lucide/svelte';
	import { z } from 'zod/v4';
	import OptionSelect from './option-select.svelte';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const formSchema = z.object({
		webauthnUserVerification: z.enum(['required', 'preferred']),
		webauthnAllowSyncedPasskeys: z.boolean(),
		webauthnAuthenticatorAttachment: z.enum(['any', 'platform', 'cross-platform'])
	});

	const userVerificationOptions = [
		{
			value: 'required' as const,
			label: m.user_verification_required(),
			description: m.config_uv_required_description()
		},
		{
			value: 'preferred' as const,
			label: m.user_verification_preferred(),
			description: m.config_uv_preferred_description()
		}
	];

	const authenticatorAttachmentOptions = [
		{
			value: 'any' as const,
			label: m.any_authenticator(),
			description: m.config_any_passkey_description()
		},
		{
			value: 'platform' as const,
			label: m.device_passkeys_only(),
			description: m.config_device_passkeys_description()
		},
		{
			value: 'cross-platform' as const,
			label: m.config_security_keys_only(),
			description: m.config_security_keys_only_description()
		}
	];

	const formStore = createForm(formSchema, pickSchemaValues(formSchema, appConfig));
	const inputs = formStore.inputs;

	trackFormChanges(() => formStore, callback);
</script>

<fieldset class="flex min-w-0 flex-col gap-10" disabled={$appConfigStore.uiConfigDisabled}>
	<SettingsSection
		title={m.config_registration()}
		description={m.config_registration_description()}
	>
		<SettingRow
			title={m.user_verification()}
			description={m.config_user_verification_description()}
			icon={LucideFingerprint}
		>
			{#snippet control(props)}
				<OptionSelect
					{...props}
					value={$inputs.webauthnUserVerification.value}
					options={userVerificationOptions}
					onValueChange={(v) => ($inputs.webauthnUserVerification.value = v)}
				/>
			{/snippet}
		</SettingRow>
		<SettingRow
			title={m.allow_synced_passkeys()}
			description={m.config_allow_synced_passkeys_description()}
			consequence={$inputs.webauthnAllowSyncedPasskeys.value
				? undefined
				: m.config_synced_passkeys_off_note()}
			icon={LucideRefreshCw}
		>
			{#snippet control(props)}
				<Switch {...props} bind:checked={$inputs.webauthnAllowSyncedPasskeys.value} />
			{/snippet}
		</SettingRow>
		<SettingRow title={m.allowed_authenticator_type()} icon={LucideKeyRound}>
			{#snippet control(props)}
				<OptionSelect
					{...props}
					value={$inputs.webauthnAuthenticatorAttachment.value}
					options={authenticatorAttachmentOptions}
					onValueChange={(v) => ($inputs.webauthnAuthenticatorAttachment.value = v)}
				/>
			{/snippet}
		</SettingRow>
	</SettingsSection>
</fieldset>
