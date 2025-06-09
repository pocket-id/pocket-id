<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideImage, Mail, SlidersHorizontal, UserSearch } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import AppConfigEmailForm from './forms/app-config-email-form.svelte';
	import AppConfigGeneralForm from './forms/app-config-general-form.svelte';
	import AppConfigLdapForm from './forms/app-config-ldap-form.svelte';
	import UpdateApplicationImages from './update-application-images.svelte';
	import AccentPicker from './accent-picker.svelte';

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
		logoLight: File | null,
		logoDark: File | null,
		backgroundImage: File | null,
		favicon: File | null
	) {
		const faviconPromise = favicon ? appConfigService.updateFavicon(favicon) : Promise.resolve();
		const lightLogoPromise = logoLight
			? appConfigService.updateLogo(logoLight, true)
			: Promise.resolve();
		const darkLogoPromise = logoDark
			? appConfigService.updateLogo(logoDark, false)
			: Promise.resolve();
		const backgroundImagePromise = backgroundImage
			? appConfigService.updateBackgroundImage(backgroundImage)
			: Promise.resolve();

		await Promise.all([lightLogoPromise, darkLogoPromise, backgroundImagePromise, faviconPromise])
			.then(() => toast.success(m.images_updated_successfully()))
			.catch(axiosErrorToast);
	}
</script>

<svelte:head>
	<title>{m.application_configuration()}</title>
</svelte:head>

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
	<CollapsibleCard id="application-configuration-images" icon={LucideImage} title={m.images()}>
		<UpdateApplicationImages callback={updateImages} />
	</CollapsibleCard>
</div>
