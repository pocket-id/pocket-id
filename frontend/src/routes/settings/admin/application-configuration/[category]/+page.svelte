<script lang="ts">
	import { goto } from '$app/navigation';
	import DocsLink from '#lib/components/docs-link.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import * as Alert from '@pocket-id/ui/alert';
	import * as Tabs from '@pocket-id/ui/tabs';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import InfoIcon from '~icons/solar/info-circle-bold-duotone';
	import { fade } from 'svelte/transition';
	import type { Category } from '../categories.ts';
	import AppConfigAppearanceForm from '../forms/appearance/app-config-appearance-form.svelte';
	import AppConfigEmailForm from '../forms/app-config-email-form.svelte';
	import AppConfigFeaturesForm from '../forms/app-config-features-form.svelte';
	import AppConfigGeneralForm from '../forms/app-config-general-form.svelte';
	import AppConfigLdapForm from '../forms/app-config-ldap-form.svelte';
	import AppConfigOidcForm from '../forms/app-config-oidc-form.svelte';
	import AppConfigPasskeysForm from '../forms/app-config-passkeys-form.svelte';
	import AppConfigSignupForm from '../forms/app-config-signup-form.svelte';

	let { data } = $props();

	// Each save replaces it with the server's response, so the forms show the saved values
	let appConfig = $derived(data.appConfig);

	const appConfigService = new AppConfigService();

	async function updateAppConfig(updatedAppConfig: Partial<AllAppConfig>) {
		appConfig = await appConfigService.update(updatedAppConfig);
		appConfigStore.set({ ...$appConfigStore, ...appConfig });
	}

	const appName = $derived($appConfigStore.appName);

	const tabs: { id: Category; label: string; intro: () => string; docs?: string }[] = $derived([
		{
			id: 'general',
			label: m.general(),
			intro: () => m.config_general_intro({ appName }),
			docs: 'https://pocket-id.org/docs/configuration/environment-variables#general'
		},
		{
			id: 'features',
			label: m.config_tab_features(),
			intro: () => m.config_features_intro({ appName })
		},
		{
			id: 'signup',
			label: m.config_tab_signup(),
			intro: m.config_signup_intro,
			docs: 'https://pocket-id.org/docs/setup/user-management#signup-links'
		},
		{
			id: 'passkeys',
			label: m.passkeys(),
			intro: m.config_passkeys_intro,
			docs: 'https://pocket-id.org/docs/guides/sign-in-methods#passkey'
		},
		{
			id: 'email',
			label: m.email(),
			intro: () => m.config_email_intro({ appName }),
			docs: 'https://pocket-id.org/docs/configuration/environment-variables#email'
		},
		{
			id: 'ldap',
			label: m.ldap(),
			intro: m.config_ldap_intro,
			docs: 'https://pocket-id.org/docs/configuration/ldap#set-up-ldap-in-the-admin-ui'
		},
		{
			id: 'oidc',
			label: m.oidc(),
			intro: () =>
				data.features?.cimd.enabled
					? m.config_oidc_intro({ appName })
					: m.config_oidc_intro_basic(),
			docs: 'https://pocket-id.org/docs/setup/connect-an-app'
		},
		{
			id: 'appearance',
			label: m.appearance(),
			intro: () => `How ${appName} looks for everyone. Changes apply when you save.`
		}
	]);

	// LDAP only has a tab while its feature is on, unless someone opened it by its URL
	const visibleTabs = $derived(
		tabs.filter(
			(tab) => tab.id !== 'ldap' || data.features?.ldap.enabled || data.category === 'ldap'
		)
	);
	const current = $derived(tabs.find((tab) => tab.id === data.category)!);

	// A tab change is a navigation, so the unsaved-changes bar can refuse it and the tab row then stays on this tab
	function openCategory(category: string) {
		void goto(`/settings/admin/application-configuration/${category}`);
	}
</script>

<!-- The intros of two tabs crossfade in one grid cell, so the header keeps its place while one replaces the other -->
{#snippet intro()}
	<span class="grid">
		{#key data.category}
			<span
				class="col-start-1 row-start-1"
				in:fade={{ duration: cssMs('--dur-base'), delay: cssMs('--dur-fast') }}
				out:fade={{ duration: cssMs('--dur-fast') }}
			>
				{current.intro()}
				{#if current.docs}
					<DocsLink href={current.docs} />
				{/if}
			</span>
		{/key}
	</span>
{/snippet}

<svelte:head>
	<title>{m.configuration()}</title>
</svelte:head>

<div class="flex flex-col">
	{#if $appConfigStore.uiConfigDisabled}
		<Alert.Root variant="info" class="mb-6">
			<InfoIcon />
			<Alert.Title>{m.config_env_managed_title()}</Alert.Title>
			<Alert.Description>
				{m.config_env_managed_description()}
				<DocsLink
					href="https://pocket-id.org/docs/configuration/environment-variables#overriding-the-ui-configuration"
				/>
			</Alert.Description>
		</Alert.Root>
	{/if}

	<PageHeader title={m.configuration()} description={intro} />

	<Tabs.Root
		bind:value={() => data.category, openCategory}
		activationMode="manual"
		class="mt-4 gap-0"
	>
		<div class="flex shadow-[inset_0_-1px_0_var(--border)]">
			<div class="scroll-fade-x min-w-0 scroll-px-6 [scrollbar-width:none] overflow-x-auto">
				<Tabs.List
					variant="line"
					aria-label={m.configuration_categories()}
					class="w-auto shadow-none max-sm:gap-4"
				>
					{#each visibleTabs as tab (tab.id)}
						<Tabs.Trigger value={tab.id}>{tab.label}</Tabs.Trigger>
					{/each}
				</Tabs.List>
			</div>
		</div>

		<Tabs.Content value={data.category}>
			{#key data.category}
				<div class="mt-8 flex flex-col gap-4" in:fade={{ duration: cssMs('--dur-base') }}>
					{#if data.category === 'general'}
						<AppConfigGeneralForm
							{appConfig}
							envOverrides={data.envOverrides}
							callback={updateAppConfig}
						/>
					{:else if data.category === 'features'}
						<AppConfigFeaturesForm
							features={data.features!}
							envOverrides={data.envOverrides}
							callback={updateAppConfig}
						/>
					{:else if data.category === 'signup'}
						<AppConfigSignupForm
							{appConfig}
							envOverrides={data.envOverrides}
							showCustomClaims={!!data.features?.customClaims.enabled}
							callback={updateAppConfig}
						/>
					{:else if data.category === 'passkeys'}
						<AppConfigPasskeysForm
							{appConfig}
							envOverrides={data.envOverrides}
							callback={updateAppConfig}
						/>
					{:else if data.category === 'email'}
						<AppConfigEmailForm
							{appConfig}
							envOverrides={data.envOverrides}
							callback={updateAppConfig}
						/>
					{:else if data.category === 'ldap'}
						<AppConfigLdapForm
							{appConfig}
							envOverrides={data.envOverrides}
							callback={updateAppConfig}
						/>
					{:else if data.category === 'oidc'}
						<AppConfigOidcForm
							{appConfig}
							envOverrides={data.envOverrides}
							oidcConfiguration={data.oidcConfiguration!}
							showCimd={!!data.features?.cimd.enabled}
							callback={updateAppConfig}
						/>
					{:else}
						<AppConfigAppearanceForm envOverrides={data.envOverrides} callback={updateAppConfig} />
					{/if}
				</div>
			{/key}
		</Tabs.Content>
	</Tabs.Root>
</div>
