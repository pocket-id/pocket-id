<script lang="ts">
	import * as Avatar from '#lib/components/ui/avatar/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Spinner } from '#lib/components/ui/spinner/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { cachedProfilePicture } from '#lib/utils/cached-image-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideRotateCcw, LucideUpload } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { openConfirmDialog } from '../confirm-dialog';
	import ProfilePictureCropDialog from './profile-picture-crop-dialog.svelte';

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

	const managedByLdap = $derived(isLdapUser && $appConfigStore.ldapEnabled);
	const labelId = $props.id();

	let fileInput = $state<HTMLInputElement>();
	let pickedFile = $state<File | null>(null);
	let isLoading = $state(false);
	let imageDataURL = $state('');
	onMount(() => {
		// The "skipCache" query will only be added to the profile picture url on client-side
		// because of that we need to set the imageDataURL after the component is mounted
		imageDataURL = cachedProfilePicture.getUrl(userId);
	});

	async function onSave(image: File) {
		isLoading = true;
		try {
			await updateCallback(image);
			imageDataURL = URL.createObjectURL(image);
		} finally {
			isLoading = false;
		}
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

<div class="flex flex-wrap items-center gap-x-4 gap-y-3">
	<div class="relative size-18 shrink-0">
		<Avatar.Root class="size-full">
			<Avatar.Image
				class={cn(
					'object-cover transition-opacity duration-(--dur-base)',
					isLoading && 'opacity-50'
				)}
				src={imageDataURL}
				alt=""
			/>
		</Avatar.Root>
		{#if isLoading}
			<Spinner class="text-fg absolute inset-0 m-auto size-4" />
		{/if}
	</div>
	<div class="flex min-w-0 flex-1 basis-48 flex-col gap-0.5">
		<span id={labelId} class="text-heading text-fg">{m.profile_picture()}</span>
		<p class="text-caption text-fg-muted">
			{managedByLdap ? m.profile_picture_from_directory() : m.profile_picture_formats()}
		</p>
	</div>
	{#if !managedByLdap}
		<div class="flex items-center gap-1">
			<Button
				variant="secondary"
				size="sm"
				aria-label={m.upload_profile_picture()}
				disabled={isLoading}
				onclick={() => fileInput?.click()}
			>
				<LucideUpload />
				{m.upload()}
			</Button>
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<Button
							{...props}
							variant="ghost"
							size="icon-sm"
							aria-label={m.reset_to_default()}
							disabled={isLoading}
							onclick={onReset}
						>
							<LucideRotateCcw />
						</Button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>{m.reset_to_default()}</Tooltip.Content>
			</Tooltip.Root>
		</div>
		<input
			bind:this={fileInput}
			type="file"
			hidden
			aria-labelledby={labelId}
			accept="image/png, image/jpeg, image/webp, image/avif, image/heic"
			onchange={(e) => {
				pickedFile = e.currentTarget.files?.[0] ?? null;
				// Clearing the input lets the same file be picked again after cancelling the crop
				e.currentTarget.value = '';
			}}
		/>
		<ProfilePictureCropDialog bind:file={pickedFile} {onSave} />
	{/if}
</div>
