<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Pagination from '$lib/components/ui/pagination';
	import ImageBox from '$lib/components/image-box.svelte';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { AccessibleOidcClient } from '$lib/types/oidc.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { LucideExternalLink, LayoutDashboard, Settings, CheckCircle } from '@lucide/svelte';
	import { cachedApplicationLogo } from '$lib/utils/cached-image-util';
	import { mode } from 'mode-watcher';
	import userStore from '$lib/stores/user-store';

	let { data } = $props();
	let apps = $state(data.apps);
	let requestOptions: SearchPaginationSortRequest = $state(data.appRequestOptions);
	let isLoading = $state(false);
	const isLightMode = $derived(mode.current === 'light');

	const oidcService = new OIDCService();

	async function loadApps() {
		isLoading = true;
		try {
			apps = await oidcService.listAccessibleClients(requestOptions);
		} catch (e) {
			console.error('Failed to load apps:', e);
		} finally {
			isLoading = false;
		}
	}

	async function onPageChange(page: number) {
		requestOptions.pagination = { limit: apps.pagination.itemsPerPage, page };
		await loadApps();
	}

	function getAppUrl(client: AccessibleOidcClient): string {
		// Try to get the main callback URL as the app URL
		return client.callbackURLs?.[0]?.replace(/\/callback.*$/, '') || '#';
	}
</script>

<svelte:head>
	<title>{m.my_apps()}</title>
</svelte:head>

<div class="space-y-6">
	<!-- Page Header -->
	<div class="pb-4">
		<h1 class="flex items-center gap-2 text-2xl font-bold">
			<LayoutDashboard class="text-primary/80 size-6" />
			{m.my_apps()}
		</h1>
		<p class="text-muted-foreground mt-2">
			{m.applications_you_have_access_to()}
		</p>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-16">
			<div class="border-primary h-8 w-8 animate-spin rounded-full border-b-2"></div>
		</div>
	{:else if apps.data.length === 0}
		<div class="py-16 text-center">
			<LayoutDashboard class="text-muted-foreground mx-auto mb-4 size-16" />
			<h3 class="text-muted-foreground mb-2 text-lg font-medium">
				{m.no_apps_available()}
			</h3>
			<p class="text-muted-foreground mx-auto max-w-md text-sm">
				{m.contact_your_administrator_for_app_access()}
			</p>
		</div>
	{:else}
		<div class="space-y-8">
			<div class="grid gap-3 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
				{#each apps.data as app}
					<Card.Root class="border-muted group transition-all duration-200 hover:shadow-md">
						<Card.Content class="flex h-full flex-col p-4">
							<div class="mb-4 flex items-center gap-3">
								<div class="flex-shrink-0">
									<ImageBox
										class="size-10"
										src={app.hasLogo
											? cachedOidcClientLogo.getUrl(app.id)
											: cachedApplicationLogo.getUrl(isLightMode)}
										alt={m.name_logo({ name: app.name })}
									/>
								</div>

								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2">
										<h3 class="text-foreground line-clamp-1 text-sm font-medium">
											{app.name}
										</h3>
										{#if app.isAuthorized}
											<CheckCircle class="size-4 flex-shrink-0 text-green-500" />
										{/if}
									</div>
									<p class="text-muted-foreground line-clamp-1 text-xs">
										{app.callbackURLs?.[0] ? new URL(app.callbackURLs[0]).hostname : 'Application'}
									</p>
								</div>
							</div>

							<div class="mt-auto">
								{#if $userStore?.isAdmin}
									<div class="grid grid-cols-2 gap-2">
										<Button
											href="/settings/admin/oidc-clients/{app.id}"
											size="sm"
											class="w-full"
											variant="outline"
										>
											<Settings class="mr-1 size-3" />
											{m.edit()}
										</Button>
										<Button
											href={getAppUrl(app)}
											target="_blank"
											size="sm"
											class="w-full"
											variant="default"
										>
											{m.launch()}
											<LucideExternalLink class="ml-1 size-3" />
										</Button>
									</div>
								{:else}
									<Button
										href={getAppUrl(app)}
										target="_blank"
										size="sm"
										class="w-full"
										variant={app.isAuthorized ? 'default' : 'outline'}
									>
										{app.isAuthorized ? m.launch() : m.authorize()}
										<LucideExternalLink class="ml-1 size-3" />
									</Button>
								{/if}
							</div>
						</Card.Content>
					</Card.Root>
				{/each}
			</div>

			<!-- Pagination -->
			{#if apps.pagination.totalPages > 1}
				<div class="border-border flex items-center justify-center border-t pt-8">
					<Pagination.Root
						class="mx-0 w-auto"
						count={apps.pagination.totalItems}
						perPage={apps.pagination.itemsPerPage}
						{onPageChange}
						page={apps.pagination.currentPage}
					>
						{#snippet children({ pages })}
							<Pagination.Content class="flex justify-center">
								<Pagination.Item>
									<Pagination.PrevButton />
								</Pagination.Item>
								{#each pages as page (page.key)}
									{#if page.type !== 'ellipsis' && page.value != 0}
										<Pagination.Item>
											<Pagination.Link {page} isActive={apps.pagination.currentPage === page.value}>
												{page.value}
											</Pagination.Link>
										</Pagination.Item>
									{/if}
								{/each}
								<Pagination.Item>
									<Pagination.NextButton />
								</Pagination.Item>
							</Pagination.Content>
						{/snippet}
					</Pagination.Root>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.line-clamp-1 {
		display: -webkit-box;
		-webkit-line-clamp: 1;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
