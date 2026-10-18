<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import ListPagination from '#lib/components/list-pagination.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button, buttonVariants } from '#lib/components/ui/button/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Empty from '#lib/components/ui/empty/index.ts';
	import * as InputGroup from '#lib/components/ui/input-group/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import OIDCService from '#lib/services/oidc-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
	import type { AccessibleOidcClient, AuthorizedOidcClient } from '#lib/types/oidc.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import {
		LucideArrowDownUp,
		LucideBan,
		LucideChevronDown,
		LucideLayoutGrid,
		LucidePlus,
		LucideSearch
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import AppTile from './app-tile.svelte';
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
	let showHiddenApps = $state(false);
	let searchValue = $state('');
	let searchInput = $state<HTMLInputElement | null>(null);
	let sort: MyAppsSort = $state(getMyAppsPreferences().sort);
	const hiddenApps = $derived(
		authorizedClientsWithoutLaunchURL.data.map(({ client, lastUsedAt }) => ({
			...client,
			lastUsedAt
		}))
	);
	const oidcService = new OIDCService();

	const sortLabels: Record<MyAppsSort, () => string> = {
		recentlyUsed: m.recently_used,
		nameAsc: m.name_a_to_z,
		nameDesc: m.name_z_to_a
	};

	const launchableCount = $derived(clients.pagination.totalItems);
	const hiddenCount = $derived(authorizedClientsWithoutLaunchURL.pagination.totalItems);

	// A search opens the apps without a launch link too, so every match is visible
	const hiddenAppsOpen = $derived(showHiddenApps || !!searchValue);

	const countText = $derived.by(() => {
		if (searchValue) {
			const matches = launchableCount + hiddenCount;
			return matches === 1
				? m.app_matches_query({ query: searchValue })
				: m.apps_match_query({ count: matches, query: searchValue });
		}
		const apps =
			launchableCount === 1 ? m.apps_count_one() : m.apps_count({ count: launchableCount });
		return hiddenCount
			? `${apps} · ${m.apps_without_launch_link_count({ count: hiddenCount })}`
			: apps;
	});

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
	}

	// Both lists share the search, sort and page size, so any change reloads both from their first page
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
			// The search is applied together with its results so the count line never describes stale results
			searchValue = search;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	const onSearch = debounced((search: string) => reloadFromFirstPage({ search }), 300);

	async function clearSearch() {
		if (searchInput) searchInput.value = '';
		await reloadFromFirstPage({ search: '' });
		searchInput?.focus();
	}

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

	async function onHiddenAppsPageChange(page: number) {
		authorizedClientRequestOptions.pagination = {
			limit: authorizedClientsWithoutLaunchURL.pagination.itemsPerPage,
			page
		};
		authorizedClientsWithoutLaunchURL = await oidcService.listOwnAuthorizedClients(
			authorizedClientRequestOptions
		);
	}

	function revokeAuthorizedClient(client: AccessibleOidcClient) {
		openConfirmDialog({
			title: m.revoke_app_access_title({ name: client.name, appName: $appConfigStore.appName }),
			message: {
				message: m.revoke_access_description,
				inputs: { clientName: client.name }
			},
			confirm: {
				label: m.revoke(),
				destructive: true,
				icon: LucideBan,
				action: async () => {
					try {
						await oidcService.revokeOwnAuthorizedClient(client.id);
						await refreshClients();
						toast.success(m.app_access_revoked({ name: client.name }));
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

<PageHeader
	title={m.my_apps()}
	description={m.my_apps_description({ appName: $appConfigStore.appName })}
/>

{#if !hasAnyApps}
	<Empty.Root class="mt-6">
		<Empty.Header>
			<Empty.Media variant="icon">
				<LucideLayoutGrid />
			</Empty.Media>
			<Empty.Title>{m.no_apps_yet()}</Empty.Title>
			<Empty.Description>
				{$userStore?.isAdmin ? m.no_apps_yet_admin_description() : m.no_apps_yet_description()}
			</Empty.Description>
		</Empty.Header>
		{#if $userStore?.isAdmin}
			<Empty.Content>
				<Button variant="secondary" href="/settings/admin/oidc-clients?new">
					<LucidePlus />
					{m.connect_an_app()}
				</Button>
			</Empty.Content>
		{/if}
	</Empty.Root>
{:else}
	<div class="mt-6 flex flex-col gap-3">
		<InputGroup.Root>
			<InputGroup.Input
				bind:ref={searchInput}
				placeholder={m.search_apps()}
				aria-label={m.search_apps()}
				aria-describedby="apps-count"
				type="search"
				oninput={(e: Event) => onSearch((e.currentTarget as HTMLInputElement).value)}
			/>
			<InputGroup.Addon>
				<LucideSearch />
			</InputGroup.Addon>
		</InputGroup.Root>
		<div class="flex min-h-(--h-sm) items-center justify-between gap-2">
			<p id="apps-count" role="status" class="text-caption text-fg-muted min-w-0 truncate">
				{countText}
			</p>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger
					class={buttonVariants({ variant: 'ghost', size: 'sm', class: '-me-3' })}
				>
					<LucideArrowDownUp />
					<span class="sr-only">{m.sort_by()}</span>
					<span class="max-sm:sr-only">{sortLabels[sort]()}</span>
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end" class="w-48">
					<DropdownMenu.Label>{m.sort_by()}</DropdownMenu.Label>
					<DropdownMenu.RadioGroup value={sort} onValueChange={onSortChange}>
						{#each Object.keys(myAppsSortOptions) as MyAppsSort[] as option (option)}
							<DropdownMenu.RadioItem value={option}>{sortLabels[option]()}</DropdownMenu.RadioItem>
						{/each}
					</DropdownMenu.RadioGroup>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</div>
	</div>

	<div class="mt-3 flex flex-col gap-6">
		{#if searchValue && launchableCount + hiddenCount === 0}
			<p class="text-control text-fg-muted wrap-anywhere">
				{m.no_apps_match_query({ query: searchValue })}
				<button
					class="inline-link focus-visible:outline-ring rounded-xs focus-visible:outline-2 focus-visible:outline-offset-2"
					onclick={clearSearch}
				>
					{m.clear_search()}
				</button>
			</p>
		{/if}

		{#if clients.data.length > 0}
			<div class="flex flex-col gap-6">
				<ul
					aria-label={m.all_apps()}
					class="grid grid-cols-1 max-sm:shadow-[inset_0_1px_0_var(--border)] sm:grid-cols-[repeat(auto-fill,minmax(200px,1fr))] sm:gap-3"
				>
					{#each clients.data as client (client.id)}
						<AppTile {client} onRevoke={revokeAuthorizedClient} />
					{/each}
				</ul>
				<ListPagination
					pagination={clients.pagination}
					pageSizes={myAppsPageSizes}
					{onPageChange}
					{onPageSizeChange}
				/>
			</div>
		{/if}

		{#if hiddenCount > 0}
			<div>
				<button
					class="text-control text-fg focus-visible:outline-ring -mx-2 inline-flex min-h-(--h-md) items-center gap-1.5 rounded-sm px-2 font-medium focus-visible:outline-2"
					aria-expanded={hiddenAppsOpen}
					aria-controls="apps-hidden"
					disabled={!!searchValue}
					onclick={() => (showHiddenApps = !showHiddenApps)}
				>
					{m.apps_without_launch_link()}
					<LucideChevronDown
						class={cn(
							'text-fg-muted size-4 transition-transform duration-(--dur-base)',
							hiddenAppsOpen && 'rotate-180'
						)}
					/>
				</button>
				{#if hiddenAppsOpen}
					<div
						id="apps-hidden"
						class="transition-motion duration-(--dur-base) starting:-translate-y-(--rise) starting:opacity-0"
					>
						<p class="text-caption text-fg-muted mt-1 mb-3">{m.apps_without_launch_link_note()}</p>
						<ul
							aria-label={m.apps_without_launch_link()}
							class="shadow-[inset_0_1px_0_var(--border)]"
						>
							{#each hiddenApps as client (client.id)}
								<AppTile {client} row onRevoke={revokeAuthorizedClient} />
							{/each}
						</ul>
						<ListPagination
							class="mt-6"
							pagination={authorizedClientsWithoutLaunchURL.pagination}
							pageSizes={myAppsPageSizes}
							onPageChange={onHiddenAppsPageChange}
							{onPageSizeChange}
						/>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
