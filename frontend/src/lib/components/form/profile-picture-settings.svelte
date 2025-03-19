<script lang="ts">
	import FileInput from '$lib/components/form/file-input.svelte';
	import * as Avatar from '$lib/components/ui/avatar';
	import Button from '$lib/components/ui/button/button.svelte';
	import { LucideLoader, LucideRefreshCw, LucideUpload } from 'lucide-svelte';
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
			imageDataURL = `/api/users/${userId}/profile-picture.png}`;
		});
		isLoading = false;
	}

	function onReset() {
		openConfirmDialog({
			title: 'Reset profile picture?',
			message:
				'This will remove the uploaded image, and reset the profile picture to default. Do you want to continue?',
			confirm: {
				label: 'Reset',
				action: async () => {
					isLoading = true;
					await resetCallback().catch();
					isLoading = false;
				}
			}
		});
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
			<Button
				variant="outline"
				size="sm"
				class="mt-5"
				on:click={onReset}
				disabled={isLoading || isLdapUser}
			>
				<LucideRefreshCw class="mr-2 h-4 w-4" />
				Reset to default
			</Button>
		</div>
		{#if isLdapUser}
			<Avatar.Root class="h-24 w-24">
				<Avatar.Image class="object-cover" src={imageDataURL} />
			</Avatar.Root>
		{:else}
			<div class="flex flex-col items-center gap-2">
				<FileInput
					id="profile-picture-input"
					variant="secondary"
					accept="image/png, image/jpeg"
					onchange={onImageChange}
				>
					<div class="group relative h-28 w-28 rounded-full">
						<Avatar.Root class="h-full w-full transition-opacity duration-200">
							<Avatar.Image
								class="object-cover group-hover:opacity-10 {isLoading ? 'opacity-10' : ''}"
								src={imageDataURL}
							/>
						</Avatar.Root>
						<div class="absolute inset-0 flex items-center justify-center">
							{#if isLoading}
								<LucideLoader class="h-5 w-5 animate-spin" />
							{:else}
								<LucideUpload
									class="h-5 w-5 opacity-0 transition-opacity group-hover:opacity-100"
								/>
							{/if}
						</div>
					</div>
				</FileInput>
			</div>
		{/if}
	</div>
</div>
