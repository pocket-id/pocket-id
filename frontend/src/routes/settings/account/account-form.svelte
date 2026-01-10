<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import ProfilePictureSettings from '$lib/components/form/profile-picture-settings.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field/index.js';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AccountUpdate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { emptyToUndefined, usernameSchema } from '$lib/utils/zod-util';
	import { toast } from 'svelte-sonner';
	import { get } from 'svelte/store';
	import { z } from 'zod/v4';

	let {
		callback,
		account,
		userId,
		isLdapUser = false,
		userInfoInputDisabled = false
	}: {
		account: AccountUpdate;
		userId: string;
		callback: (user: AccountUpdate) => Promise<boolean>;
		isLdapUser?: boolean;
		userInfoInputDisabled?: boolean;
	} = $props();

	let isLoading = $state(false);
	let hasManualDisplayNameEdit = $state(!!account.displayName);

	const userService = new UserService();

	const formSchema = z.object({
		firstName: z.string().min(1).max(50),
		lastName: emptyToUndefined(z.string().max(50).optional()),
		displayName: z.string().min(1).max(100),
		username: usernameSchema,
		email: get(appConfigStore).requireUserEmail ? z.email() : emptyToUndefined(z.email().optional())
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, {
		...account,
		email: account.email || ''
	});

	function onNameInput() {
		if (!hasManualDisplayNameEdit) {
			$inputs.displayName.value = `${$inputs.firstName.value}${
				$inputs.lastName?.value ? ' ' + $inputs.lastName.value : ''
			}`;
		}
	}

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;
		await callback(data);
		isLoading = false;
	}

	async function updateProfilePicture(image: File) {
		await userService
			.updateCurrentUsersProfilePicture(image)
			.then(() => toast.success(m.profile_picture_updated_successfully()))
			.catch(axiosErrorToast);
	}

	async function resetProfilePicture() {
		await userService
			.resetCurrentUserProfilePicture()
			.then(() => toast.success(m.profile_picture_has_been_reset()))
			.catch(axiosErrorToast);
	}
</script>

<form onsubmit={preventDefault(onSubmit)} class="space-y-6">
	<ProfilePictureSettings
		{userId}
		{isLdapUser}
		updateCallback={updateProfilePicture}
		resetCallback={resetProfilePicture}
	/>

	<Field.Separator class="m-2" />

	<fieldset disabled={userInfoInputDisabled}>
		<Field.Group class="grid grid-cols-1 gap-4 sm:grid-cols-2">
			<FormInput label={m.first_name()} bind:input={$inputs.firstName} onInput={onNameInput} />
			<FormInput label={m.last_name()} bind:input={$inputs.lastName} onInput={onNameInput} />
			<FormInput
				label={m.display_name()}
				bind:input={$inputs.displayName}
				onInput={() => (hasManualDisplayNameEdit = true)}
			/>
			<FormInput label={m.username()} bind:input={$inputs.username} />
			<FormInput label={m.email()} type="email" bind:input={$inputs.email} />
		</Field.Group>

		<div class="flex justify-end pt-4">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
