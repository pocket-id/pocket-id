<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import * as Avatar from '$lib/components/ui/avatar';
	import Button from '$lib/components/ui/button/button.svelte';
	import { LucideLoader, LucideRefreshCw, LucideUpload } from 'lucide-svelte';
	import { openConfirmDialog } from '../confirm-dialog';
	import { m } from '$lib/paraglide/messages';
	import type UserService from '$lib/services/user-service';

	let {
		userId,
		isLdapUser = false,
		resetCallback,
		updateCallback
	}: {
		userId: string;
		isLdapUser?: boolean;
		resetCallback: () => Promise<void>;
		updateCallback: (image: File) => Promise<void>;
	} = $props();

	let isLoading = $state(false);
	let imageDataURL = $state(`/api/users/${userId}/profile-picture.png`);

	async function onImageChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || null;
		if (!file) return;

		isLoading = true;

		const reader = new FileReader();
		reader.onload = (event) => {
			imageDataURL = event.target?.result as string;
		};
		reader.readAsDataURL(file);

		await updateCallback(file).catch(() => {
			imageDataURL = `/api/users/${userId}/profile-picture.png`;
		});
		isLoading = false;
	}

	function onReset() {
		openConfirmDialog({
			title: m.reset_profile_picture_question(),
			message: m.this_will_remove_the_uploaded_image_and_reset_the_profile_picture_to_default(),
			confirm: {
				label: m.reset(),
				action: async () => {
					isLoading = true;
					await resetCallback().catch();
					isLoading = false;
				}
			}
		});
	}
</script>

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
							class="object-cover group-hover:opacity-30 {isLoading ? 'opacity-30' : ''}"
							src={imageDataURL}
						/>
					</Avatar.Root>
					<div class="absolute inset-0 flex items-center justify-center">
						{#if isLoading}
							<LucideLoader class="h-5 w-5 animate-spin" />
						{:else}
							<LucideUpload class="h-5 w-5 opacity-0 transition-opacity group-hover:opacity-100" />
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
			<Button variant="outline" size="sm" on:click={onReset} disabled={isLoading || isLdapUser}>
				<LucideRefreshCw class="mr-2 h-4 w-4" />
				{m.reset_to_default()}
			</Button>
		{/if}
	</div>
</div>
