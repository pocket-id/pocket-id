<script lang="ts">
	import Logo from '#lib/components/logo.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import {
		cachedApplicationLogo,
		cachedBackgroundImage,
		cachedDefaultProfilePicture,
		cachedEmailLogo
	} from '#lib/utils/cached-image-util.ts';
	import { trackUnsavedSection } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import ApplicationImage from '../application-image.svelte';

	const appConfigService = new AppConfigService();

	let logoLight = $state<File | null>();
	let logoDark = $state<File | null>();
	let logoEmail = $state<File | null>();
	let favicon = $state<File | null>();
	let defaultProfilePicture = $state<File | null>();
	let backgroundImage = $state<File | null>();

	// Bumped on discard to remount the image fields, resetting their previews
	let resetKey = $state(0);

	function clearPendingImages() {
		logoLight =
			logoDark =
			logoEmail =
			favicon =
			defaultProfilePicture =
			backgroundImage =
				undefined;
	}

	async function saveImages() {
		await Promise.all([
			favicon && appConfigService.updateFavicon(favicon),
			logoEmail && appConfigService.updateEmailLogo(logoEmail),
			logoLight === null
				? appConfigService.deleteLogo(true)
				: logoLight && appConfigService.updateLogo(logoLight, true),
			logoDark === null
				? appConfigService.deleteLogo(false)
				: logoDark && appConfigService.updateLogo(logoDark, false),
			defaultProfilePicture === null
				? appConfigService.deleteDefaultProfilePicture()
				: defaultProfilePicture &&
					appConfigService.updateDefaultProfilePicture(defaultProfilePicture),
			backgroundImage === null
				? appConfigService.deleteBackgroundImage()
				: backgroundImage && appConfigService.updateBackgroundImage(backgroundImage)
		]);
		// The previews already show the saved images, so only the pending files are cleared
		clearPendingImages();
	}

	trackUnsavedSection(
		() =>
			[logoLight, logoDark, logoEmail, favicon, defaultProfilePicture, backgroundImage].filter(
				(image) => image !== undefined
			).length,
		saveImages,
		() => {
			clearPendingImages();
			resetKey++;
		}
	);
</script>

{#snippet lightLogoFallback()}
	<Logo defaultOnly colorScheme="light" class="size-full" />
{/snippet}

{#snippet darkLogoFallback()}
	<Logo defaultOnly colorScheme="dark" class="size-full" />
{/snippet}

{#snippet spec(text: string)}
	<p class="text-caption text-fg-muted pt-2 pb-3">{text}</p>
{/snippet}

{#key resetKey}
	<fieldset class="flex min-w-0 flex-col gap-10" disabled={$appConfigStore.uiConfigDisabled}>
		<SettingsSection title={m.logo()} description={m.config_logo_description()}>
			{@render spec(m.config_logo_spec())}
			<div class="grid items-start gap-x-2 gap-y-4 sm:grid-cols-2">
				<ApplicationImage
					name={m.config_image_light_logo()}
					label={m.light()}
					bind:image={logoLight}
					imageURL={cachedApplicationLogo.getUrl(true)}
					accept="image/svg+xml, image/png, image/webp"
					fallback={lightLogoFallback}
					canReset
					minWidth={256}
				/>
				<ApplicationImage
					name={m.config_image_dark_logo()}
					label={m.dark()}
					bind:image={logoDark}
					imageURL={cachedApplicationLogo.getUrl(false)}
					accept="image/svg+xml, image/png, image/webp"
					well="dark"
					fallback={darkLogoFallback}
					canReset
					minWidth={256}
				/>
			</div>
		</SettingsSection>

		<SettingsSection
			title={m.config_signin_background()}
			description={m.config_signin_background_description()}
		>
			{@render spec(m.config_signin_background_spec())}
			<ApplicationImage
				name={m.config_image_background()}
				bind:image={backgroundImage}
				imageURL={cachedBackgroundImage.getUrl()}
				accept="image/jpeg, image/png, image/webp"
				variant="background"
				well="surface"
				canRemove
				minWidth={1920}
			/>
		</SettingsSection>

		<SettingsSection
			title={m.email_logo()}
			description={m.config_email_logo_description({ appName: $appConfigStore.appName })}
		>
			{@render spec(m.config_email_logo_spec())}
			<ApplicationImage
				name={m.config_image_email_logo()}
				bind:image={logoEmail}
				imageURL={cachedEmailLogo.getUrl()}
				accept="image/png, image/jpeg"
			/>
		</SettingsSection>

		<SettingsSection title={m.favicon()}>
			{@render spec(m.config_favicon_spec())}
			<ApplicationImage
				name={m.config_image_favicon()}
				bind:image={favicon}
				imageURL="/api/application-images/favicon"
				accept="image/svg+xml, image/png, image/x-icon"
			/>
		</SettingsSection>

		<SettingsSection
			title={m.default_profile_picture()}
			description={m.config_default_profile_picture_description()}
		>
			{@render spec(m.config_default_profile_picture_spec())}
			<ApplicationImage
				name={m.config_image_default_profile_picture()}
				bind:image={defaultProfilePicture}
				imageURL={cachedDefaultProfilePicture.getUrl()}
				accept="image/png, image/jpeg, image/webp"
				variant="avatar"
				canRemove
			/>
		</SettingsSection>
	</fieldset>
{/key}
