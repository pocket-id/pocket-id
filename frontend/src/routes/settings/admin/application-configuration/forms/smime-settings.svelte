<script lang="ts">
	import { onMount } from 'svelte';
	import { z } from 'zod/v4';
	import { toast } from 'svelte-sonner';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import FormInput from '#lib/components/form/form-input.svelte';
	import SwitchWithLabel from '#lib/components/form/switch-with-label.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { SigningStatus } from '#lib/types/smime.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';

	let { emailSettingsDirty, sender }: { emailSettingsDirty: boolean; sender: string } = $props();
	const service = new AppConfigService();
	const passwordForm = createForm(z.object({ password: z.string().default('') }), { password: '' });
	const passwordInputs = passwordForm.inputs;
	let status = $state<SigningStatus | null>(null);
	let busy = $state(false);
	let loading = $state(true);
	let files = $state<FileList>();
	let fileInput = $state<HTMLInputElement | null>(null);
	let enabled = $state(false);
	const disabled = $derived(
		busy || loading || !status || emailSettingsDirty || $appConfigStore.uiConfigDisabled
	);
	const statusLabels = $derived({
		valid: m.smime_valid(),
		expiringSoon: m.smime_expiring_soon(),
		expired: m.smime_expired(),
		notYetValid: m.smime_not_yet_valid()
	});
	const matchesSender = $derived(
		status?.certificate?.emails.some((email) => email.toLowerCase() === sender.toLowerCase()) ??
			false
	);

	function applyStatus(value: SigningStatus) {
		status = value;
		enabled = value.enabled;
	}
	async function load() {
		loading = true;
		try {
			applyStatus(await service.getSigningStatus());
		} catch (error) {
			axiosErrorToast(error);
		} finally {
			loading = false;
		}
	}
	onMount(() => {
		void load();
	});

	async function upload() {
		const file = files?.[0];
		if (!file) return;
		if (file.size > 1000000) {
			toast.error(m.smime_file_too_large());
			return;
		}
		busy = true;
		try {
			// Preserve password whitespace because PKCS#12 passwords are exact values
			applyStatus(
				await service.uploadSigningCertificate(file, $passwordInputs.password.value ?? '')
			);
			passwordForm.setValue('password', '');
			if (fileInput) fileInput.value = '';
			files = undefined;
			toast.success(m.smime_certificate_saved());
		} catch (error) {
			axiosErrorToast(error);
		} finally {
			busy = false;
		}
	}
	async function setEnabled(value: boolean) {
		if (disabled) {
			enabled = status?.enabled ?? false;
			return;
		}
		busy = true;
		try {
			applyStatus(await service.setSigningEnabled(value));
		} catch (error) {
			enabled = status?.enabled ?? false;
			axiosErrorToast(error);
		} finally {
			busy = false;
		}
	}
	function remove() {
		openConfirmDialog({
			title: m.smime_remove_certificate(),
			message: m.smime_remove_confirmation(),
			confirm: {
				label: m.smime_remove_certificate(),
				destructive: true,
				action: async () => {
					busy = true;
					try {
						applyStatus(await service.removeSigningCertificate());
					} catch (error) {
						axiosErrorToast(error);
					} finally {
						busy = false;
					}
				}
			}
		});
	}
</script>

<section class="mt-10" aria-labelledby="smime-heading">
	<h4 id="smime-heading" class="text-lg font-semibold">{m.smime_title()}</h4>
	<p class="text-muted-foreground mt-2 text-sm">{m.smime_description()}</p>
	{#if loading}
		<p class="mt-4 text-sm" role="status">{m.smime_loading()}</p>
	{:else if !status}
		<Button class="mt-4" variant="secondary" onclick={load}>{m.smime_reload()}</Button>
	{:else}
		<fieldset {disabled} class="mt-4 flex flex-col gap-5">
			<SwitchWithLabel
				id="smime-enabled"
				label={m.smime_enable()}
				description={m.smime_enable_description()}
				bind:checked={enabled}
				onCheckedChange={setEnabled}
				disabled={disabled ||
					(!status.enabled &&
						(!status.certificate ||
							!matchesSender ||
							status.certificate.status === 'expired' ||
							status.certificate.status === 'notYetValid'))}
			/>
			{#if status.certificate}
				<div class="rounded-xl border p-4">
					<dl class="grid grid-cols-1 gap-3 text-sm md:grid-cols-[auto_1fr]">
						<dt class="font-medium">{m.smime_status()}</dt>
						<dd>{statusLabels[status.certificate.status]}</dd>
						<dt class="font-medium">{m.smime_subject()}</dt>
						<dd class="break-all">{status.certificate.subject}</dd>
						<dt class="font-medium">{m.smime_issuer()}</dt>
						<dd class="break-all">{status.certificate.issuer}</dd>
						<dt class="font-medium">{m.email()}</dt>
						<dd class="break-all">{status.certificate.emails.join(', ')}</dd>
						<dt class="font-medium">{m.smime_valid_from()}</dt>
						<dd>{new Date(status.certificate.notBefore).toLocaleString()}</dd>
						<dt class="font-medium">{m.smime_valid_until()}</dt>
						<dd>{new Date(status.certificate.notAfter).toLocaleString()}</dd>
						<dt class="font-medium">{m.smime_fingerprint()}</dt>
						<dd class="font-mono text-xs break-all">{status.certificate.fingerprint}</dd>
					</dl>
					{#if !matchesSender}<p class="text-destructive mt-3 text-sm" role="alert">
							{m.smime_sender_mismatch()}
						</p>{/if}
					{#if status.enabled && (status.certificate.status === 'expired' || status.certificate.status === 'notYetValid' || !matchesSender)}
						<p class="text-destructive mt-3 text-sm" role="alert">{m.smime_delivery_blocked()}</p>
					{/if}
					<p class="text-muted-foreground mt-3 text-xs">{m.smime_trust_note()}</p>
				</div>
			{:else}
				<p class="text-muted-foreground text-sm">{m.smime_no_certificate()}</p>
			{/if}
			<div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
				<Field.Field>
					<Field.Label for="smime-file">{m.smime_certificate_file()}</Field.Label>
					<Field.Description>{m.smime_upload_description()}</Field.Description>
					<Input id="smime-file" type="file" accept=".p12,.pfx" bind:files bind:ref={fileInput} />
				</Field.Field>
				<FormInput
					label={m.smime_password()}
					type="password"
					bind:input={$passwordInputs.password}
					description={m.smime_password_description()}
				/>
			</div>
			<div class="flex flex-wrap gap-3">
				<Button
					variant="secondary"
					isLoading={busy}
					disabled={disabled || !files?.length}
					onclick={upload}
				>
					{status.certificate ? m.smime_replace_certificate() : m.smime_upload_certificate()}
				</Button>
				{#if status.certificate}<Button variant="destructive" {disabled} onclick={remove}
						>{m.smime_remove_certificate()}</Button
					>{/if}
			</div>
		</fieldset>
		{#if emailSettingsDirty}<p class="text-muted-foreground mt-3 text-sm">
				{m.smime_save_email_settings()}
			</p>{/if}
	{/if}
</section>
