<script lang="ts">
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsCard from '#lib/components/settings-card.svelte';
	import { Switch } from '@pocket-id/ui/switch';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { createForm, pickSchemaValues } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import FingerprintIcon from '~icons/solar/face-scan-square-bold-duotone';
	import KeyRoundIcon from '~icons/solar/key-minimalistic-bold-duotone';
	import RefreshCwIcon from '~icons/solar/refresh-bold-duotone';
	import ShieldCheckIcon from '~icons/solar/shield-check-bold-duotone';
	import { z } from 'zod/v4';
	import OptionSelect from './option-select.svelte';

	let {
		callback,
		appConfig,
		envOverrides
	}: {
		appConfig: AllAppConfig;
		envOverrides: Set<keyof AllAppConfig>;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const formSchema = z.object({
		webauthnUserVerification: z.enum(['required', 'preferred']),
		webauthnAllowSyncedPasskeys: z.boolean(),
		webauthnAuthenticatorAttachment: z.enum(['any', 'platform', 'cross-platform']),
		webauthnAttestation: z.enum(['none', 'verified', 'l1', 'l2', 'l3'])
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

	const attestationOptions = [
		{
			value: 'none' as const,
			label: m.config_attestation_none(),
			description: m.config_attestation_none_description()
		},
		{
			value: 'verified' as const,
			label: m.config_attestation_verified(),
			description: m.config_attestation_verified_description()
		},
		{
			value: 'l1' as const,
			label: m.config_attestation_l1(),
			description: m.config_attestation_l1_description()
		},
		{
			value: 'l2' as const,
			label: m.config_attestation_l2(),
			description: m.config_attestation_l2_description()
		},
		{
			value: 'l3' as const,
			label: m.config_attestation_l3(),
			description: m.config_attestation_l3_description()
		}
	];

	const formStore = createForm(formSchema, pickSchemaValues(formSchema, appConfig));
	const inputs = formStore.inputs;

	const attestationRequired = $derived($inputs.webauthnAttestation.value !== 'none');

	trackFormChanges(() => formStore, callback);
</script>

<fieldset class="flex min-w-0 flex-col gap-4" disabled={$appConfigStore.uiConfigDisabled}>
	<SettingsCard title={m.config_registration()} description={m.config_registration_description()}>
		<SettingRow
			title={m.user_verification()}
			description={m.config_user_verification_description()}
			icon={FingerprintIcon}
			envOverride={envOverrides.has('webauthnUserVerification')}
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
			disabledReason={attestationRequired ? m.config_allow_synced_passkeys_locked() : undefined}
			icon={RefreshCwIcon}
			envOverride={envOverrides.has('webauthnAllowSyncedPasskeys')}
		>
			{#snippet control(props)}
				<Switch
					{...props}
					disabled={attestationRequired}
					bind:checked={$inputs.webauthnAllowSyncedPasskeys.value}
				/>
			{/snippet}
		</SettingRow>
		<SettingRow
			title={m.allowed_authenticator_type()}
			icon={KeyRoundIcon}
			envOverride={envOverrides.has('webauthnAuthenticatorAttachment')}
		>
			{#snippet control(props)}
				<OptionSelect
					{...props}
					value={$inputs.webauthnAuthenticatorAttachment.value}
					options={authenticatorAttachmentOptions}
					onValueChange={(v) => ($inputs.webauthnAuthenticatorAttachment.value = v)}
				/>
			{/snippet}
		</SettingRow>
		<SettingRow
			title={m.authenticator_verification()}
			description={m.config_authenticator_verification_description()}
			icon={ShieldCheckIcon}
			envOverride={envOverrides.has('webauthnAttestation')}
		>
			{#snippet control(props)}
				<OptionSelect
					{...props}
					value={$inputs.webauthnAttestation.value}
					options={attestationOptions}
					onValueChange={(v) => {
						$inputs.webauthnAttestation.value = v;
						// Synced passkeys send no attestation, so verification already rejects them
						if (v !== 'none') $inputs.webauthnAllowSyncedPasskeys.value = false;
					}}
				/>
			{/snippet}
		</SettingRow>
	</SettingsCard>
</fieldset>
