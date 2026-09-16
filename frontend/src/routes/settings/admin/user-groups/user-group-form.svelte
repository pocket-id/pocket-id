<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserGroupCreate } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { trackFormChanges } from '$lib/utils/unsaved-changes-util.svelte';
	import { z } from 'zod/v4';

	let {
		callback,
		existingUserGroup
	}: {
		existingUserGroup?: UserGroupCreate;
		callback: (userGroup: UserGroupCreate) => Promise<void>;
	} = $props();

	let isLoading = $state(false);
	let inputDisabled = $derived(!!existingUserGroup?.ldapId && $appConfigStore.ldapEnabled);
	let hasManualNameEdit = $state(!!existingUserGroup?.friendlyName);

	const userGroup = {
		name: existingUserGroup?.name || '',
		friendlyName: existingUserGroup?.friendlyName || ''
	};

	const formSchema = z.object({
		friendlyName: z.string().min(2).max(50),
		name: z.string().min(2).max(255)
	});
	type FormSchema = typeof formSchema;

	const formStore = createForm<FormSchema>(formSchema, userGroup);
	const { inputs } = formStore;

	function onFriendlyNameInput(e: any) {
		if (!hasManualNameEdit) {
			$inputs.name.value = e.target!.value.toLowerCase().replace(/[^a-z0-9_]/g, '_');
		}
	}

	function onNameInput() {
		hasManualNameEdit = true;
	}

	async function saveUserGroup(data: z.infer<FormSchema>) {
		await callback(data);
		// Reset form if user group was successfully created
		if (!existingUserGroup) {
			formStore.reset();
			hasManualNameEdit = false;
		}
	}

	// Create mode has its own Save button rather than going through the unsaved-changes bar.
	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			await saveUserGroup(data);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}

	if (existingUserGroup) {
		trackFormChanges(() => formStore, saveUserGroup);
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<fieldset disabled={inputDisabled}>
		<div class="flex flex-col gap-3 sm:flex-row">
			<div class="w-full">
				<FormInput
					label={m.friendly_name()}
					description={m.name_that_will_be_displayed_in_the_ui()}
					bind:input={$inputs.friendlyName}
					onInput={onFriendlyNameInput}
				/>
			</div>
			<div class="w-full">
				<FormInput
					label={m.name()}
					description={m.name_that_will_be_in_the_groups_claim()}
					bind:input={$inputs.name}
					onInput={onNameInput}
				/>
			</div>
		</div>
		{#if !existingUserGroup}
			<div class="mt-5 flex justify-end">
				<Button {isLoading} type="submit">{m.save()}</Button>
			</div>
		{/if}
	</fieldset>
</form>
