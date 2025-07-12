<script lang="ts">
	import * as Pagination from '$lib/components/ui/pagination';
	import AppClientCard from './app-client-card.svelte';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { AccessibleOidcClient } from '$lib/types/oidc.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { LayoutDashboard } from '@lucide/svelte';

	let { data } = $props();
	let apps: Paginated<AccessibleOidcClient> = $state(data.apps);
	let requestOptions: SearchPaginationSortRequest = $state(data.appRequestOptions);

	const oidcService = new OIDCService();

	async function onRefresh(options: SearchPaginationSortRequest) {
		apps = await oidcService.listAccessibleClients(options);
	}

	async function onPageChange(page: number) {
		requestOptions.pagination = { limit: apps.pagination.itemsPerPage, page };
		onRefresh(requestOptions);
	}
</script>

{#snippet appCard(app: AccessibleOidcClient)}
	<AppClientCard {app} />
{/snippet}

<svelte:head>
	<title>{m.my_apps()}</title>
</svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="flex items-center gap-2 text-2xl font-bold">
			<LayoutDashboard class="text-primary/80 size-6" />
			{m.my_apps()}
		</h1>
		<p class="text-muted-foreground mt-2">
			{m.applications_you_have_access_to()}
		</p>
	</div>

	{#if apps.data.length === 0}
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
					{@render appCard(app)}
				{/each}
			</div>

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
