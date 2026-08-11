<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import { Button } from '$lib/components/ui/button';
	import * as Empty from '$lib/components/ui/empty';
	import * as Pagination from '$lib/components/ui/pagination';
	import { Separator } from '$lib/components/ui/separator';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
	import type {
		AccessibleOidcClient,
		AuthorizedOidcClient,
		OidcClientMetaData
	} from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { cn } from '$lib/utils/style';
	import { ChevronDown, LayoutDashboard } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import AuthorizedOidcClientCard from './authorized-oidc-client-card.svelte';

	let { data } = $props();
	let clients: Paginated<AccessibleOidcClient> = $state(data.clients);
	let requestOptions: ListRequestOptions = $state(data.appRequestOptions);
	let authorizedClientsWithoutLaunchURL: Paginated<AuthorizedOidcClient> = $state(
		data.authorizedClientsWithoutLaunchURL
	);
	let authorizedClientRequestOptions: ListRequestOptions = $state(
		data.authorizedClientRequestOptions
	);
	let showAllApps = $state(false);
	const hiddenAuthorizedClients = $derived(
		authorizedClientsWithoutLaunchURL.data.map(({ client, lastUsedAt }) => ({
			...client,
			lastUsedAt
		}))
	);
	const oidcService = new OIDCService();

	async function refreshClients() {
		[clients, authorizedClientsWithoutLaunchURL] = await Promise.all([
			oidcService.listOwnAccessibleClients(requestOptions),
			oidcService.listOwnAuthorizedClients(authorizedClientRequestOptions)
		]);
		if (authorizedClientsWithoutLaunchURL.pagination.totalItems === 0) {
			showAllApps = false;
		}
	}

	async function onPageChange(page: number) {
		requestOptions.pagination = { limit: clients.pagination.itemsPerPage, page };
		clients = await oidcService.listOwnAccessibleClients(requestOptions);
	}

	async function onAuthorizedClientPageChange(page: number) {
		authorizedClientRequestOptions.pagination = {
			limit: authorizedClientsWithoutLaunchURL.pagination.itemsPerPage,
			page
		};
		authorizedClientsWithoutLaunchURL = await oidcService.listOwnAuthorizedClients(
			authorizedClientRequestOptions
		);
	}

	async function revokeAuthorizedClient(client: OidcClientMetaData) {
		openConfirmDialog({
			title: m.revoke_access(),
			message: {
				message: m.revoke_access_description,
				inputs: { clientName: client.name }
			},
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					try {
						await oidcService.revokeOwnAuthorizedClient(client.id);
						await refreshClients();
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
<div>
	<div>
		<h1 class="flex items-center gap-2 text-2xl font-bold mb-5">
			<LayoutDashboard class="text-primary/80 size-6" />
			{m.my_apps()}
		</h1>
	</div>

	{#if clients.data.length === 0 && !showAllApps}
		<Empty.Root class="mt-20">
			<Empty.Header>
				<Empty.Media variant="icon">
					<LayoutDashboard />
				</Empty.Media>
				<Empty.Title>{m.no_apps_available()}</Empty.Title>
				<Empty.Description>
					{m.contact_your_administrator_for_app_access()}
				</Empty.Description>
			</Empty.Header>
			{#if authorizedClientsWithoutLaunchURL.pagination.totalItems > 0}
				<Empty.Content>
					<Button variant="outline" size="sm" onclick={() => (showAllApps = !showAllApps)}
						>{m.show_hidden_apps()}</Button
					>
				</Empty.Content>
			{/if}
		</Empty.Root>
	{:else}
		{#if clients.data.length > 0}
			<div
				class="grid gap-3"
				style="grid-template-columns: repeat(auto-fit, minmax(min(300px, 100%), 1fr));"
			>
				{#each clients.data as client (client.id)}
					<AuthorizedOidcClientCard {client} onRevoke={revokeAuthorizedClient} />
				{/each}
				<!-- Gap fix if two elements are present-->
				{#if clients.data.length === 2}
					<div></div>
				{/if}
			</div>
		{/if}

		{#if clients.pagination.totalPages > 1}
			<div class="flex items-center justify-center mt-5">
				<Pagination.Root
					class="mx-0 w-auto"
					count={clients.pagination.totalItems}
					perPage={clients.pagination.itemsPerPage}
					{onPageChange}
					page={clients.pagination.currentPage}
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
											isActive={clients.pagination.currentPage === page.value}
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

		{#if showAllApps}
			<div transition:slide={{ duration: 200 }}>
				{#if clients.data.length > 0}
					<Separator class="my-8" />
				{/if}
				<div
					class="grid gap-3"
					style="grid-template-columns: repeat(auto-fit, minmax(min(300px, 100%), 1fr));"
				>
					{#each hiddenAuthorizedClients as client (client.id)}
						<AuthorizedOidcClientCard {client} onRevoke={revokeAuthorizedClient} />
					{/each}
					<!-- Gap fix if two elements are present-->
					{#if hiddenAuthorizedClients.length === 2}
						<div></div>
					{/if}
				</div>

				{#if authorizedClientsWithoutLaunchURL.pagination.totalPages > 1}
					<div class="flex items-center justify-center mt-5">
						<Pagination.Root
							class="mx-0 w-auto"
							count={authorizedClientsWithoutLaunchURL.pagination.totalItems}
							perPage={authorizedClientsWithoutLaunchURL.pagination.itemsPerPage}
							onPageChange={onAuthorizedClientPageChange}
							page={authorizedClientsWithoutLaunchURL.pagination.currentPage}
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
													isActive={authorizedClientsWithoutLaunchURL.pagination.currentPage ===
														page.value}
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
	{/if}

	{#if authorizedClientsWithoutLaunchURL.pagination.totalItems > 0 && clients.data.length !== 0}
		<div class="flex justify-center mt-10">
			<Button
				variant="ghost"
				class="text-muted-foreground"
				onclick={() => (showAllApps = !showAllApps)}
			>
				{showAllApps ? m.hide_all_apps() : m.show_all_apps()}
				({authorizedClientsWithoutLaunchURL.pagination.totalItems})
				<ChevronDown
					data-icon="inline-end"
					class={cn('transition-transform duration-200', showAllApps && 'rotate-180 transform')}
				/>
			</Button>
		</div>
	{/if}
</div>
