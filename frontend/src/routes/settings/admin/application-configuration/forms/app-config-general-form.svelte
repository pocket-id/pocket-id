<script lang="ts">
	import { env } from '$env/dynamic/public';
	import CheckboxWithLabel from '$lib/components/form/checkbox-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import type { AllAppConfig } from '$lib/types/application-configuration';
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

	const uiConfigDisabled = env.PUBLIC_UI_CONFIG_DISABLED === 'true';
	let isLoading = $state(false);

	const updatedAppConfig = {
		appName: appConfig.appName,
		sessionDuration: appConfig.sessionDuration,
		emailsVerified: appConfig.emailsVerified,
		allowOwnAccountEdit: appConfig.allowOwnAccountEdit
	};

	const formSchema = z.object({
		appName: z.string().min(2).max(30),
		sessionDuration: z.number().min(1).max(43200),
		emailsVerified: z.boolean(),
		allowOwnAccountEdit: z.boolean()
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, updatedAppConfig);
	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;
		await callback(data).finally(() => (isLoading = false));
		toast.success('Application configuration updated successfully');
	}
</script>

<form onsubmit={onSubmit}>
	<fieldset class="flex flex-col gap-5" disabled={uiConfigDisabled}>
		<div class="flex flex-col gap-5">
			<FormInput label="Application Name" bind:input={$inputs.appName} />
			<FormInput
				label="Session Duration"
				type="number"
				description="The duration of a session in minutes before the user has to sign in again."
				bind:input={$inputs.sessionDuration}
			/>
			<CheckboxWithLabel
				id="self-account-editing"
				label="Enable Self-Account Editing"
				description="Whether the users should be able to edit their own account details."
				bind:checked={$inputs.allowOwnAccountEdit.value}
			/>
			<CheckboxWithLabel
				id="emails-verified"
				label="Emails Verified"
				description="Whether the user's email should be marked as verified for the OIDC clients."
				bind:checked={$inputs.emailsVerified.value}
			/>
		</div>
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">Save</Button>
		</div>
	</fieldset>
</form>
