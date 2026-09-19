<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { createForm, pickSchemaValues } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const appConfigService = new AppConfigService();
	const tlsOptions = {
		none: 'None',
		starttls: 'StartTLS',
		tls: 'TLS'
	};

	let isSendingTestEmail = $state(false);

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
			emailVerificationEnabled: z.boolean(),
			emailOneTimeAccessAsAdminEnabled: z.boolean(),
			emailLoginNotificationEnabled: z.boolean(),
			emailApiKeyExpirationEnabled: z.boolean()
		})
		.superRefine((data, ctx) => {
			const requiredSmtpFields: (keyof z.infer<typeof formSchema>)[] = [
				'smtpHost',
				'smtpPort',
				'smtpFrom'
			];

			const emailFields: (keyof z.infer<typeof formSchema>)[] = [
				'emailOneTimeAccessAsUnauthenticatedEnabled',
				'emailVerificationEnabled',
				'emailOneTimeAccessAsAdminEnabled',
				'emailLoginNotificationEnabled',
				'emailApiKeyExpirationEnabled'
			];

			function requireFieldsWhen(condition: boolean, message: string) {
				if (!condition) return;

				for (const f of requiredSmtpFields) {
					if (!data[f]) {
						ctx.addIssue({
							code: 'custom',
							path: [f],
							message
						});
					}
				}
			}

			const anyProvided = requiredSmtpFields.some((f) => !!data[f]);
			requireFieldsWhen(anyProvided, m.smtp_field_required_when_other_provided());

			const emailEnabled = emailFields.some((f) => data[f]);
			requireFieldsWhen(emailEnabled, m.smtp_field_required_when_email_enabled());
		});

	const formStore = createForm(formSchema, pickSchemaValues(formSchema, appConfig));
	const inputs = formStore.inputs;

	async function saveEmailConfig(data: z.infer<typeof formSchema>) {
		await callback(data);
	}

	// Saving from the "send test email" prompt is outside the unsaved-changes bar, so it
	// reports failures itself rather than letting them reach the bar.
	async function saveForTestEmail() {
		const data = formStore.validate();
		if (!data) return false;
		try {
			await saveEmailConfig(data);
			formStore.commit(data);
			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}

	trackFormChanges(() => formStore, saveEmailConfig);

	async function onTestEmail() {
		const hasChanges = formStore.isDirty();

		if (hasChanges) {
			openConfirmDialog({
				title: m.save_changes_question(),
				message:
					m.you_have_to_save_the_changes_before_sending_a_test_email_do_you_want_to_save_now(),
				confirm: {
					label: m.save_and_send(),
					action: async () => {
						if (await saveForTestEmail()) {
							sendTestEmail();
						}
					}
				}
			});
		} else {
			sendTestEmail();
		}
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

<div>
	<fieldset disabled={$appConfigStore.uiConfigDisabled}>
		<h4 class="mb-4 text-lg font-semibold">{m.general()}</h4>
		<div class="flex flex-col gap-5">
			<SwitchWithLabel
				id="require-user-email"
				label={m.require_user_email()}
				description={m.require_user_email_description()}
				bind:checked={$inputs.requireUserEmail.value}
			/>
			<SwitchWithLabel
				id="emails-verified-by-default"
				label={m.emails_verified_by_default()}
				description={m.emails_verified_by_default_description()}
				bind:checked={$inputs.emailsVerified.value}
			/>
		</div>
		<h4 class="mt-10 text-lg font-semibold">{m.smtp_configuration()}</h4>
		<div class="mt-4 grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput label={m.smtp_host()} bind:input={$inputs.smtpHost} />
			<FormInput label={m.smtp_port()} type="number" bind:input={$inputs.smtpPort} />
			<FormInput label={m.smtp_user()} bind:input={$inputs.smtpUser} />
			<FormInput label={m.smtp_password()} type="password" bind:input={$inputs.smtpPassword} />
			<FormInput label={m.smtp_from()} bind:input={$inputs.smtpFrom} />
			<Field.Field>
				<Field.Label for="smtp-tls">{m.smtp_tls_option()}</Field.Label>
				<Select.Root
					type="single"
					value={$inputs.smtpTls.value}
					onValueChange={(v) => ($inputs.smtpTls.value = v as typeof $inputs.smtpTls.value)}
				>
					<Select.Trigger class="w-full" placeholder={m.email_tls_option()}>
						{tlsOptions[$inputs.smtpTls.value]}
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="none" label="None" />
						<Select.Item value="starttls" label="StartTLS" />
						<Select.Item value="tls" label="TLS" />
					</Select.Content>
				</Select.Root>
			</Field.Field>
			<SwitchWithLabel
				id="skip-cert-verify"
				label={m.skip_certificate_verification()}
				description={m.this_can_be_useful_for_selfsigned_certificates()}
				bind:checked={$inputs.smtpSkipCertVerify.value}
			/>
		</div>
		<h4 class="mt-10 text-lg font-semibold">{m.enabled_emails()}</h4>
		<div class="mt-4 flex flex-col gap-5">
			<SwitchWithLabel
				id="email-login-notification"
				label={m.email_login_notification()}
				description={m.send_an_email_to_the_user_when_they_log_in_from_a_new_device()}
				bind:checked={$inputs.emailLoginNotificationEnabled.value}
			/>
			<SwitchWithLabel
				id="email-verification"
				label={m.email_verification()}
				description={m.email_verification_description()}
				bind:checked={$inputs.emailVerificationEnabled.value}
			/>
			<SwitchWithLabel
				id="email-login-admin"
				label={m.email_login_code_from_admin()}
				description={m.allows_an_admin_to_send_a_login_code_to_the_user()}
				bind:checked={$inputs.emailOneTimeAccessAsAdminEnabled.value}
			/>
			<SwitchWithLabel
				id="api-key-expiration"
				label={m.api_key_expiration()}
				description={m.send_an_email_to_the_user_when_their_api_key_is_about_to_expire()}
				bind:checked={$inputs.emailApiKeyExpirationEnabled.value}
			/>
			<SwitchWithLabel
				id="email-login-user"
				label={m.emai_login_code_requested_by_user()}
				description={m.allow_users_to_sign_in_with_a_login_code_sent_to_their_email()}
				bind:checked={$inputs.emailOneTimeAccessAsUnauthenticatedEnabled.value}
			/>
		</div>
	</fieldset>
	<div class="mt-8 flex flex-wrap justify-end gap-3">
		<Button isLoading={isSendingTestEmail} variant="secondary" onclick={onTestEmail}
			>{m.send_test_email()}</Button
		>
	</div>
</div>
