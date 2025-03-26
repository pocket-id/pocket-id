<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import type { UserCreate } from '$lib/types/user.type';
	import { createForm } from '$lib/utils/form-util';
	import { z } from 'zod';
	import * as Avatar from '$lib/components/ui/avatar';
	import FileInput from '$lib/components/form/file-input.svelte';
	import { LucideLoader, LucideRefreshCw, LucideUpload, BookUser } from 'lucide-svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import ProfilePictureSettings from '$lib/components/form/profile-picture-settings.svelte';
	import UserService from '$lib/services/user-service';
	import { toast } from 'svelte-sonner';
	import { axiosErrorToast } from '$lib/utils/error-util';

	let {
		callback,
		account,
		userId,
		isLdapUser = false
	}: {
		account: UserCreate;
		userId: string;
		callback: (user: UserCreate) => Promise<boolean>;
		isLdapUser?: boolean;
	} = $props();

	let isLoading = $state(false);

	const userService = new UserService();

	const formSchema = z.object({
		firstName: z.string().min(1).max(50),
		lastName: z.string().min(1).max(50),
		username: z
			.string()
			.min(2)
			.max(30)
			.regex(/^[a-z0-9_@.-]+$/, m.username_can_only_contain()),
		email: z.string().email(),
		isAdmin: z.boolean()
	});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, account);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return;
		isLoading = true;
		await callback(data);
		isLoading = false;
	}

	async function updateProfilePicture(image: File) {
		await userService
			.updateProfilePicture(userId, image)
			.then(() => toast.success(m.profile_picture_updated_successfully()))
			.catch(axiosErrorToast);
	}

	async function resetProfilePicture() {
		openConfirmDialog({
			title: m.reset_profile_picture_question(),
			message: m.this_will_remove_the_uploaded_image_and_reset_the_profile_picture_to_default(),
			confirm: {
				label: m.reset(),
				action: async () => {
					await userService
						.resetProfilePicture(userId)
						.then(() => toast.success(m.profile_picture_has_been_reset()))
						.catch(axiosErrorToast);
				}
			}
		});
	}
</script>

<form onsubmit={onSubmit} class="space-y-6">
	<!-- Profile Picture Section -->
	<ProfilePictureSettings
		{userId}
		{isLdapUser}
		updateCallback={updateProfilePicture}
		resetCallback={resetProfilePicture}
	/>

	<!-- Divider -->
	<hr class="border-border" />

	<!-- User Information -->
	<div>
		<h3 class="mb-5 flex items-center gap-2 text-xl font-semibold">
			<BookUser class="text-primary/80 h-5 w-5" />
			{m.personal_information()}
		</h3>

		<div class="flex flex-col gap-3 sm:flex-row">
			<div class="w-full">
				<FormInput label={m.first_name()} bind:input={$inputs.firstName} />
			</div>
			<div class="w-full">
				<FormInput label={m.last_name()} bind:input={$inputs.lastName} />
			</div>
		</div>
		<div class="mt-3 flex flex-col gap-3 sm:flex-row">
			<div class="w-full">
				<FormInput label={m.email()} bind:input={$inputs.email} />
			</div>
			<div class="w-full">
				<FormInput label={m.username()} bind:input={$inputs.username} />
			</div>
		</div>
	</div>

	<div class="flex justify-end pt-2">
		<Button {isLoading} type="submit">{m.save()}</Button>
	</div>
</form>
