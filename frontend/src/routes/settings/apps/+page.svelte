<script lang="ts">
	import { onMount } from 'svelte';
	import { PersistedState } from 'runed';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import { Button } from '$lib/components/ui/button';
	import * as Empty from '$lib/components/ui/empty';
	import { Input } from '$lib/components/ui/input';
	import * as Pagination from '$lib/components/ui/pagination';
	import * as Select from '$lib/components/ui/select';
	import { Separator } from '$lib/components/ui/separator';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { ListRequestOptions, Paginated, SortRequest } from '$lib/types/list-request.type';
	import type {
		AccessibleOidcClient,
		AuthorizedOidcClient,
		OidcClientMetaData
	} from '$lib/types/oidc.type';
	import { debounced } from '$lib/utils/debounce-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { cn } from '$lib/utils/style';
	import { ChevronDown, LayoutDashboard } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import AuthorizedOidcClientCard from './authorized-oidc-client-card.svelte';

	type AppPreferences = {
		paginationLimit: number;
		sort: SortRequest;
	};

	const availablePageSizes: number[] = [20, 50, 100];
	const defaultSort: SortRequest = { column: 'lastUsedAt', direction: 'desc' };

	const appPreferences = new PersistedState<AppPreferences>('my-apps-preferences', {
		paginationLimit: 20,
		sort: defaultSort
	});

	let { data } = $props();
	let clients: Paginated<AccessibleOidcClient> = $state(data.clients);
	let authorizedClientsWithoutLaunchURL: Paginated<AuthorizedOidcClient> = $state(
		data.authorizedClientsWithoutLaunchURL
	);
	let requestOptions: ListRequestOptions = $state({
		...data.appRequestOptions,
		sort: appPreferences.current.sort ?? defaultSort,
		pagination: { limit: appPreferences.current.paginationLimit ?? 20, page: 1 }
	});
	let authorizedClientRequestOptions: ListRequestOptions = $state({
		...data.authorizedClientRequestOptions,
		sort: appPreferences.current.sort ?? defaultSort,
		pagination: { limit: appPreferences.current.paginationLimit ?? 20, page: 1 }
	});
	let showAllApps = $state(false);
	let searchValue = $state('');

	const sortOptions = [
		{
			label: m.last_used(),
			value: 'lastUsedAt-desc',
			column: 'lastUsedAt',
			direction: 'desc' as const
		},
		{ label: m.name_asc(), value: 'name-asc', column: 'name', direction: 'asc' as const },
		{ label: m.name_desc(), value: 'name-desc', column: 'name', direction: 'desc' as const }
	];

	let sortValue = $derived(
		`${requestOptions.sort?.column || 'lastUsedAt'}-${requestOptions.sort?.direction || 'desc'}`
	);

	const initialHasApps =
		data.clients.pagination.totalItems > 0 ||
		data.authorizedClientsWithoutLaunchURL.pagination.totalItems > 0;

	const hiddenAuthorizedClients = $derived(
		authorizedClientsWithoutLaunchURL.data.map(({ client, lastUsedAt }) => ({
			...client,
			lastUsedAt
		}))
	);
	const oidcService = new OIDCService();

	onMount(async () => {
		const currentSort = appPreferences.current.sort ?? defaultSort;
		const currentLimit = appPreferences.current.paginationLimit ?? 20;
		if (
			currentSort.column !== 'lastUsedAt' ||
			currentSort.direction !== 'desc' ||
			currentLimit !== 20
		) {
			await refreshClients();
		}
	});

	async function refreshClients() {
		[clients, authorizedClientsWithoutLaunchURL] = await Promise.all([
			oidcService.listOwnAccessibleClients(requestOptions),
			oidcService.listOwnAuthorizedClients(authorizedClientRequestOptions)
		]);
		if (authorizedClientsWithoutLaunchURL.pagination.totalItems === 0) {
			showAllApps = false;
		}
	}

	const onSearch = debounced(async (search: string) => {
		searchValue = search;
		requestOptions.search = search;
		requestOptions.pagination = { limit: requestOptions.pagination?.limit ?? 20, page: 1 };
		authorizedClientRequestOptions.search = search;
		authorizedClientRequestOptions.pagination = {
			limit: authorizedClientRequestOptions.pagination?.limit ?? 20,
			page: 1
		};
		await refreshClients();
	}, 300);

	async function onSortChange(value: string) {
		const option = sortOptions.find((o) => o.value === value);
		if (!option) return;

		const sort: SortRequest = { column: option.column, direction: option.direction };
		appPreferences.current.sort = sort;
		requestOptions.sort = sort;
		requestOptions.pagination = { limit: requestOptions.pagination?.limit ?? 20, page: 1 };
		authorizedClientRequestOptions.sort = sort;
		authorizedClientRequestOptions.pagination = {
			limit: authorizedClientRequestOptions.pagination?.limit ?? 20,
			page: 1
		};
		await refreshClients();
	}

	async function onPageSizeChange(size: number) {
		appPreferences.current.paginationLimit = size;
		requestOptions.pagination = { limit: size, page: 1 };
		authorizedClientRequestOptions.pagination = { limit: size, page: 1 };
		await refreshClients();
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
	<div class="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<h1 class="flex items-center gap-2 text-2xl font-bold">
			<LayoutDashboard class="text-primary/80 size-6" />
			{m.my_apps()}
		</h1>
		{#if initialHasApps || searchValue}
			<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
				<Input
					value={searchValue}
					class="w-full sm:w-64"
					placeholder={m.search()}
					type="text"
					oninput={(e: Event) => onSearch((e.currentTarget as HTMLInputElement).value)}
				/>
				<Select.Root type="single" value={sortValue} onValueChange={onSortChange}>
					<Select.Trigger class="w-full sm:w-44" aria-label={m.sort_by()}>
						{sortOptions.find((o) => o.value === sortValue)?.label}
					</Select.Trigger>
					<Select.Content>
						{#each sortOptions as option (option.value)}
							<Select.Item value={option.value}>{option.label}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
		{/if}
	</div>

	{#if clients.data.length === 0 && !showAllApps}
		<Empty.Root class={searchValue ? 'mt-12' : 'mt-20'}>
			<Empty.Header>
				<Empty.Media variant="icon">
					<LayoutDashboard />
				</Empty.Media>
				<Empty.Title>{searchValue ? m.no_items_found() : m.no_apps_available()}</Empty.Title>
				{#if !searchValue}
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

		{#if clients.data.length > 0}
			<div class="mt-5 flex flex-col-reverse items-center justify-between gap-3 sm:flex-row">
				<div class="flex items-center space-x-2">
					<p class="text-sm font-medium">{m.items_per_page()}</p>
					<Select.Root
						type="single"
						value={clients.pagination.itemsPerPage.toString()}
						onValueChange={(v) => onPageSizeChange(Number(v))}
					>
						<Select.Trigger class="w-20">
							{clients.pagination.itemsPerPage}
						</Select.Trigger>
						<Select.Content>
							{#each availablePageSizes as size (size)}
								<Select.Item value={size.toString()}>{size}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
				{#if clients.pagination.totalPages > 1}
					<Pagination.Root
						class="mx-0 w-auto"
						count={clients.pagination.totalItems}
						perPage={clients.pagination.itemsPerPage}
						{onPageChange}
						page={clients.pagination.currentPage}
					>
						{#snippet children({ pages })}
							<Pagination.Content class="flex justify-end">
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
				{/if}
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
