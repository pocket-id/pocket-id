<script lang="ts">
	import { goto } from '$app/navigation';
	import DocsLink from '#lib/components/docs-link.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import AppConfigService from '#lib/services/app-config-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AllAppConfig } from '#lib/types/application-configuration.type.ts';
	import { LucideInfo } from '@lucide/svelte';
	import type { Category } from '../categories.ts';
	import AppConfigBrandingForm from '../forms/app-config-branding-form.svelte';
	import AppConfigEmailForm from '../forms/app-config-email-form.svelte';
	import AppConfigFeaturesForm from '../forms/app-config-features-form.svelte';
	import AppConfigGeneralForm from '../forms/app-config-general-form.svelte';
	import AppConfigLdapForm from '../forms/app-config-ldap-form.svelte';
	import AppConfigOidcForm from '../forms/app-config-oidc-form.svelte';
	import AppConfigPasskeysForm from '../forms/app-config-passkeys-form.svelte';
	import AppConfigSignupForm from '../forms/app-config-signup-form.svelte';

	let { data } = $props();

	// Each save replaces it with the server's response, so the next save of the same tab merges into current values
	let appConfig = $derived(data.appConfig);

	const appConfigService = new AppConfigService();

	async function updateAppConfig(updatedAppConfig: Partial<AllAppConfig>) {
		appConfig = await appConfigService.update({ ...appConfig, ...updatedAppConfig });
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
		{ id: 'branding', label: m.config_tab_branding(), intro: m.config_branding_intro }
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

<svelte:head>
	<title>{m.configuration()}</title>
</svelte:head>

<div class="flex flex-col">
	{#if $appConfigStore.uiConfigDisabled}
		<Alert.Root variant="info" class="mb-6">
			<LucideInfo />
			<Alert.Title>{m.config_env_managed_title()}</Alert.Title>
			<Alert.Description>
				{m.config_env_managed_description()}
				<DocsLink
					href="https://pocket-id.org/docs/configuration/environment-variables#overriding-the-ui-configuration"
				/>
			</Alert.Description>
		</Alert.Root>
	{/if}

	<PageHeader title={m.configuration()} />

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
			<p class="text-body text-fg-muted mt-4">
				{current.intro()}
				{#if current.docs}
					<DocsLink href={current.docs} />
				{/if}
			</p>
			<div class="mt-6 flex flex-col gap-10">
				{#if data.category === 'general'}
					<AppConfigGeneralForm {appConfig} callback={updateAppConfig} />
				{:else if data.category === 'features'}
					<AppConfigFeaturesForm features={data.features!} callback={updateAppConfig} />
				{:else if data.category === 'signup'}
					<AppConfigSignupForm
						{appConfig}
						showCustomClaims={!!data.features?.customClaims.enabled}
						callback={updateAppConfig}
					/>
				{:else if data.category === 'passkeys'}
					<AppConfigPasskeysForm {appConfig} callback={updateAppConfig} />
				{:else if data.category === 'email'}
					<AppConfigEmailForm {appConfig} callback={updateAppConfig} />
				{:else if data.category === 'ldap'}
					<AppConfigLdapForm {appConfig} callback={updateAppConfig} />
				{:else if data.category === 'oidc'}
					<AppConfigOidcForm
						{appConfig}
						oidcConfiguration={data.oidcConfiguration!}
						showCimd={!!data.features?.cimd.enabled}
						callback={updateAppConfig}
					/>
				{:else}
					<AppConfigBrandingForm />
				{/if}
			</div>
		</Tabs.Content>
	</Tabs.Root>
</div>
