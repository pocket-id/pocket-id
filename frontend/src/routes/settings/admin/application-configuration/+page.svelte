<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import {
		LucideImage,
		LucideInfo,
		Mail,
		SlidersHorizontal,
		UserSearch,
		Users
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import AppConfigEmailForm from './forms/app-config-email-form.svelte';
	import AppConfigGeneralForm from './forms/app-config-general-form.svelte';
	import AppConfigLdapForm from './forms/app-config-ldap-form.svelte';
	import AppConfigSignupDefaultsForm from './forms/app-config-signup-defaults-form.svelte';
	import UpdateApplicationImages from './update-application-images.svelte';

	let { data } = $props();
	let appConfig = $state(data.appConfig);

	const appConfigService = new AppConfigService();

	async function updateAppConfig(updatedAppConfig: Partial<AllAppConfig>) {
		appConfig = await appConfigService
			.update({
				...appConfig,
				...updatedAppConfig
			})
			.catch((e) => {
				axiosErrorToast(e);
				throw e;
			});
		await appConfigStore.reload();
	}

	async function updateImages(
		logoLight: File | undefined,
		logoDark: File | undefined,
		logoEmail: File | undefined,
		defaultProfilePicture: File | null | undefined,
		backgroundImage: File | undefined,
		favicon: File | undefined
	) {
		const faviconPromise = favicon ? appConfigService.updateFavicon(favicon) : Promise.resolve();

		const lightLogoPromise = logoLight
			? appConfigService.updateLogo(logoLight, true)
			: Promise.resolve();

		const darkLogoPromise = logoDark
			? appConfigService.updateLogo(logoDark, false)
			: Promise.resolve();

		const emailLogoPromise = logoEmail
			? appConfigService.updateEmailLogo(logoEmail)
			: Promise.resolve();

		const defaultProfilePicturePromise =
			defaultProfilePicture === null
				? appConfigService.deleteDefaultProfilePicture()
				: defaultProfilePicture
					? appConfigService.updateDefaultProfilePicture(defaultProfilePicture)
					: Promise.resolve();

		const backgroundImagePromise = backgroundImage
			? appConfigService.updateBackgroundImage(backgroundImage)
			: Promise.resolve();

		await Promise.all([
			lightLogoPromise,
			darkLogoPromise,
			emailLogoPromise,
			defaultProfilePicturePromise,
			backgroundImagePromise,
			faviconPromise
		])
			.then(() => toast.success(m.images_updated_successfully()))
			.catch(axiosErrorToast);
	}
</script>

<svelte:head>
	<title>{m.application_configuration()}</title>
</svelte:head>

{#if $appConfigStore.uiConfigDisabled}
	<Alert.Root variant="info">
		<LucideInfo class="size-4" />
		<Alert.Title>{m.ui_config_disabled_info_title()}</Alert.Title>
		<Alert.Description>
			{m.ui_config_disabled_info_description()}
		</Alert.Description>
	</Alert.Root>
{/if}
<div>
	<CollapsibleCard
		id="application-configuration-general"
		icon={SlidersHorizontal}
		title={m.general()}
		defaultExpanded
	>
		<AppConfigGeneralForm {appConfig} callback={updateAppConfig} />
	</CollapsibleCard>
</div>

<div>
	<CollapsibleCard
		id="application-configuration-signup-defaults"
		icon={Users}
		title={m.user_creation()}
		description={m.configure_user_creation()}
	>
		<AppConfigSignupDefaultsForm {appConfig} callback={updateAppConfig} />
	</CollapsibleCard>
</div>

<div>
	<CollapsibleCard
		id="application-configuration-email"
		icon={Mail}
		title={m.email()}
		description={m.configure_smtp_to_send_emails()}
	>
		<AppConfigEmailForm {appConfig} callback={updateAppConfig} />
	</CollapsibleCard>
</div>

<div>
	<CollapsibleCard
		id="application-configuration-ldap"
		icon={UserSearch}
		title={m.ldap()}
		description={m.configure_ldap_settings_to_sync_users_and_groups_from_an_ldap_server()}
	>
		<AppConfigLdapForm {appConfig} callback={updateAppConfig} />
	</CollapsibleCard>
</div>

<div>
	<CollapsibleCard
		id="application-configuration-images"
		icon={LucideImage}
		title={m.images()}
		description={m.configure_application_images()}
	>
		<UpdateApplicationImages callback={updateImages} />
	</CollapsibleCard>
</div>
