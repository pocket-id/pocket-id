<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import ListPagination from '#lib/components/list-pagination.svelte';
	import { Button, buttonVariants } from '#lib/components/ui/button/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Empty from '#lib/components/ui/empty/index.ts';
	import * as InputGroup from '#lib/components/ui/input-group/index.js';
	import { Separator } from '#lib/components/ui/separator/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OIDCService from '#lib/services/oidc-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
	import type {
		AccessibleOidcClient,
		AuthorizedOidcClient,
		OidcClientMetaData
	} from '#lib/types/oidc.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { ArrowUpDown, ChevronDown, LayoutDashboard, Search, SearchX } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { prefersReducedMotion } from 'svelte/motion';
	import { fade, slide } from 'svelte/transition';
	import AuthorizedOidcClientCard from './authorized-oidc-client-card.svelte';
	import {
		getMyAppsPreferences,
		myAppsPageSizes,
		myAppsPreferences,
		myAppsSortOptions,
		type MyAppsSort
	} from './my-apps-preferences.svelte.ts';

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
	let searchValue = $state('');
	let sort: MyAppsSort = $state(getMyAppsPreferences().sort);
	const hiddenAuthorizedClients = $derived(
		authorizedClientsWithoutLaunchURL.data.map(({ client, lastUsedAt }) => ({
			...client,
			lastUsedAt
		}))
	);
	const oidcService = new OIDCService();

	const animationDuration = $derived(
		$appConfigStore.disableAnimations || prefersReducedMotion.current ? 0 : 200
	);

	const sortLabels: Record<MyAppsSort, () => string> = {
		recentlyUsed: m.recently_used,
		nameAsc: m.name_a_to_z,
		nameDesc: m.name_z_to_a
	};

	// Only unfiltered results tell whether the user has any apps, so the toolbar never unmounts while a search is being typed or cleared
	let hasAnyApps = $state(
		data.clients.pagination.totalItems +
			data.authorizedClientsWithoutLaunchURL.pagination.totalItems >
			0
	);

	async function refreshClients() {
		[clients, authorizedClientsWithoutLaunchURL] = await Promise.all([
			oidcService.listOwnAccessibleClients(requestOptions),
			oidcService.listOwnAuthorizedClients(authorizedClientRequestOptions)
		]);
		if (!requestOptions.search) {
			hasAnyApps =
				clients.pagination.totalItems + authorizedClientsWithoutLaunchURL.pagination.totalItems > 0;
		}
		if (authorizedClientsWithoutLaunchURL.pagination.totalItems === 0) {
			showAllApps = false;
		}
	}

	// Both grids share the search, sort and page size, so any change reloads both from their first page
	async function reloadFromFirstPage({
		search = searchValue,
		limit = getMyAppsPreferences().paginationLimit
	}: { search?: string; limit?: number } = {}) {
		for (const options of [requestOptions, authorizedClientRequestOptions]) {
			options.search = search || undefined;
			options.sort = { ...myAppsSortOptions[sort] };
			options.pagination = { page: 1, limit };
		}

		try {
			await refreshClients();
			// The search is applied together with its results so the empty state never describes stale results
			searchValue = search;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	const onSearch = debounced((search: string) => reloadFromFirstPage({ search }), 300);

	async function onSortChange(value: string) {
		sort = value as MyAppsSort;
		myAppsPreferences.current.sort = sort;
		await reloadFromFirstPage();
	}

	async function onPageSizeChange(size: number) {
		myAppsPreferences.current.paginationLimit = size;
		await reloadFromFirstPage({ limit: size });
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
	<div class="mb-5 flex flex-wrap items-center justify-between gap-3">
		<h1 class="flex items-center gap-2 text-2xl font-bold">
			<LayoutDashboard class="text-primary/80 size-6" />
			{m.my_apps()}
		</h1>
		{#if hasAnyApps}
			<div class="flex w-full items-center gap-2 sm:w-auto">
				<InputGroup.Root class="w-full sm:w-64">
					<InputGroup.Input
						value={searchValue}
						placeholder={m.search()}
						aria-label={m.search_apps()}
						type="search"
						oninput={(e: Event) => onSearch((e.currentTarget as HTMLInputElement).value)}
					/>
					<InputGroup.Addon>
						<Search />
					</InputGroup.Addon>
				</InputGroup.Root>
				<DropdownMenu.Root>
					<DropdownMenu.Trigger
						class={buttonVariants({ variant: 'outline', size: 'icon', class: 'shrink-0' })}
						aria-label={m.sort_by()}
						title={m.sort_by()}
					>
						<ArrowUpDown />
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="end" class="w-48">
						<DropdownMenu.Label>{m.sort_by()}</DropdownMenu.Label>
						<DropdownMenu.RadioGroup value={sort} onValueChange={onSortChange}>
							{#each Object.keys(myAppsSortOptions) as MyAppsSort[] as option (option)}
								<DropdownMenu.RadioItem value={option}
									>{sortLabels[option]()}</DropdownMenu.RadioItem
								>
							{/each}
						</DropdownMenu.RadioGroup>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</div>
		{/if}
	</div>

	{#if clients.data.length === 0 && !showAllApps}
		<Empty.Root class="mt-20">
			<Empty.Header>
				{#if searchValue}
					<Empty.Media variant="icon">
						<SearchX />
					</Empty.Media>
					<Empty.Title>{m.no_apps_match_your_search()}</Empty.Title>
					<Empty.Description>{m.try_a_different_search_term()}</Empty.Description>
				{:else}
					<Empty.Media variant="icon">
						<LayoutDashboard />
					</Empty.Media>
					<Empty.Title>{m.no_apps_available()}</Empty.Title>
					<Empty.Description>
						{m.contact_your_administrator_for_app_access()}
					</Empty.Description>
				{/if}
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
			<div class="grid">
				{#key clients}
					<div
						class="col-start-1 row-start-1 grid gap-3 self-start"
						style="grid-template-columns: repeat(auto-fit, minmax(min(300px, 100%), 1fr));"
						transition:fade={{ duration: animationDuration }}
					>
						{#each clients.data as client (client.id)}
							<AuthorizedOidcClientCard {client} onRevoke={revokeAuthorizedClient} />
						{/each}
						<!-- Gap fix if two elements are present-->
						{#if clients.data.length === 2}
							<div></div>
						{/if}
					</div>
				{/key}
			</div>
		{/if}

		<ListPagination
			pagination={clients.pagination}
			pageSizes={myAppsPageSizes}
			{onPageChange}
			{onPageSizeChange}
			hideWhenSinglePage
		/>

		{#if showAllApps}
			<div transition:slide={{ duration: animationDuration }}>
				{#if clients.data.length > 0}
					<Separator class="my-8" />
				{/if}
				<div class="grid">
					{#key authorizedClientsWithoutLaunchURL}
						<div
							class="col-start-1 row-start-1 grid gap-3 self-start"
							style="grid-template-columns: repeat(auto-fit, minmax(min(300px, 100%), 1fr));"
							transition:fade={{ duration: animationDuration }}
						>
							{#each hiddenAuthorizedClients as client (client.id)}
								<AuthorizedOidcClientCard {client} onRevoke={revokeAuthorizedClient} />
							{/each}
							<!-- Gap fix if two elements are present-->
							{#if hiddenAuthorizedClients.length === 2}
								<div></div>
							{/if}
						</div>
					{/key}
				</div>

				<ListPagination
					pagination={authorizedClientsWithoutLaunchURL.pagination}
					pageSizes={myAppsPageSizes}
					onPageChange={onAuthorizedClientPageChange}
					{onPageSizeChange}
					hideWhenSinglePage
				/>
			</div>
		{/if}
	{/if}

	{#if authorizedClientsWithoutLaunchURL.pagination.totalItems > 0 && clients.data.length !== 0}
		<div class="mt-10 flex justify-center">
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
