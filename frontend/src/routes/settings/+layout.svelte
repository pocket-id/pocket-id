<script lang="ts">
	import { onNavigate } from '$app/navigation';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import AppSidebar from '#lib/components/shell/app-sidebar.svelte';
	import { navGroups } from '#lib/components/shell/nav-items.ts';
	import PhoneNav from '#lib/components/shell/phone-nav.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
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
	const groups = $derived(
		navGroups({ isAdmin, verificationPending: isEmailVerificationPending(user, $appConfigStore) })
	);
	const collapsed = new PersistedState('sidebar-collapsed', false);

	// List pages run gutter to gutter, the other pages keep the centered column
	const listPage = $derived(
		[
			'/settings/admin/users',
			'/settings/admin/user-groups',
			'/settings/admin/oidc-clients',
			'/settings/admin/apis',
			'/settings/admin/api-keys',
			'/settings/audit-log',
			'/settings/audit-log/global'
		].includes(page.route.id ?? '')
	);

	let mainEl: HTMLElement;

	// Navigating from the APIs or API keys pages, their tables' page updates included, re-checks their rows, so a first API or key has its row before you leave
	// SvelteKit passes a from without a url when the app starts
	afterNavigate(async ({ from }) => {
		if (from?.url?.pathname.startsWith('/settings/admin/api'))
			recheckedApiRows = await checkApiRows();
	});

	// Only the content region crossfades between pages; the shell never moves (docs/design-system.md, Page transitions)
	onNavigate((navigation) => {
		if (
			navigation.shallow ||
			!document.startViewTransition ||
			document.documentElement.classList.contains('no-motion') ||
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
							<FormattedMessage message={m.sqlite_storage_warning} />
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
</Tooltip.Provider>
