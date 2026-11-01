<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import FormInput from '#lib/components/form/form-input.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { createForm, pickSchemaValues } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import {
		LucideAlarmClock,
		LucideAtSign,
		LucideBadgeCheck,
		LucideBell,
		LucideCircleAlert,
		LucideHash,
		LucideLock,
		LucideMailCheck,
		LucideSend,
		LucideShieldOff,
		LucideTriangleAlert
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import OptionSelect from './option-select.svelte';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const appConfigService = new AppConfigService();
	const appName = $derived($appConfigStore.appName);

	const tlsOptions = [
		{ value: 'none' as const, label: m.none() },
		{ value: 'starttls' as const, label: 'StartTLS' },
		{ value: 'tls' as const, label: 'TLS' }
	];

	const notificationOptions = [
		{ value: 'disabled' as const, label: m.never() },
		{ value: 'always' as const, label: m.login_notification_always() },
		{
			value: 'ipAndUserAgent' as const,
			label: m.login_notification_ip_and_user_agent(),
			description: m.config_notify_ip_description()
		},
		{
			value: 'browserRecognition' as const,
			label: m.login_notification_browser_recognition(),
			description: m.config_notify_browser_description()
		}
	];

	const emailVerificationOptions = [
		{ value: 'disabled' as const, label: m.config_off() },
		{
			value: 'optional' as const,
			label: m.email_verification_optional(),
			description: m.config_verification_optional_description()
		},
		{
			value: 'required' as const,
			label: m.email_verification_required(),
			description: m.config_verification_required_description()
		}
	];

	let isSendingTestEmail = $state(false);

	const requiredSmtpFields = ['smtpHost', 'smtpPort', 'smtpFrom'] as const;

	function emailFeaturesOn(data: {
		emailLoginNotificationMode: string;
		emailVerification: string;
		emailOneTimeAccessAsUnauthenticatedEnabled: boolean;
		emailOneTimeAccessAsAdminEnabled: boolean;
		emailApiKeyExpirationEnabled: boolean;
	}) {
		return (
			data.emailLoginNotificationMode !== 'disabled' ||
			data.emailVerification !== 'disabled' ||
			data.emailOneTimeAccessAsUnauthenticatedEnabled ||
			data.emailOneTimeAccessAsAdminEnabled ||
			data.emailApiKeyExpirationEnabled
		);
	}

	const formSchema = z
		.object({
			requireUserEmail: z.boolean(),
			emailsVerified: z.boolean(),
			smtpHost: z.string().optional(),
			smtpPort: z
				.preprocess((v: string) => (!v ? undefined : parseInt(v)), z.number().optional().nullable())
				.transform((v) => v?.toString() ?? '')
				.optional(),
			smtpUser: z.string().optional(),
			smtpPassword: z.string().optional(),
			smtpFrom: z.email().or(z.literal('')).optional(),
			smtpTls: z.enum(['none', 'starttls', 'tls']),
			smtpSkipCertVerify: z.boolean(),
			emailOneTimeAccessAsUnauthenticatedEnabled: z.boolean(),
			emailVerification: z.enum(['disabled', 'optional', 'required']),
			emailOneTimeAccessAsAdminEnabled: z.boolean(),
			emailLoginNotificationMode: z.enum([
				'disabled',
				'always',
				'ipAndUserAgent',
				'browserRecognition'
			]),
			emailApiKeyExpirationEnabled: z.boolean()
		})
		.superRefine((data, ctx) => {
			function requireSmtpFieldsWhen(condition: boolean, message: string) {
				if (!condition) return;
				for (const field of requiredSmtpFields) {
					if (!data[field]) ctx.addIssue({ code: 'custom', path: [field], message });
				}
			}

			requireSmtpFieldsWhen(emailFeaturesOn(data), m.config_required_while_emails_on());
			requireSmtpFieldsWhen(
				requiredSmtpFields.some((field) => !!data[field]),
				m.config_required_with_mail_server()
			);

			// Requiring verification only protects accounts that have an email
			if (data.emailVerification === 'required' && !data.requireUserEmail) {
				ctx.addIssue({
					code: 'custom',
					path: ['emailVerification'],
					message: m.email_verification_required_needs_user_email()
				});
			}
		});

	const formStore = createForm(formSchema, pickSchemaValues(formSchema, appConfig));
	const inputs = formStore.inputs;

	const verificationRequired = $derived($inputs.emailVerification.value === 'required');
	const smtpMissing = $derived(
		emailFeaturesOn({
			emailLoginNotificationMode: $inputs.emailLoginNotificationMode.value,
			emailVerification: $inputs.emailVerification.value,
			emailOneTimeAccessAsUnauthenticatedEnabled:
				$inputs.emailOneTimeAccessAsUnauthenticatedEnabled.value,
			emailOneTimeAccessAsAdminEnabled: $inputs.emailOneTimeAccessAsAdminEnabled.value,
			emailApiKeyExpirationEnabled: $inputs.emailApiKeyExpirationEnabled.value
		}) && requiredSmtpFields.some((field) => !$inputs[field]?.value)
	);

	trackFormChanges(() => formStore, callback);

	// Saving from the "send test email" prompt is outside the unsaved-changes bar, so it reports failures itself rather than letting them reach the bar
	async function saveForTestEmail() {
		const data = formStore.validate();
		if (!data) return false;
		try {
			await callback(data);
			formStore.commit(data);
			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}

	function onTestEmail() {
		if (!formStore.isDirty()) {
			sendTestEmail();
			return;
		}
		openConfirmDialog({
			title: m.save_changes_question(),
			message: m.you_have_to_save_the_changes_before_sending_a_test_email_do_you_want_to_save_now(),
			confirm: {
				label: m.save_and_send(),
				action: async () => {
					if (await saveForTestEmail()) sendTestEmail();
				}
			}
		});
	}

	async function sendTestEmail() {
		isSendingTestEmail = true;
		await appConfigService
			.sendTestEmail()
			.then(() => toast.success(m.test_email_sent_successfully()))
			.catch(() => toast.error(m.failed_to_send_test_email()))
			.finally(() => (isSendingTestEmail = false));
	}
</script>

{#snippet verificationError()}
	<span class="text-danger-fg flex items-start gap-1.5">
		<LucideCircleAlert class="mt-[3px] size-3 shrink-0" />
		{$inputs.emailVerification.error}
	</span>
{/snippet}

<div class="flex min-w-0 flex-col gap-10">
	<SettingsSection title={m.config_email_addresses()}>
		<fieldset class="contents" disabled={$appConfigStore.uiConfigDisabled}>
			<SettingRow
				title={m.config_require_email()}
				description={m.config_require_email_description()}
				icon={LucideAtSign}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.requireUserEmail.value} />
				{/snippet}
			</SettingRow>
			<SettingRow
				title={m.email_verification()}
				description={$inputs.emailVerification.error ? verificationError : undefined}
				consequence={verificationRequired ? m.config_verification_required_note() : undefined}
				icon={LucideMailCheck}
			>
				{#snippet control(props)}
					<OptionSelect
						{...props}
						aria-invalid={!!$inputs.emailVerification.error}
						value={$inputs.emailVerification.value}
						options={emailVerificationOptions}
						onValueChange={(value) => {
							$inputs.emailVerification.value = value;
							// Addresses verified by default would let every new account skip the verification
							if (value === 'required') $inputs.emailsVerified.value = false;
						}}
					/>
				{/snippet}
			</SettingRow>
			<SettingRow
				title={m.config_treat_as_verified()}
				disabledReason={verificationRequired ? m.config_treat_as_verified_locked() : undefined}
				icon={LucideBadgeCheck}
			>
				{#snippet control(props)}
					<Switch
						{...props}
						disabled={verificationRequired}
						bind:checked={$inputs.emailsVerified.value}
					/>
				{/snippet}
			</SettingRow>
		</fieldset>
	</SettingsSection>

	<SettingsSection title={m.config_mail_server()} description={m.config_mail_server_description()}>
		<fieldset class="contents" disabled={$appConfigStore.uiConfigDisabled}>
			{#if smtpMissing}
				<Alert.Root variant="warning" class="mt-2">
					<LucideTriangleAlert />
					<Alert.Title>{m.config_smtp_missing_title()}</Alert.Title>
					<Alert.Description>{m.config_smtp_missing_description()}</Alert.Description>
				</Alert.Root>
			{/if}
			<div class="flex flex-col gap-4 py-4 shadow-[inset_0_-1px_0_var(--border)]">
				<FormInput
					class="max-w-[400px]"
					inputClass="font-mono"
					label={m.config_smtp_host()}
					placeholder="smtp.example.com"
					bind:input={$inputs.smtpHost}
				/>
				<FormInput
					class="max-w-[400px]"
					inputClass="max-w-30"
					label={m.config_smtp_port()}
					type="number"
					bind:input={$inputs.smtpPort}
				/>
				<FormInput
					class="max-w-[400px]"
					inputClass="font-mono"
					label={m.username()}
					optional
					bind:input={$inputs.smtpUser}
				/>
				<FormInput
					class="max-w-[400px]"
					inputClass="font-mono"
					label={m.config_password()}
					optional
					type="password"
					bind:input={$inputs.smtpPassword}
				/>
				<FormInput
					class="max-w-[400px]"
					inputClass="font-mono"
					label={m.config_from_address()}
					placeholder="id@example.com"
					bind:input={$inputs.smtpFrom}
				/>
			</div>
			<SettingRow title={m.config_encryption()} icon={LucideLock}>
				{#snippet control(props)}
					<OptionSelect
						{...props}
						value={$inputs.smtpTls.value}
						options={tlsOptions}
						onValueChange={(v) => ($inputs.smtpTls.value = v)}
					/>
				{/snippet}
			</SettingRow>
			<SettingRow
				title={m.skip_certificate_verification()}
				description={m.config_self_signed_only()}
				icon={LucideShieldOff}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.smtpSkipCertVerify.value} />
				{/snippet}
			</SettingRow>
		</fieldset>
		<!-- Sending changes nothing, so it stays available while the configuration is managed by environment variables -->
		<div class="flex pt-3">
			<Button size="sm" variant="secondary" isLoading={isSendingTestEmail} onclick={onTestEmail}>
				<LucideSend />
				{m.send_test_email()}
			</Button>
		</div>
	</SettingsSection>

	<SettingsSection title={m.config_emails_app_sends({ appName })}>
		<fieldset class="contents" disabled={$appConfigStore.uiConfigDisabled}>
			<SettingRow
				title={m.config_signin_notifications()}
				description={m.config_signin_notifications_description()}
				icon={LucideBell}
			>
				{#snippet control(props)}
					<OptionSelect
						{...props}
						value={$inputs.emailLoginNotificationMode.value}
						options={notificationOptions}
						onValueChange={(v) => ($inputs.emailLoginNotificationMode.value = v)}
					/>
				{/snippet}
			</SettingRow>
			<SettingRow
				title={m.config_admin_login_codes()}
				description={m.config_admin_login_codes_description()}
				icon={LucideSend}
			>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.emailOneTimeAccessAsAdminEnabled.value} />
				{/snippet}
			</SettingRow>
			<SettingRow title={m.config_api_key_expiry()} icon={LucideAlarmClock}>
				{#snippet control(props)}
					<Switch {...props} bind:checked={$inputs.emailApiKeyExpirationEnabled.value} />
				{/snippet}
			</SettingRow>
			<SettingRow
				title={m.config_user_login_codes()}
				description={m.config_user_login_codes_description()}
				consequence={$inputs.emailOneTimeAccessAsUnauthenticatedEnabled.value
					? m.config_user_login_codes_note()
					: undefined}
				icon={LucideHash}
			>
				{#snippet control(props)}
					<Switch
						{...props}
						bind:checked={$inputs.emailOneTimeAccessAsUnauthenticatedEnabled.value}
					/>
				{/snippet}
			</SettingRow>
		</fieldset>
	</SettingsSection>
</div>
