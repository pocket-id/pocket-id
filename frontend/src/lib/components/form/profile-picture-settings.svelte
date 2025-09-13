<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import * as Avatar from '$lib/components/ui/avatar';
	import Button from '$lib/components/ui/button/button.svelte';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { cachedProfilePicture } from '$lib/utils/cached-image-util';
	import { LucideLoader, LucideRefreshCw, LucideUpload } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { openConfirmDialog } from '../confirm-dialog';

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
	let imageDataURL = $state('');
	onMount(() => {
		// The "skipCache" query will only be added to the profile picture url on client-side
		// because of that we need to set the imageDataURL after the component is mounted
		imageDataURL = cachedProfilePicture.getUrl(userId);
	});

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
			imageDataURL = cachedProfilePicture.getUrl(userId);
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
					try {
						await resetCallback();
						await fetch(cachedProfilePicture.getUrl(userId, { skipCache: true }))
							.then((response) => response.blob())
							.then((blob) => {
								imageDataURL = URL.createObjectURL(blob);
							});
					} finally {
						isLoading = false;
					}
				}
			}
		});
	}
</script>

<div class="flex flex-col items-center gap-6 sm:flex-row">
	<div class="shrink-0">
		{#if isLdapUser && $appConfigStore.ldapEnabled}
			<Avatar.Root class="size-24">
				<Avatar.Image class="object-cover" src={imageDataURL} />
			</Avatar.Root>
		{:else}
			<FileInput
				id="profile-picture-input"
				variant="secondary"
				accept="image/png, image/jpeg, image/webp, image/avif, image/heic"
				onchange={onImageChange}
			>
				<div class="group relative size-24 rounded-full">
					<Avatar.Root class="h-full w-full transition-opacity duration-200">
						<Avatar.Image
							class="object-cover group-hover:opacity-30 {isLoading ? 'opacity-30' : ''}"
							src={imageDataURL}
						/>
					</Avatar.Root>
					<div class="absolute inset-0 flex items-center justify-center">
						{#if isLoading}
							<LucideLoader class="size-5 animate-spin" />
						{:else}
							<LucideUpload class="size-5 opacity-0 transition-opacity group-hover:opacity-100" />
						{/if}
					</div>
				</div>
			</FileInput>
		{/if}
	</div>

	<div class="grow">
		<h3 class="font-medium">{m.profile_picture()}</h3>
		{#if isLdapUser && $appConfigStore.ldapEnabled}
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
				onclick={onReset}
				disabled={isLoading || (isLdapUser && $appConfigStore.ldapEnabled)}
			>
				<LucideRefreshCw class="mr-2 size-4" />
				{m.reset_to_default()}
			</Button>
		{/if}
	</div>
</div>
