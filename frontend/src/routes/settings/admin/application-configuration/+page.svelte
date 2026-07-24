<script lang="ts">
	import * as Alert from '$lib/components/ui/alert';
	import * as Card from '$lib/components/ui/card';
	import * as Tabs from '$lib/components/ui/tabs';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideInfo } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import AppConfigDynamicClientsForm from './forms/app-config-dynamic-clients-form.svelte';
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
		backgroundImage: File | null | undefined,
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

		const backgroundImagePromise =
			backgroundImage === null
				? appConfigService.deleteBackgroundImage()
				: backgroundImage
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
<Tabs.Root value="general" useHash class="gap-4">
	<div class="overflow-x-auto pb-1">
		<Tabs.List variant="line" class="min-w-max">
			<Tabs.Trigger value="general">
				{m.general()}
			</Tabs.Trigger>
			<Tabs.Trigger value="user-creation">
				{m.user_creation()}
			</Tabs.Trigger>
			<Tabs.Trigger value="email">
				{m.email()}
			</Tabs.Trigger>
			<Tabs.Trigger value="ldap">
				{m.ldap()}
			</Tabs.Trigger>
			<Tabs.Trigger value="dynamic-clients">
				{m.dynamic_clients()}
			</Tabs.Trigger>
			<Tabs.Trigger value="images">
				{m.images()}
			</Tabs.Trigger>
		</Tabs.List>
	</div>

	<Tabs.Content value="general" id="application-configuration-general">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.general()}</Card.Title>
			</Card.Header>
			<Card.Content>
				<AppConfigGeneralForm {appConfig} callback={updateAppConfig} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="user-creation" id="application-configuration-signup-defaults">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.user_creation()}</Card.Title>
				<Card.Description>{m.configure_user_creation()}</Card.Description>
			</Card.Header>
			<Card.Content>
				<AppConfigSignupDefaultsForm {appConfig} callback={updateAppConfig} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="email" id="application-configuration-email">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.email()}</Card.Title>
				<Card.Description>{m.configure_smtp_to_send_emails()}</Card.Description>
			</Card.Header>
			<Card.Content>
				<AppConfigEmailForm {appConfig} callback={updateAppConfig} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="ldap" id="application-configuration-ldap">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.ldap()}</Card.Title>
				<Card.Description>
					{m.configure_ldap_settings_to_sync_users_and_groups_from_an_ldap_server()}
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<AppConfigLdapForm {appConfig} callback={updateAppConfig} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="dynamic-clients" id="application-configuration-dynamic-clients">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.dynamic_clients()}</Card.Title>
				<Card.Description>{m.configure_dynamic_clients()}</Card.Description>
			</Card.Header>
			<Card.Content>
				<AppConfigDynamicClientsForm {appConfig} callback={updateAppConfig} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="images" id="application-configuration-images">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.images()}</Card.Title>
				<Card.Description>{m.configure_application_images()}</Card.Description>
			</Card.Header>
			<Card.Content>
				<UpdateApplicationImages callback={updateImages} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>
</Tabs.Root>
