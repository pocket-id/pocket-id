<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import * as Pagination from '$lib/components/ui/pagination';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { AuthorizedOidcClient, OidcClientMetaData } from '$lib/types/oidc.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LayoutDashboard } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { default as AuthorizedOidcClientCard } from './authorized-oidc-client-card.svelte';

	let { data } = $props();
	let authorizedClients: Paginated<AuthorizedOidcClient> = $state(data.authorizedClients);
	let requestOptions: SearchPaginationSortRequest = $state(data.appRequestOptions);

	const oidcService = new OIDCService();

	async function onRefresh(options: SearchPaginationSortRequest) {
		authorizedClients = await oidcService.listAuthorizedClients(options);
	}

	async function onPageChange(page: number) {
		requestOptions.pagination = { limit: authorizedClients.pagination.itemsPerPage, page };
		onRefresh(requestOptions);
	}

	async function revokeAuthorizedClient(client: OidcClientMetaData) {
		openConfirmDialog({
			title: m.revoke_access(),
			message: m.revoke_access_description({
				clientName: client.name
			}),
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					try {
						await oidcService.revokeOwnAuthorizedClient(client.id);
						onRefresh(requestOptions);
						toast.success(
							m.revoke_access_successful({
								clientName: client.name
							})
						);
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<svelte:head>
	<title>{m.my_apps()}</title>
</svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="flex items-center gap-2 text-2xl font-bold">
			<LayoutDashboard class="text-primary/80 size-6" />
			{m.my_apps()}
		</h1>
	</div>

	{#if authorizedClients.data.length === 0}
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
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4">
				{#each authorizedClients.data as authorizedClient}
					<AuthorizedOidcClientCard {authorizedClient} onRevoke={revokeAuthorizedClient} />
				{/each}
			</div>

			{#if authorizedClients.pagination.totalPages > 1}
				<div class="border-border flex items-center justify-center border-t pt-3">
					<Pagination.Root
						class="mx-0 w-auto"
						count={authorizedClients.pagination.totalItems}
						perPage={authorizedClients.pagination.itemsPerPage}
						{onPageChange}
						page={authorizedClients.pagination.currentPage}
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
												isActive={authorizedClients.pagination.currentPage === page.value}
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
</div>
