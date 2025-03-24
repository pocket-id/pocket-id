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

	let {
		callback,
		account,
		userId,
		updateProfilePicture,
		resetProfilePicture,
		isLdapUser = false
	}: {
		account: UserCreate;
		userId: string;
		callback: (user: UserCreate) => Promise<boolean>;
		updateProfilePicture: (image: File) => Promise<void>;
		resetProfilePicture: () => Promise<void>;
		isLdapUser?: boolean;
	} = $props();

	let isLoading = $state(false);
	let isImageLoading = $state(false);
	let imageDataURL = $state(`/api/users/${userId}/profile-picture.png`);

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

	async function onImageChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || null;
		if (!file) return;

		isImageLoading = true;

		const reader = new FileReader();
		reader.onload = (event) => {
			imageDataURL = event.target?.result as string;
		};
		reader.readAsDataURL(file);

		await updateProfilePicture(file).catch(() => {
			imageDataURL = `/api/users/${userId}/profile-picture.png`;
		});
		isImageLoading = false;
	}

	function onReset() {
		openConfirmDialog({
			title: m.reset_profile_picture_question(),
			message: m.this_will_remove_the_uploaded_image_and_reset_the_profile_picture_to_default(),
			confirm: {
				label: m.reset(),
				action: async () => {
					isImageLoading = true;
					await resetProfilePicture().catch();
					isImageLoading = false;
				}
			}
		});
	}
</script>

<form onsubmit={onSubmit} class="space-y-6">
	<!-- Profile Picture Row -->
	<div class="flex flex-col items-center gap-6 sm:flex-row">
		<div class="shrink-0">
			{#if isLdapUser}
				<Avatar.Root class="h-24 w-24">
					<Avatar.Image class="object-cover" src={imageDataURL} />
				</Avatar.Root>
			{:else}
				<FileInput
					id="profile-picture-input"
					variant="secondary"
					accept="image/png, image/jpeg"
					onchange={onImageChange}
				>
					<div class="group relative h-24 w-24 rounded-full">
						<Avatar.Root class="h-full w-full transition-opacity duration-200">
							<Avatar.Image
								class="object-cover group-hover:opacity-30 {isImageLoading ? 'opacity-30' : ''}"
								src={imageDataURL}
							/>
						</Avatar.Root>
						<div class="absolute inset-0 flex items-center justify-center">
							{#if isImageLoading}
								<LucideLoader class="h-5 w-5 animate-spin" />
							{:else}
								<LucideUpload
									class="h-5 w-5 opacity-0 transition-opacity group-hover:opacity-100"
								/>
							{/if}
						</div>
					</div>
				</FileInput>
			{/if}
		</div>

		<div class="grow">
			<h3 class="font-medium">{m.profile_picture()}</h3>
			{#if isLdapUser}
				<p class="text-muted-foreground text-sm">
					{m.profile_picture_is_managed_by_ldap_server()}
				</p>
			{:else}
				<p class="text-muted-foreground text-sm">
					{m.click_profile_picture_to_upload_custom()}
				</p>
				<p class="text-muted-foreground mb-2 text-sm">{m.image_should_be_in_format()}</p>
				<Button
					variant="outline"
					size="sm"
					on:click={onReset}
					disabled={isImageLoading || isLdapUser}
				>
					<LucideRefreshCw class="mr-2 h-4 w-4" />
					{m.reset_to_default()}
				</Button>
			{/if}
		</div>
	</div>

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
