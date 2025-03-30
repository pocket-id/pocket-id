<script lang="ts">
	import { page } from '$app/state';
	import FadeWrapper from '$lib/components/fade-wrapper.svelte';
	import { m } from '$lib/paraglide/messages';
	import userStore from '$lib/stores/user-store';
	import { LucideExternalLink, LucideSettings } from 'lucide-svelte';
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

	const links = [
		{ href: '/settings/account', label: m.my_account() },
		{ href: '/settings/audit-log', label: m.audit_log() },		
	];

	const adminLinks = [
		{ href: '/settings/admin/users', label: m.users() },
		{ href: '/settings/admin/user-groups', label: m.user_groups() },
		{ href: '/settings/admin/oidc-clients', label: m.oidc_clients() },
		{ href: '/settings/admin/api-keys', label: m.api_keys() },
		{ href: '/settings/admin/application-configuration', label: m.application_configuration() }
	];

	if (user?.isAdmin || $userStore?.isAdmin) {
		links.push(...adminLinks);
	}
</script>

<section>
	<div class="bg-muted/40 flex min-h-[calc(100vh-64px)] w-full flex-col justify-between">
		<main
			in:fade={{ duration: 300 }}
			class="mx-auto flex w-full max-w-[1640px] flex-col gap-x-8 gap-y-8 overflow-hidden p-4 md:p-8 lg:flex-row"
		>
			<div class="min-w-[200px] xl:min-w-[250px]">
				<div in:fly={{ x: -15, duration: 300 }} class="sticky top-6">
					<div class="mx-auto grid w-full gap-2">
						<h1 class="mb-4 flex items-center gap-2 text-2xl font-semibold">
							<LucideSettings class="h-5 w-5" />
							{m.settings()}
						</h1>
					</div>
					<nav class="text-muted-foreground grid gap-2 text-sm">
						{#each links as { href, label }, i}
							<a
								{href}
								class={`animate-fade-in ${
									page.url.pathname.startsWith(href)
										? 'text-primary bg-card rounded-md px-3 py-1.5 font-medium shadow-sm transition-all'
										: 'hover:text-foreground hover:bg-muted/70 rounded-md px-3 py-1.5 transition-all hover:-translate-y-[2px] hover:shadow-sm'
								}`}
								style={`animation-delay: ${150 + i * 75}ms;`}
							>
								{label}
							</a>
						{/each}
						{#if $userStore?.isAdmin && versionInformation.isUpToDate === false}
							<a
								href="https://github.com/pocket-id/pocket-id/releases/latest"
								target="_blank"
								class="animate-fade-in hover:text-foreground hover:bg-muted/70 mt-1 flex items-center gap-2 rounded-md px-3 py-1.5 text-orange-500 transition-all hover:-translate-y-[2px] hover:shadow-sm"
								style={`animation-delay: ${150 + links.length * 75}ms;`}
							>
								{m.update_pocket_id()}
								<LucideExternalLink class="my-auto inline-block h-3 w-3" />
							</a>
						{/if}
					</nav>
				</div>
			</div>
			<div class="flex w-full flex-col gap-4 overflow-hidden">
				<FadeWrapper>
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
