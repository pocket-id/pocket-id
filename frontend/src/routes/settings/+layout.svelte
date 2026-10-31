<script lang="ts">
	import { onNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import DocsLink from '#lib/components/docs-link.svelte';
	import AppSidebar from '#lib/components/shell/app-sidebar.svelte';
	import CommandPalette from '#lib/components/shell/command-palette.svelte';
	import { navGroups } from '#lib/components/shell/nav-items.ts';
	import PhoneNav from '#lib/components/shell/phone-nav.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { pageTransition } from '#lib/stores/page-transition-store.svelte.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { isEmailVerificationPending } from '#lib/utils/email-verification-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideTriangleAlert } from '@lucide/svelte';
	import { PersistedState } from 'runed';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';

	let {
		children,
		data
	}: {
		children: Snippet;
		data: LayoutData;
	} = $props();

	const { versionInformation, sqliteStorageWarning, user } = data;

	const isAdmin = $derived(!!($userStore?.isAdmin || user?.isAdmin));
	const verificationPending = $derived(isEmailVerificationPending(user, $appConfigStore));
	const showUpdate = $derived(isAdmin && versionInformation.isUpToDate === false);
	const groups = $derived(navGroups({ isAdmin, verificationPending, features: data.features }));
	const collapsed = new PersistedState('sidebar-collapsed', false);

	// List pages run gutter to gutter, the other pages keep the centered column
	const listPage = $derived(
		[
			'/settings/admin/users',
			'/settings/admin/user-groups',
			'/settings/admin/oidc-clients',
			'/settings/admin/claim-mappings',
			'/settings/admin/apis',
			'/settings/admin/api-keys',
			'/settings/audit-log',
			'/settings/audit-log/global'
		].includes(page.route.id ?? '')
	);

	let mainEl: HTMLElement;

	// Only the content region crossfades between pages, so the shell never moves
	onNavigate((navigation) => {
		if (
			navigation.shallow ||
			!document.startViewTransition ||
			document.documentElement.classList.contains('no-motion') ||
			window.matchMedia('(prefers-reduced-motion: reduce)').matches ||
			!navigation.to?.url.pathname.startsWith('/settings/') ||
			navigation.from?.url.pathname === navigation.to.url.pathname ||
			// Configuration tabs are routes, but switching them swaps only the tab content
			(navigation.to.route.id === '/settings/admin/application-configuration/[category]' &&
				navigation.from?.route.id === navigation.to.route.id)
		) {
			return;
		}

		// onNavigate runs after the load functions resolved, so the transition never delays the new page
		return new Promise((resolve) => {
			pageTransition.running = true;
			// The outgoing column fades out where it was instead of at the new page's scroll position
			mainEl.style.viewTransitionName = 'settings-content-old';
			const transition = document.startViewTransition(async () => {
				mainEl.style.viewTransitionName = '';
				resolve();
				await navigation.complete;
			});
			transition.finished.finally(() => (pageTransition.running = false));
		});
	});
</script>

<Tooltip.Provider>
	<div class="bg-surface flex min-h-dvh">
		<AppSidebar {groups} {versionInformation} {showUpdate} bind:collapsed={collapsed.current} />
		<div class="flex min-w-0 flex-1 flex-col">
			<PhoneNav
				{groups}
				version="Pocket ID {versionInformation.currentVersion}"
				newestVersion={showUpdate ? versionInformation.newestVersion : null}
			/>
			<main
				bind:this={mainEl}
				class={cn(
					'settings-content mx-auto flex w-full flex-col gap-4 px-4 pt-6 pb-[calc(var(--clear-end-phone)+env(safe-area-inset-bottom))] sm:px-8 sm:pt-10 sm:pb-20 xl:px-10',
					!listPage && 'max-w-[calc(var(--col-content)+5rem)]'
				)}
			>
				{#if sqliteStorageWarning && isAdmin}
					<Alert.Root variant="warning">
						<LucideTriangleAlert />
						<Alert.Description>
							{m.sqlite_network_storage_warning()}
							<DocsLink
								href="https://pocket-id.org/docs/configuration/environment-variables#sqlite"
							/>
						</Alert.Description>
					</Alert.Root>
				{/if}
				<!-- SvelteKit reuses the page when only the id changes, so a remount makes its state start from the new entity -->
				{#key page.params.id}
					{@render children()}
				{/key}
			</main>
		</div>
	</div>
	<CommandPalette {groups} features={data.features} isAdmin={isAdmin && !verificationPending} />
</Tooltip.Provider>
