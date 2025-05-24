<script lang="ts">
	import Button from '$lib/components/ui/button/button.svelte';
	import { m } from '$lib/paraglide/messages';
	import ApplicationImage from './application-image.svelte';

	let {
		callback
	}: {
		callback: (
			logoLight: File | null,
			logoDark: File | null,
			backgroundImage: File | null,
			favicon: File | null
		) => void;
	} = $props();

	let logoLight = $state<File | null>(null);
	let logoDark = $state<File | null>(null);
	let backgroundImage = $state<File | null>(null);
	let favicon = $state<File | null>(null);
</script>

<div class="flex flex-col gap-8">
	<ApplicationImage
		id="favicon"
		imageClass="size-14 p-2"
		label={m.favicon()}
		bind:image={favicon}
		imageURL="/api/application-configuration/favicon"
		accept="image/x-icon"
	/>
	<ApplicationImage
		id="logo-light"
		imageClass="size-32"
		label={m.light_mode_logo()}
		bind:image={logoLight}
		imageURL="/api/application-configuration/logo?light=true"
		forceColorScheme="light"
	/>
	<ApplicationImage
		id="logo-dark"
		imageClass="size-32"
		label={m.dark_mode_logo()}
		bind:image={logoDark}
		imageURL="/api/application-configuration/logo?light=false"
		forceColorScheme="dark"
	/>
	<ApplicationImage
		id="background-image"
		imageClass="h-[350px] max-w-[500px]"
		label={m.background_image()}
		bind:image={backgroundImage}
		imageURL="/api/application-configuration/background-image"
	/>
</div>
<div class="flex justify-end">
	<Button class="mt-5" onclick={() => callback(logoLight, logoDark, backgroundImage, favicon)}
		>{m.save()}</Button
	>
</div>
