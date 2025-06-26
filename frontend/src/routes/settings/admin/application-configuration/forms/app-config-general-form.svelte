<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import AccentColorPicker from './accent-color-picker.svelte';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	let isLoading = $state(false);

	// Dynamic signup options with translations
	const signupOptions = {
		disabled: {
			label: m.signup_disabled(),
			description: m.signup_disabled_description()
		},
		withToken: {
			label: m.signup_with_token(),
			description: m.signup_with_token_description()
		},
		open: {
			label: m.signup_open(),
			description: m.signup_open_description()
		}
	};

	const updatedAppConfig = {
		appName: appConfig.appName,
		sessionDuration: appConfig.sessionDuration,
		emailsVerified: appConfig.emailsVerified,
		allowOwnAccountEdit: appConfig.allowOwnAccountEdit,
		allowUserSignups: appConfig.allowUserSignups,
		disableAnimations: appConfig.disableAnimations,
		accentColor: appConfig.accentColor
	};

	const formSchema = z.object({
		appName: z.string().min(2).max(30),
		sessionDuration: z.number().min(1).max(43200),
		emailsVerified: z.boolean(),
		allowOwnAccountEdit: z.boolean(),
		allowUserSignups: z.enum(['disabled', 'withToken', 'open']),
		disableAnimations: z.boolean(),
		accentColor: z.string()
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, updatedAppConfig);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;

		await callback(data).finally(() => (isLoading = false));
		toast.success(m.application_configuration_updated_successfully());
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset class="flex flex-col gap-5" disabled={$appConfigStore.uiConfigDisabled}>
		<div class="flex flex-col gap-5">
			<FormInput label={m.application_name()} bind:input={$inputs.appName} />
			<FormInput
				label={m.session_duration()}
				type="number"
				description={m.the_duration_of_a_session_in_minutes_before_the_user_has_to_sign_in_again()}
				bind:input={$inputs.sessionDuration}
			/>
			<div class="grid gap-2">
				<Label class="mb-0" for="enable-user-signup">{m.enable_user_signups()}</Label>
				<p class="text-muted-foreground text-[0.8rem]">
					{m.enable_user_signups_description()}
				</p>
				<Select.Root
					disabled={$appConfigStore.uiConfigDisabled}
					type="single"
					value={$inputs.allowUserSignups.value}
					onValueChange={(v) =>
						($inputs.allowUserSignups.value = v as typeof $inputs.allowUserSignups.value)}
				>
					<Select.Trigger
						class="w-full"
						aria-label="Enable user signups"
						placeholder={m.enable_user_signups()}
					>
						{signupOptions[$inputs.allowUserSignups.value]?.label}
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="disabled">
							<div class="flex flex-col items-start gap-1">
								<span class="font-medium">{signupOptions.disabled.label}</span>
								<span class="text-muted-foreground text-xs">
									{signupOptions.disabled.description}
								</span>
							</div>
						</Select.Item>
						<Select.Item value="withToken">
							<div class="flex flex-col items-start gap-1">
								<span class="font-medium">{signupOptions.withToken.label}</span>
								<span class="text-muted-foreground text-xs">
									{signupOptions.withToken.description}
								</span>
							</div>
						</Select.Item>
						<Select.Item value="open">
							<div class="flex flex-col items-start gap-1">
								<span class="font-medium">{signupOptions.open.label}</span>
								<span class="text-muted-foreground text-xs">
									{signupOptions.open.description}
								</span>
							</div>
						</Select.Item>
					</Select.Content>
				</Select.Root>
			</div>
			<SwitchWithLabel
				id="self-account-editing"
				label={m.enable_self_account_editing()}
				description={m.whether_the_users_should_be_able_to_edit_their_own_account_details()}
				bind:checked={$inputs.allowOwnAccountEdit.value}
			/>

			<SwitchWithLabel
				id="emails-verified"
				label={m.emails_verified()}
				description={m.whether_the_users_email_should_be_marked_as_verified_for_the_oidc_clients()}
				bind:checked={$inputs.emailsVerified.value}
			/>
			<SwitchWithLabel
				id="disable-animations"
				label={m.disable_animations()}
				description={m.turn_off_ui_animations()}
				bind:checked={$inputs.disableAnimations.value}
			/>

			<div class="space-y-5">
				<div>
					<Label class="mb-0 text-sm font-medium">
						{m.accent_color()}
					</Label>
					<p class="text-muted-foreground text-[0.8rem]">
						{m.select_an_accent_color_to_customize_the_appearance_of_pocket_id()}
					</p>
				</div>
				<AccentColorPicker
					previousColor={appConfig.accentColor}
					bind:selectedColor={$inputs.accentColor.value}
				/>
			</div>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
