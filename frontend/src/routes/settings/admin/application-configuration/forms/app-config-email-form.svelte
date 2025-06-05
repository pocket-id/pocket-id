<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import CheckboxWithLabel from '$lib/components/form/checkbox-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import Label from '$lib/components/ui/label/label.svelte';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';

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

	const formSchema = z.object({
		smtpHost: z.string().min(1),
		smtpPort: z.number().min(1),
		smtpUser: z.string(),
		smtpPassword: z.string(),
		smtpFrom: z.string().email(),
		smtpTls: z.enum(['none', 'starttls', 'tls']),
		smtpSkipCertVerify: z.boolean(),
		emailOneTimeAccessAsUnauthenticatedEnabled: z.boolean(),
		emailOneTimeAccessAsAdminEnabled: z.boolean(),
		emailLoginNotificationEnabled: z.boolean(),
		emailApiKeyExpirationEnabled: z.boolean()
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, appConfig);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return false;
		await callback(data);

		// Update the app config to don't display the unsaved changes warning
		Object.entries(data).forEach(([key, value]) => {
			// @ts-ignore
			appConfig[key] = value;
		});

		toast.success(m.email_configuration_updated_successfully());
		return true;
	}
	async function onTestEmail() {
		// @ts-ignore
		const hasChanges = Object.keys($inputs).some((key) => $inputs[key].value !== appConfig[key]);

		if (hasChanges) {
			openConfirmDialog({
				title: m.save_changes_question(),
				message:
					m.you_have_to_save_the_changes_before_sending_a_test_email_do_you_want_to_save_now(),
				confirm: {
					label: m.save_and_send(),
					action: async () => {
						const saved = await onSubmit();
						if (saved) {
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

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={$appConfigStore.uiConfigDisabled}>
		<h4 class="text-lg font-semibold">{m.smtp_configuration()}</h4>
		<div class="mt-4 grid grid-cols-1 items-end gap-5 md:grid-cols-2">
			<FormInput label={m.smtp_host()} bind:input={$inputs.smtpHost} />
			<FormInput label={m.smtp_port()} type="number" bind:input={$inputs.smtpPort} />
			<FormInput label={m.smtp_user()} bind:input={$inputs.smtpUser} />
			<FormInput label={m.smtp_password()} type="password" bind:input={$inputs.smtpPassword} />
			<FormInput label={m.smtp_from()} bind:input={$inputs.smtpFrom} />
			<div class="grid gap-2">
				<Label class="mb-0" for="smtp-tls">{m.smtp_tls_option()}</Label>
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
			</div>
			<CheckboxWithLabel
				id="skip-cert-verify"
				label={m.skip_certificate_verification()}
				description={m.this_can_be_useful_for_selfsigned_certificates()}
				bind:checked={$inputs.smtpSkipCertVerify.value}
			/>
		</div>
		<h4 class="mt-10 text-lg font-semibold">{m.enabled_emails()}</h4>
		<div class="mt-4 flex flex-col gap-5">
			<CheckboxWithLabel
				id="email-login-notification"
				label={m.email_login_notification()}
				description={m.send_an_email_to_the_user_when_they_log_in_from_a_new_device()}
				bind:checked={$inputs.emailLoginNotificationEnabled.value}
			/>

			<CheckboxWithLabel
				id="email-login-admin"
				label={m.email_login_code_from_admin()}
				description={m.allows_an_admin_to_send_a_login_code_to_the_user()}
				bind:checked={$inputs.emailOneTimeAccessAsAdminEnabled.value}
			/>
			<CheckboxWithLabel
				id="api-key-expiration"
				label={m.api_key_expiration()}
				description={m.send_an_email_to_the_user_when_their_api_key_is_about_to_expire()}
				bind:checked={$inputs.emailApiKeyExpirationEnabled.value}
			/>
			<CheckboxWithLabel
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
		<Button type="submit" disabled={$appConfigStore.uiConfigDisabled}>{m.save()}</Button>
	</div>
</form>
