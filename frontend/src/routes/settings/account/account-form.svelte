<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import ProfilePictureSettings from '$lib/components/form/profile-picture-settings.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { emptyToUndefined, usernameSchema } from '$lib/utils/zod-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		callback,
		account,
		userId,
		isLdapUser = false,
		userInfoInputDisabled = false
	}: {
		account: UserCreate;
		userId: string;
		callback: (user: UserCreate) => Promise<boolean>;
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
		email: z.email(),
		isAdmin: z.boolean()
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, account);

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

	<hr class="border-border" />

	<fieldset disabled={userInfoInputDisabled}>
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
			<div>
				<FormInput label={m.first_name()} bind:input={$inputs.firstName} onInput={onNameInput} />
			</div>
			<div>
				<FormInput label={m.last_name()} bind:input={$inputs.lastName} onInput={onNameInput} />
			</div>
			<div>
				<FormInput
					label={m.display_name()}
					bind:input={$inputs.displayName}
					onInput={() => (hasManualDisplayNameEdit = true)}
				/>
			</div>
			<div>
				<FormInput label={m.username()} bind:input={$inputs.username} />
			</div>
			<div>
				<FormInput label={m.email()} bind:input={$inputs.email} />
			</div>
		</div>

		<div class="flex justify-end pt-4">
			<Button {isLoading} type="submit">{m.save()}</Button>
		</div>
	</fieldset>
</form>
