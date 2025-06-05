<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserGroupCreate } from '$lib/types/user-group.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod';

	let {
		callback,
		existingUserGroup
	}: {
		existingUserGroup?: UserGroupCreate;
		callback: (userGroup: UserGroupCreate) => Promise<boolean>;
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

	const { inputs, ...form } = createForm<FormSchema>(formSchema, userGroup);

	function onFriendlyNameInput(e: any) {
		if (!hasManualNameEdit) {
			$inputs.name.value = e.target!.value.toLowerCase().replace(/[^a-z0-9_]/g, '_');
		}
	}

	function onNameInput(_: Event) {
		hasManualNameEdit = true;
	}

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;
		const success = await callback(data);
		// Reset form if user group was successfully created
		if (success && !existingUserGroup) {
			form.reset();
			hasManualNameEdit = false;
		}
		isLoading = false;
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
		<div class="mt-5 flex justify-end">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
