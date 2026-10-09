<script lang="ts">
	import { onNavigate } from '$app/navigation';
	import EmailVerificationStateBox from '#lib/components/email-verification-state-box.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import Sidebar from '#lib/components/sidebar.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { LucideTriangleAlert } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import type { LayoutData } from './$types';

	let {
		children,
		data
	}: {
		children: Snippet;
		data: LayoutData;
	} = $props();

	const { versionInformation, sqliteStorageWarning, user } = data;

	type NavItem = {
		href?: string;
		label: string;
		children?: NavItem[];
	};

	const items: NavItem[] = [
		{ href: '/settings/account', label: m.my_account() },
		{ href: '/settings/apps', label: m.my_apps() },
		{ href: '/settings/audit-log', label: m.audit_log() }
	];

	const adminChildren: NavItem[] = [
		{ href: '/settings/admin/users', label: m.users() },
		{ href: '/settings/admin/user-groups', label: m.user_groups() },
		{ href: '/settings/admin/oidc-clients', label: m.oidc_clients() },
		{ href: '/settings/admin/oidc-claim-mapping-policies', label: m.claim_mapping_policies() },
		{ href: '/settings/admin/apis', label: m.apis() },
		{ href: '/settings/admin/api-keys', label: m.api_keys() },
		{
			href: '/settings/admin/application-configuration',
			label: m.application_configuration()
		}
	];

	if (user?.isAdmin || $userStore?.isAdmin) {
		items.push({ label: m.administration(), children: adminChildren });
	}

	onNavigate((navigation) => {
		if (
			navigation.shallow ||
			!document.startViewTransition ||
			$appConfigStore.disableAnimations ||
			window.matchMedia('(prefers-reduced-motion: reduce)').matches ||
			!navigation.to?.url.pathname.startsWith('/settings/') ||
			navigation.from?.url.pathname === navigation.to.url.pathname
		) {
			return;
		}

		// onNavigate runs after the load functions resolved, so the transition never delays the new page
		return new Promise((resolve) => {
			document.startViewTransition(async () => {
				resolve();
				await navigation.complete;
			});
		});
	});
</script>

<section>
	<div
		class="bg-muted/40 dark:bg-background flex min-h-[calc(100vh-86px)] w-full flex-col justify-between"
	>
		<main
			in:fade={{ duration: 200 }}
			class="mx-auto flex w-full max-w-[1720px] flex-col gap-x-8 gap-y-8 p-4 md:p-8 lg:flex-row"
		>
			<div class="w-full lg:w-[200px] lg:shrink-0 xl:w-[250px]">
				<div in:fly={{ x: -15, duration: 200 }} class="sticky top-6">
					<Sidebar
						{items}
						storageKey="sidebar-open:settings"
						isAdmin={$userStore?.isAdmin || user?.isAdmin}
						isUpToDate={versionInformation?.isUpToDate}
					/>
				</div>
			</div>

			<div class="settings-content flex w-full flex-col gap-4 overflow-hidden px-2 pb-2">
				{#if sqliteStorageWarning && ($userStore?.isAdmin || user?.isAdmin)}
					<Alert.Root variant="destructive">
						<LucideTriangleAlert />
						<Alert.Description>
							<FormattedMessage message={m.sqlite_storage_warning} />
						</Alert.Description>
					</Alert.Root>
				{/if}
				<EmailVerificationStateBox />
				{@render children()}
			</div>
		</main>
		<div class="animate-fade-in flex flex-col items-center" style="animation-delay: 400ms;">
			<p class="text-muted-foreground py-3 text-xs">
				{m.powered_by()}
				<a
					class="text-foreground transition-all hover:underline"
					href="https://github.com/pocket-id/pocket-id"
					target="_blank">Pocket ID</a
				>
				({versionInformation.currentVersion})
			</p>
		</div>
	</div>
</section>
