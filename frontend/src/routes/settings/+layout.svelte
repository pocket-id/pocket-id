<script lang="ts">
	import { page } from '$app/stores';
	import userStore from '$lib/stores/user-store';
	import { LucideExternalLink } from 'lucide-svelte';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';

	let {
		children,
		data
	}: {
		children: Snippet;
		data: LayoutData;
	} = $props();

	const { versionInformation } = data;

	let links = $state([
		{ href: '/settings/account', label: 'My Account' },
		{ href: '/settings/audit-log', label: 'Audit Log' }
	]);

	if ($userStore?.isAdmin) {
		links = [
			// svelte-ignore state_referenced_locally
			...links,
			{ href: '/settings/admin/users', label: 'Users' },
			{ href: '/settings/admin/user-groups', label: 'User Groups' },
			{ href: '/settings/admin/oidc-clients', label: 'OIDC Clients' },
			{ href: '/settings/admin/application-configuration', label: 'Application Configuration' }
		];
	}
</script>

<section>
	<div class="bg-muted/40 flex min-h-[calc(100vh-64px)] w-full flex-col justify-between">
		<main
			class="mx-auto flex w-full max-w-[1640px] flex-col gap-x-4 gap-y-10 p-4 md:p-10 lg:flex-row"
		>
			<div class="min-w-[200px] xl:min-w-[250px]">
				<div class="mx-auto grid w-full gap-2">
					<h1 class="mb-5 text-3xl font-semibold">Settings</h1>
				</div>
				<nav class="text-muted-foreground grid gap-4 text-sm">
					{#each links as { href, label }}
						<a {href} class={$page.url.pathname.startsWith(href) ? 'text-primary font-bold' : ''}>
							{label}
						</a>
					{/each}
					{#if $userStore?.isAdmin && versionInformation.isUpToDate === false}
						<a
							href="https://github.com/pocket-id/pocket-id/releases/latest"
							target="_blank"
							class="flex items-center gap-2"
						>
							Update Pocket ID <LucideExternalLink class="my-auto inline-block h-3 w-3" />
						</a>
					{/if}
				</nav>
			</div>
			<div class="flex w-full flex-col gap-5 overflow-x-hidden">
				{@render children()}
			</div>
		</main>
		<div class="flex flex-col items-center">
			<p class="text-muted-foreground py-3 text-xs">
				Powered by <a
					class="text-foreground"
					href="https://github.com/pocket-id/pocket-id"
					target="_blank">Pocket ID</a
				>
				({versionInformation.currentVersion})
			</p>
		</div>
	</div>
</section>
