<script lang="ts">
	import Button from '$lib/components/ui/button/button.svelte';
	import { m } from '$lib/paraglide/messages';
	import {
		cachedApplicationLogo,
		cachedBackgroundImage,
		cachedDefaultProfilePicture,
		cachedEmailLogo
	} from '$lib/utils/cached-image-util';
	import ApplicationImage from './application-image.svelte';

	let {
		callback
	}: {
		callback: (
			logoLight: File | undefined,
			logoDark: File | undefined,
			logoEmail: File | undefined,
			defaultProfilePicture: File | null | undefined,
			backgroundImage: File | null | undefined,
			favicon: File | undefined
		) => void;
	} = $props();

	let logoLight = $state<File | undefined>();
	let logoDark = $state<File | undefined>();
	let logoEmail = $state<File | undefined>();
	let defaultProfilePicture = $state<File | null | undefined>();
	let backgroundImage = $state<File | null| undefined>();
	let favicon = $state<File | undefined>();

	let defaultProfilePictureSet = $state(true);
	let backgroundImageSet = $state(true);
</script>

<div class="flex flex-col gap-8">
	<ApplicationImage
		id="favicon"
		imageClass="size-14 p-2"
		label={m.favicon()}
		bind:image={favicon}
		imageURL="/api/application-images/favicon"
		accept="image/x-icon"
	/>
	<ApplicationImage
		id="logo-light"
		imageClass="size-24"
		label={m.light_mode_logo()}
		bind:image={logoLight}
		imageURL={cachedApplicationLogo.getUrl(true)}
		forceColorScheme="light"
	/>
	<ApplicationImage
		id="logo-dark"
		imageClass="size-24"
		label={m.dark_mode_logo()}
		bind:image={logoDark}
		imageURL={cachedApplicationLogo.getUrl(false)}
		forceColorScheme="dark"
	/>
	<ApplicationImage
		id="logo-email"
		imageClass="size-24"
		label={m.email_logo()}
		bind:image={logoEmail}
		imageURL={cachedEmailLogo.getUrl()}
		accept="image/png, image/jpeg"
		forceColorScheme="light"
	/>
	<ApplicationImage
		id="default-profile-picture"
		imageClass="size-24"
		label={m.default_profile_picture()}
		isResetable
		bind:image={defaultProfilePicture}
		imageURL={cachedDefaultProfilePicture.getUrl()}
		isImageSet={defaultProfilePictureSet}
	/>
	<ApplicationImage
		id="background-image"
		imageClass="max-h-[350px] max-w-[500px]"
		label={m.background_image()}
		isResetable
		bind:image={backgroundImage}
		imageURL={cachedBackgroundImage.getUrl()}
		isImageSet={backgroundImageSet}
	/>
</div>
<div class="flex justify-end">
	<Button
		class="mt-5"
		usePromiseLoading
		onclick={() =>
			callback(logoLight, logoDark, logoEmail, defaultProfilePicture, backgroundImage, favicon)}
		>{m.save()}</Button
	>
</div>
