<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Pagination from '$lib/components/ui/pagination';
	import ImageBox from '$lib/components/image-box.svelte';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { OidcClient } from '$lib/types/oidc.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { LucideExternalLink, LayoutDashboard } from '@lucide/svelte';
	import { cachedApplicationLogo } from '$lib/utils/cached-image-util';
	import { mode } from 'mode-watcher';

	let { data } = $props();
	let apps: Paginated<OidcClient> = $state(data.apps);
	let requestOptions: SearchPaginationSortRequest = $state(data.appRequestOptions);
	let isLoading = $state(false);
	const isLightMode = $derived(mode.current === 'light');

	const oidcService = new OIDCService();

	async function loadApps() {
		isLoading = true;
		try {
			apps = await oidcService.listClients(requestOptions);
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

	function getAppUrl(client: OidcClient): string {
		// Try to get the main callback URL as the app URL
		return client.callbackURLs?.[0]?.replace(/\/callback.*$/, '') || '#';
	}
</script>

<svelte:head>
	<title>{m.my_apps()}</title>
</svelte:head>

<div class="space-y-6">
	<Card.Root>
		<Card.Header class="pb-4">
			<Card.Title class="flex items-center gap-2">
				<LayoutDashboard class="text-primary/80 size-5" />
				{m.my_apps()}
			</Card.Title>
			<Card.Description>
				{m.applications_you_have_access_to()}
			</Card.Description>
		</Card.Header>
		<Card.Content class="p-0">
			{#if isLoading}
				<div class="flex items-center justify-center py-16">
					<div class="border-primary h-8 w-8 animate-spin rounded-full border-b-2"></div>
				</div>
			{:else if apps.data.length === 0}
				<div class="px-6 py-16 text-center">
					<LayoutDashboard class="text-muted-foreground mx-auto mb-4 size-16" />
					<h3 class="text-muted-foreground mb-2 text-lg font-medium">
						{m.no_apps_available()}
					</h3>
					<p class="text-muted-foreground mx-auto max-w-md text-sm">
						{m.contact_your_administrator_for_app_access()}
					</p>
				</div>
			{:else}
				<div class="p-4 pt-0">
					<div class="grid gap-3 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
						{#each apps.data as app}
							<Card.Root class="border-muted group transition-all duration-200 hover:shadow-md">
								<Card.Content class="p-4">
									<div class="flex items-center gap-3">
										<!-- App Icon -->
										<div class="flex-shrink-0">
											<ImageBox
												class="ring-border size-10 rounded-lg ring-1"
												src={app.hasLogo
													? cachedOidcClientLogo.getUrl(app.id)
													: cachedApplicationLogo.getUrl(isLightMode)}
												alt={m.name_logo({ name: app.name })}
											/>
										</div>

										<!-- App Info & Button -->
										<div class="min-w-0 flex-1">
											<div class="flex items-center justify-between gap-2">
												<div class="min-w-0 flex-1">
													<h3 class="text-foreground line-clamp-1 text-sm font-medium">
														{app.name}
													</h3>
													<p class="text-muted-foreground line-clamp-1 text-xs">
														{app.callbackURLs?.[0]
															? new URL(app.callbackURLs[0]).hostname
															: 'Application'}
													</p>
												</div>

												<Button
													href={getAppUrl(app)}
													target="_blank"
													size="sm"
													class="flex-shrink-0"
													variant="outline"
												>
													{m.launch()}
													<LucideExternalLink class="ml-1 size-3" />
												</Button>
											</div>
										</div>
									</div>
								</Card.Content>
							</Card.Root>
						{/each}
					</div>

					<!-- Pagination -->
					{#if apps.pagination.totalPages > 1}
						<div class="border-border mt-8 flex items-center justify-center border-t pt-8">
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
													<Pagination.Link
														{page}
														isActive={apps.pagination.currentPage === page.value}
													>
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
		</Card.Content>
	</Card.Root>
</div>

<style>
	.line-clamp-1 {
		display: -webkit-box;
		-webkit-line-clamp: 1;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
