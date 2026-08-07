<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import * as Select from '$lib/components/ui/select';
	import { Switch } from '$lib/components/ui/switch';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let isLoading = $state(false);

	const formSchema = z.object({
		webauthnUserVerification: z.enum(['required', 'preferred']),
		webauthnAllowSyncedPasskeys: z.boolean(),
		webauthnAuthenticatorAttachment: z.enum(['any', 'platform', 'cross-platform'])
	});

	const initialConfig = {
		webauthnUserVerification: appConfig.webauthnUserVerification,
		webauthnAllowSyncedPasskeys: appConfig.webauthnAllowSyncedPasskeys,
		webauthnAuthenticatorAttachment: appConfig.webauthnAuthenticatorAttachment
	};

	const userVerificationOptions = {
		required: {
			label: m.user_verification_required(),
			description: m.user_verification_required_description()
		},
		preferred: {
			label: m.user_verification_preferred(),
			description: m.user_verification_preferred_description()
		}
	};

	const authenticatorAttachmentOptions = {
		any: {
			label: m.any_authenticator(),
			description: m.any_authenticator_description()
		},
		platform: {
			label: m.device_passkeys_only(),
			description: m.device_passkeys_only_description()
		},
		'cross-platform': {
			label: m.external_security_keys_only(),
			description: m.external_security_keys_only_description()
		}
	};

	let { inputs, ...form } = $derived(createForm(formSchema, initialConfig));

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;

		isLoading = true;
		try {
			await callback(data);
			toast.success(m.passkey_configuration_updated_successfully());
		} finally {
			isLoading = false;
		}
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={$appConfigStore.uiConfigDisabled}>
		<Field.Group>
			<Field.Field data-invalid={!!$inputs.webauthnUserVerification.error}>
				<div>
					<Field.Label for="passkey-user-verification">{m.user_verification()}</Field.Label>
					<Field.Description>{m.user_verification_description()}</Field.Description>
				</div>
				<Select.Root
					type="single"
					value={$inputs.webauthnUserVerification.value}
					onValueChange={(value) =>
						($inputs.webauthnUserVerification.value = value as 'required' | 'preferred')}
				>
					<Select.Trigger
						id="passkey-user-verification"
						class="w-full"
						aria-label={m.user_verification()}
						aria-invalid={!!$inputs.webauthnUserVerification.error}
						placeholder={m.user_verification()}
					>
						{userVerificationOptions[$inputs.webauthnUserVerification.value]?.label}
					</Select.Trigger>
					<Select.Content>
						<Select.Group>
							{#each Object.entries(userVerificationOptions) as [value, option] (value)}
								<Select.Item {value} label={option.label}>
									<div class="flex flex-col items-start gap-1">
										<span class="font-medium">{option.label}</span>
										<span class="text-muted-foreground text-xs">{option.description}</span>
									</div>
								</Select.Item>
							{/each}
						</Select.Group>
					</Select.Content>
				</Select.Root>
			</Field.Field>

			<Field.Field orientation="horizontal">
				<Field.Content>
					<div>
						<Field.Label for="allow-synced-passkeys">{m.allow_synced_passkeys()}</Field.Label>
						<Field.Description>{m.allow_synced_passkeys_description()}</Field.Description>
					</div>
				</Field.Content>
				<Switch
					id="allow-synced-passkeys"
					class="my-auto"
					bind:checked={$inputs.webauthnAllowSyncedPasskeys.value}
				/>
			</Field.Field>

			<Field.Field data-invalid={!!$inputs.webauthnAuthenticatorAttachment.error}>
				<div>
					<Field.Label for="passkey-authenticator-type"
						>{m.allowed_authenticator_type()}</Field.Label
					>
					<Field.Description>{m.allowed_authenticator_type_description()}</Field.Description>
				</div>
				<Select.Root
					type="single"
					value={$inputs.webauthnAuthenticatorAttachment.value}
					onValueChange={(value) =>
						($inputs.webauthnAuthenticatorAttachment.value = value as
							'any' | 'platform' | 'cross-platform')}
				>
					<Select.Trigger
						id="passkey-authenticator-type"
						class="w-full"
						aria-label={m.allowed_authenticator_type()}
						aria-invalid={!!$inputs.webauthnAuthenticatorAttachment.error}
						placeholder={m.allowed_authenticator_type()}
					>
						{authenticatorAttachmentOptions[$inputs.webauthnAuthenticatorAttachment.value]?.label}
					</Select.Trigger>
					<Select.Content>
						<Select.Group>
							{#each Object.entries(authenticatorAttachmentOptions) as [value, option] (value)}
								<Select.Item {value} label={option.label}>
									<div class="flex flex-col items-start gap-1">
										<span class="font-medium">{option.label}</span>
										<span class="text-muted-foreground text-xs">{option.description}</span>
									</div>
								</Select.Item>
							{/each}
						</Select.Group>
					</Select.Content>
				</Select.Root>
			</Field.Field>

			<Field.Field orientation="horizontal" class="justify-end">
				<Button type="submit" disabled={$appConfigStore.uiConfigDisabled} {isLoading}>
					{m.save()}
				</Button>
			</Field.Field>
		</Field.Group>
	</fieldset>
</form>
