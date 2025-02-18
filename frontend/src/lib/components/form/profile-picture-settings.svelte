<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import * as Avatar from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';

	let {
		userId,
		isLdapUser = false,
		callback
	}: {
		userId: string;
		isLdapUser?: boolean;
		callback: (image: File) => Promise<void>;
	} = $props();

	let isLoading = $state(false);
	let image: File | null = $state(null);

	let imageDataURL = $state(`/api/users/${userId}/profile-picture.png`);

	function onImageChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0] || null;
		if (!file) return;

		image = file;

		const reader = new FileReader();
		reader.onload = (event) => {
			imageDataURL = event.target?.result as string;
		};
		reader.readAsDataURL(file);
	}

	async function onSave() {
		isLoading = true;
		await callback(image!).catch();
		image = null;
		isLoading = false;
	}
</script>

<div class="flex gap-5">
	<div class="flex w-full flex-col justify-between gap-5 sm:flex-row">
		<div>
			<h3 class="text-xl font-semibold">Profile Picture</h3>
			{#if isLdapUser}
				<p class="text-muted-foreground mt-1 text-sm">
					The profile picture is managed by the LDAP server and cannot be changed here.
				</p>
			{:else}
				<p class="text-muted-foreground mt-1 text-sm">
					Click on the profile picture to upload a custom one from your files.
				</p>
				<p class="text-muted-foreground mt-1 text-sm">The image should be in PNG or JPEG format.</p>
			{/if}
		</div>
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
						<Avatar.Image class="object-cover group-hover:opacity-10" src={imageDataURL} />
					</Avatar.Root>
					<span
						class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 transform font-medium opacity-0 transition-opacity group-hover:opacity-100"
					>
						Update
					</span>
				</div>
			</FileInput>
		{/if}
	</div>

	<Button class="ml-[5%] self-end" {isLoading} disabled={!image} onclick={onSave}>Save</Button>
</div>
