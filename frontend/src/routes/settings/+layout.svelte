<script lang="ts">
	import EmailVerificationStateBox from '$lib/components/email-verification-state-box.svelte';
	import FadeWrapper from '$lib/components/fade-wrapper.svelte';
	import Sidebar from '$lib/components/sidebar.svelte';
	import { m } from '$lib/paraglide/messages';
	import userStore from '$lib/stores/user-store';
	import { LucideSettings } from '@lucide/svelte';
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

	const { versionInformation, user } = data;

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
		{ href: '/settings/admin/api-keys', label: m.api_keys() },
		{ href: '/settings/admin/application-configuration', label: m.application_configuration() }
	];

	if (user?.isAdmin || $userStore?.isAdmin) {
		items.push({ label: m.administration(), children: adminChildren });
	}
</script>

<section>
	<div
		class="bg-muted/40 dark:bg-background flex min-h-[calc(100vh-64px)] w-full flex-col justify-between"
	>
		<main
			in:fade={{ duration: 200 }}
			class="mx-auto flex w-full max-w-[1640px] flex-col gap-x-8 gap-y-8 p-4 md:p-8 lg:flex-row"
		>
			<div class="min-w-[200px] xl:min-w-[250px]">
				<div in:fly={{ x: -15, duration: 200 }} class="sticky top-6">
					<div class="mx-auto grid w-full gap-2">
						<h1 class="mb-4 flex items-center gap-2 text-2xl font-semibold">
							<LucideSettings class="size-5" />
							{m.settings()}
						</h1>
					</div>

					<Sidebar
						{items}
						storageKey="sidebar-open:settings"
						isAdmin={$userStore?.isAdmin || user?.isAdmin}
						isUpToDate={versionInformation?.isUpToDate}
					/>
				</div>
			</div>

			<div class="flex w-full flex-col gap-4 overflow-hidden">
				<FadeWrapper>
					<EmailVerificationStateBox />
					{@render children()}
				</FadeWrapper>
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
