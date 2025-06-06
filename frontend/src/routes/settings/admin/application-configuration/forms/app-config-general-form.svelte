<script lang="ts">
	import CheckboxWithLabel from '$lib/components/form/checkbox-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
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

	const updatedAppConfig = {
		appName: appConfig.appName,
		sessionDuration: appConfig.sessionDuration,
		emailsVerified: appConfig.emailsVerified,
		allowOwnAccountEdit: appConfig.allowOwnAccountEdit,
		disableAnimations: appConfig.disableAnimations
	};

	const formSchema = z.object({
		appName: z.string().min(2).max(30),
		sessionDuration: z.number().min(1).max(43200),
		emailsVerified: z.boolean(),
		allowOwnAccountEdit: z.boolean(),
		disableAnimations: z.boolean()
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
			<CheckboxWithLabel
				id="self-account-editing"
				label={m.enable_self_account_editing()}
				description={m.whether_the_users_should_be_able_to_edit_their_own_account_details()}
				bind:checked={$inputs.allowOwnAccountEdit.value}
			/>
			<CheckboxWithLabel
				id="emails-verified"
				label={m.emails_verified()}
				description={m.whether_the_users_email_should_be_marked_as_verified_for_the_oidc_clients()}
				bind:checked={$inputs.emailsVerified.value}
			/>
			<CheckboxWithLabel
				id="disable-animations"
				label={m.disable_animations()}
				description={m.turn_off_all_animations_throughout_the_admin_ui()}
				bind:checked={$inputs.disableAnimations.value}
			/>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
