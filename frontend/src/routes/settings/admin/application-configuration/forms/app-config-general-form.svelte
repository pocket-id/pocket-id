<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
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

	const homePageUrlOptions = [
		{ label: m.my_account(), value: '/settings/account' },
		{ label: m.my_apps(), value: '/settings/apps' }
	];

	const updatedAppConfig = {
		appName: appConfig.appName,
		homePageUrl: appConfig.homePageUrl,
		sessionDuration: appConfig.sessionDuration,
		emailsVerified: appConfig.emailsVerified,
		allowOwnAccountEdit: appConfig.allowOwnAccountEdit,
		disableAnimations: appConfig.disableAnimations,
		accentColor: appConfig.accentColor
	};

	const formSchema = z.object({
		appName: z.string().min(2).max(30),
		homePageUrl: z.string(),
		sessionDuration: z.number().min(1).max(43200),
		emailsVerified: z.boolean(),
		allowOwnAccountEdit: z.boolean(),
		disableAnimations: z.boolean(),
		accentColor: z.string()
	});

	let { inputs, ...form } = $derived(createForm(formSchema, updatedAppConfig));

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
			<Field.Field>
				<Field.Label>{m.app_config_home_page()}</Field.Label>
				<Field.Description>
					{m.app_config_home_page_description()}
				</Field.Description>
				<Select.Root
					type="single"
					value={$inputs.homePageUrl.value}
					onValueChange={(v) => ($inputs.homePageUrl.value = v as string)}
				>
					<Select.Trigger
						class="w-full"
						aria-label={m.app_config_home_page()}
					>
						{homePageUrlOptions.find((option) => option.value === $inputs.homePageUrl.value)
							?.label ?? $inputs.homePageUrl.value}
					</Select.Trigger>
					<Select.Content>
						{#each homePageUrlOptions as option}
							<Select.Item value={option.value}>
								{option.label}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</Field.Field>
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

			<Field.Field class="space-y-5">
				<div>
					<Field.Label>
						{m.accent_color()}
					</Field.Label>
					<Field.Description>
						{m.select_an_accent_color_to_customize_the_appearance_of_pocket_id()}
					</Field.Description>
				</div>
				<AccentColorPicker
					previousColor={appConfig.accentColor}
					bind:selectedColor={$inputs.accentColor.value}
					disabled={$appConfigStore.uiConfigDisabled}
				/>
			</Field.Field>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
