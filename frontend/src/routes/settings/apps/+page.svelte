<script lang="ts">
	import AdvancedDisclosure from '#lib/components/advanced-disclosure.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import { Button, buttonVariants } from '@pocket-id/ui/button';
	import * as DropdownMenu from '@pocket-id/ui/dropdown-menu';
	import * as InputGroup from '@pocket-id/ui/input-group';
	import { m } from '#lib/paraglide/messages.js';
	import OIDCService from '#lib/services/oidc-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { Paginated } from '#lib/types/list-request.type.ts';
	import type { AccessibleOidcClient, AuthorizedOidcClient } from '#lib/types/oidc.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import ArrowUpRightIcon from '~icons/solar/arrow-right-up-linear';
	import BanIcon from '~icons/solar/forbidden-circle-bold';
	import PinIcon from '~icons/solar/pin-bold-duotone';
	import PlusIcon from '~icons/solar/add-circle-bold';
	import SearchIcon from '~icons/solar/magnifier-linear';
	import SortIcon from '~icons/solar/sort-vertical-linear';
	import { formatDistanceToNow } from 'date-fns';
	import { PersistedState } from 'runed';
	import { flip } from 'svelte/animate';
	import { quintOut } from 'svelte/easing';
	import { toast } from 'svelte-sonner';
	import AppMenu from './app-menu.svelte';
	import {
		getMyAppsSort,
		myAppsSort,
		myAppsSortOptions,
		type MyAppsSort
	} from './my-apps-sort.svelte.ts';

	let { data } = $props();

	const oidcService = new OIDCService();
	const requestOptions = data.appRequestOptions;
	const otherRequestOptions = data.authorizedClientRequestOptions;

	let clients: Paginated<AccessibleOidcClient> = $state(data.clients);
	let authorizedClients: Paginated<AuthorizedOidcClient> = $state(
		data.authorizedClientsWithoutLaunchURL
	);
	let searchValue = $state('');
	let showOtherApps = $state(false);
	let searchInput = $state<HTMLInputElement | null>(null);

	// Pins live in this browser only, as the IDs of apps with a launch link in the order the user arranged them
	const pinnedIds = new PersistedState<string[]>('my-apps-pinned', []);

	// Apps the user signed in to that have no launch link, in the shape of the launchable ones
	const otherApps = $derived(
		authorizedClients.data.map(({ client, lastUsedAt }) => ({ ...client, lastUsedAt }))
	);
	// A pin whose app is no longer listed, because access was removed or it lost its launch link, stays stored but hidden
	const pinnedApps = $derived(
		pinnedIds.current
			.map((id) => clients.data.find((client) => client.id === id))
			.filter((client) => !!client)
	);
	// A search hides the pinned row, so only then does the list repeat the pinned apps
	const listedApps = $derived(
		searchValue
			? clients.data
			: clients.data.filter((client) => !pinnedIds.current.includes(client.id))
	);
	const hasMoreApps = $derived(clients.pagination.currentPage < clients.pagination.totalPages);
	const noMatches = $derived(
		!!searchValue && clients.pagination.totalItems + authorizedClients.pagination.totalItems === 0
	);

	// Only unfiltered results tell whether the user has any apps, so the page never swaps to the empty state while a search is typed
	let hasAnyApps = $state(
		data.clients.pagination.totalItems +
			data.authorizedClientsWithoutLaunchURL.pagination.totalItems >
			0
	);

	const hour = new Date().getHours();
	const greeting = (name: string) =>
		hour >= 5 && hour < 12
			? m.good_morning_name({ name })
			: hour >= 12 && hour < 18
				? m.good_afternoon_name({ name })
				: m.good_evening_name({ name });

	async function reload(search = searchValue) {
		for (const options of [requestOptions, otherRequestOptions]) {
			options.search = search || undefined;
			options.pagination!.page = 1;
		}
		try {
			[clients, authorizedClients] = await Promise.all([
				oidcService.listOwnAccessibleClients(requestOptions),
				oidcService.listOwnAuthorizedClients(otherRequestOptions)
			]);
			// The search is applied together with its results so the no-match line never describes stale results
			searchValue = search;
			// A search that matches apps without a launch link opens their section, so no match stays tucked away
			if (search && authorizedClients.data.length > 0) showOtherApps = true;
			if (!search) {
				hasAnyApps = clients.pagination.totalItems + authorizedClients.pagination.totalItems > 0;
			}
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	const onSearch = debounced((search: string) => reload(search), 300);

	let sort: MyAppsSort = $state(getMyAppsSort());
	const sortLabels: Record<MyAppsSort, () => string> = {
		nameAsc: m.name_a_to_z,
		nameDesc: m.name_z_to_a,
		recentlyUsed: m.recently_used
	};

	async function onSortChange(value: string) {
		sort = value as MyAppsSort;
		myAppsSort.current = sort;
		for (const options of [requestOptions, otherRequestOptions]) {
			options.sort = { ...myAppsSortOptions[sort] };
		}
		await reload();
	}

	async function clearSearch() {
		if (searchInput) searchInput.value = '';
		await reload('');
		searchInput?.focus();
	}

	async function loadMoreApps() {
		requestOptions.pagination!.page = clients.pagination.currentPage + 1;
		const next = await oidcService.listOwnAccessibleClients(requestOptions);
		clients = { ...next, data: [...clients.data, ...next.data] };
	}

	async function loadMoreOtherApps() {
		otherRequestOptions.pagination!.page = authorizedClients.pagination.currentPage + 1;
		const next = await oidcService.listOwnAuthorizedClients(otherRequestOptions);
		authorizedClients = { ...next, data: [...authorizedClients.data, ...next.data] };
	}

	// Pinning moves an app between the tiles and the list, so the leaving copy glides toward the arriving one while they crossfade
	// Unlike Svelte's crossfade it never scales between the two shapes, which would squash the text of a row turning into a tile
	const sending = new Map<string, Element>();
	const receiving = new Map<string, Element>();
	const send = travel(sending, receiving, false);
	const receive = travel(receiving, sending, true);

	function travel(own: Map<string, Element>, counterparts: Map<string, Element>, intro: boolean) {
		return (node: Element, { key }: { key: string }) => {
			own.set(key, node);
			// The deferred config runs once both copies are registered, so each finds the other
			return () => {
				const counterpart = counterparts.get(key);
				if (!counterpart) {
					own.delete(key);
					return { duration: 0 };
				}
				counterparts.delete(key);
				const from = counterpart.getBoundingClientRect();
				const to = node.getBoundingClientRect();
				const dx = from.left - to.left;
				const dy = from.top - to.top;
				// A leaving item is held in place by an inline transform, which the animation would otherwise replace
				const base = getComputedStyle(node).transform;
				const held = base === 'none' ? '' : base;
				return {
					duration: cssMs('--dur-unfold'),
					easing: quintOut,
					// The leaving copy is gone before the arriving one is fully in, so the two shapes never show side by side
					css: (t: number, u: number) =>
						`opacity: ${intro ? Math.min(1, t * 2) : Math.max(0, (t - 0.6) / 0.4)}; transform: ${held} translate(${u * dx}px, ${u * dy}px) scale(${intro ? 0.96 + 0.04 * t : 1})`
				};
			};
		};
	}

	const slide = () => ({ duration: cssMs('--dur-unfold'), easing: quintOut });

	function togglePin(client: AccessibleOidcClient) {
		pinnedIds.current = pinnedIds.current.includes(client.id)
			? pinnedIds.current.filter((id) => id !== client.id)
			: [...pinnedIds.current, client.id];
	}

	// Dragging a pinned tile reorders the pins live, and a drag that ends outside the tiles or is cancelled restores the order it started with
	let draggedId: string | null = null;
	let placeholderId = $state<string | null>(null);
	let idsBeforeDrag: string[] = [];

	function onPinDragStart(e: DragEvent, client: AccessibleOidcClient) {
		draggedId = client.id;
		idsBeforeDrag = pinnedIds.current;
		e.dataTransfer!.effectAllowed = 'move';
		e.dataTransfer!.setData('text/uri-list', client.launchURL!);
		// The browser snapshots the tile for the drag image after this handler, so it only turns into a placeholder from the next frame
		requestAnimationFrame(() => (placeholderId = draggedId));
	}

	function onPinDragOver(e: DragEvent & { currentTarget: HTMLUListElement }) {
		if (!draggedId) return;
		e.preventDefault();

		// Hit-test the layout boxes, which ignore the flip transforms, so a tile sliding past the pointer does not swap back
		const list = e.currentTarget.getBoundingClientRect();
		const x = e.clientX - list.left;
		const y = e.clientY - list.top;
		const target = [...e.currentTarget.querySelectorAll<HTMLElement>(':scope > li')].find(
			(li) =>
				x >= li.offsetLeft &&
				x < li.offsetLeft + li.offsetWidth &&
				y >= li.offsetTop &&
				y < li.offsetTop + li.offsetHeight
		)?.dataset.id;
		if (!target || target === draggedId) return;

		// Moving within the full list keeps the hidden pins of unlisted apps in place
		const ids = pinnedIds.current.filter((id) => id !== draggedId);
		ids.splice(pinnedIds.current.indexOf(target), 0, draggedId);
		pinnedIds.current = ids;
	}

	function onPinDragEnd(e: DragEvent) {
		if (e.dataTransfer?.dropEffect === 'none') pinnedIds.current = idsBeforeDrag;
		draggedId = null;
		placeholderId = null;
	}

	function revoke(client: AccessibleOidcClient) {
		openConfirmDialog({
			title: m.revoke_app_access_title({ name: client.name, appName: $appConfigStore.appName }),
			message: { message: m.revoke_access_description, inputs: { clientName: client.name } },
			confirm: {
				label: m.revoke(),
				destructive: true,
				icon: BanIcon,
				action: async () => {
					try {
						await oidcService.revokeOwnAuthorizedClient(client.id);
						await reload();
						toast.success(m.app_access_revoked({ name: client.name }));
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

{#snippet appRow(client: AccessibleOidcClient)}
	<OidcClientAvatar id={client.id} name={client.name} hasLogo={client.hasLogo} size={32} />
	{#if client.launchURL}
		<a
			href={client.launchURL}
			target="_blank"
			rel="noopener noreferrer"
			class="text-control text-fg focus-visible:after:outline-ring min-w-0 flex-1 truncate font-medium outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:outline-2"
		>
			{client.name}
		</a>
	{:else}
		<span class="text-control text-fg min-w-0 flex-1 truncate font-medium">{client.name}</span>
	{/if}
	<AppMenu
		{client}
		pinned={pinnedIds.current.includes(client.id)}
		onTogglePin={client.launchURL ? () => togglePin(client) : undefined}
		onRevoke={revoke}
		class="opacity-0 group-focus-within/app:opacity-100 group-hover/app:opacity-100 data-[state=open]:opacity-100 [@media(hover:none)]:opacity-100"
	/>
{/snippet}

<svelte:head>
	<title>{m.my_apps()}</title>
</svelte:head>

<header class="flex flex-wrap items-start justify-between gap-x-6 gap-y-4">
	<div class="flex min-w-0 flex-col gap-1">
		<h1 class="font-display text-display-l text-fg max-sm:text-display-m wrap-anywhere">
			{greeting($userStore?.firstName ?? '')}
		</h1>
		{#if hasAnyApps}
			<p class="text-body text-fg-muted">{m.where_would_you_like_to_go()}</p>
		{/if}
	</div>
	{#if hasAnyApps}
		<div class="flex w-full items-center gap-2 sm:w-auto">
			<InputGroup.Root class="min-w-0 flex-1 sm:w-64 sm:flex-none">
				<InputGroup.Input
					bind:ref={searchInput}
					type="search"
					placeholder={m.search_apps()}
					aria-label={m.search_apps()}
					oninput={(e: Event) => onSearch((e.currentTarget as HTMLInputElement).value)}
				/>
				<InputGroup.Addon>
					<SearchIcon />
				</InputGroup.Addon>
			</InputGroup.Root>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger
					class={buttonVariants({ variant: 'secondary', size: 'icon', class: 'shrink-0' })}
					aria-label={m.sort_by()}
				>
					<SortIcon />
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
	{/if}
</header>

{#if !hasAnyApps}
	<!-- Ghost tiles in the shape of the app grid, matching the ghost rows of the admin lists' empty states -->
	<div class="mt-10">
		<div
			aria-hidden="true"
			class="pointer-events-none grid grid-cols-2 gap-3 mask-b-from-30% sm:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]"
		>
			{#each ['w-24', 'w-30', 'w-20', 'w-28'] as name, i (i)}
				<div
					class="bg-surface flex min-h-[140px] flex-col gap-2 rounded-md p-4 shadow-[inset_0_0_0_1px_var(--border)]"
				>
					<div class="bg-fill-selected mb-1 size-10 rounded-[8px]"></div>
					<div class="bg-fill-selected h-2 max-w-full rounded-xs {name}"></div>
					<div class="bg-fill-selected h-1.5 w-16 rounded-xs"></div>
				</div>
			{/each}
		</div>
		<div class="-mt-6 flex flex-col items-center gap-2 pb-12 text-center">
			{#if $userStore?.isAdmin}
				<h2 class="text-title">{m.connect_your_first_app()}</h2>
				<p class="text-body text-fg-muted max-w-[380px]">{m.no_apps_yet_admin_description()}</p>
				<div class="mt-4 flex flex-wrap items-center justify-center gap-4">
					<Button href="/settings/admin/oidc-clients?new">
						<PlusIcon />
						{m.connect_an_app()}
					</Button>
					<Button
						variant="link"
						class="gap-1"
						href="https://pocket-id.org/docs/client-examples"
						target="_blank"
						rel="noreferrer"
					>
						{m.docs()}
						<ArrowUpRightIcon />
					</Button>
				</div>
			{:else}
				<h2 class="text-title">{m.no_apps_yet()}</h2>
				<p class="text-body text-fg-muted max-w-[380px]">{m.no_apps_yet_description()}</p>
			{/if}
		</div>
	</div>
{:else}
	{#if !searchValue && clients.data.length > 0}
		<section class="mt-10" aria-labelledby="apps-pinned">
			<h2 id="apps-pinned" class="text-heading text-fg mb-3">{m.pinned()}</h2>
			<!-- The list stays mounted while empty so the first pin and the last unpin still animate -->
			<ul
				class="relative grid grid-cols-2 gap-3 lg:grid-cols-4"
				ondragover={onPinDragOver}
				ondrop={(e) => e.preventDefault()}
			>
				{#each pinnedApps as client (client.id)}
					<li
						in:receive={{ key: client.id }}
						out:send={{ key: client.id }}
						animate:flip={slide()}
						data-id={client.id}
						draggable={pinnedApps.length > 1}
						ondragstart={(e) => onPinDragStart(e, client)}
						ondragend={onPinDragEnd}
						class={[
							'group/app bg-surface hover:shadow-popover relative flex flex-col gap-6 rounded-lg p-4 shadow-[inset_0_0_0_1px_var(--border)] transition-[box-shadow,translate,scale,opacity] duration-(--dur-base) hover:-translate-y-0.5 has-[a:active]:scale-(--press-scale)',
							placeholderId === client.id && 'opacity-40'
						]}
					>
						<div class="flex items-start justify-between">
							<OidcClientAvatar
								id={client.id}
								name={client.name}
								hasLogo={client.hasLogo}
								size={40}
							/>
							<AppMenu
								{client}
								pinned
								onTogglePin={() => togglePin(client)}
								onRevoke={revoke}
								class="-me-1.5 -mt-1.5 opacity-0 group-focus-within/app:opacity-100 group-hover/app:opacity-100 data-[state=open]:opacity-100 [@media(hover:none)]:opacity-100"
							/>
						</div>
						<div class="flex min-w-0 flex-col">
							<!-- The name link stretches over the whole tile so the tile is one hit target, and the menu button sits above it -->
							<!-- It is not draggable itself so a drag on it moves the tile instead of the link -->
							<a
								href={client.launchURL}
								draggable="false"
								target="_blank"
								rel="noopener noreferrer"
								class="text-control text-fg focus-visible:after:outline-ring truncate font-medium outline-none after:absolute after:inset-0 after:rounded-lg focus-visible:after:outline-2"
							>
								{client.name}
							</a>
							<span class="text-caption text-fg-muted truncate">
								{client.lastUsedAt
									? m.app_used_ago({
											time: formatDistanceToNow(client.lastUsedAt, { addSuffix: true })
										})
									: m.not_signed_in_yet()}
							</span>
						</div>
					</li>
				{/each}
			</ul>
			{#if pinnedApps.length === 0}
				<div
					class="text-control text-fg-muted border-border-strong transition-motion flex items-center gap-3 rounded-lg border border-dashed px-4 py-5 duration-(--dur-slow) starting:opacity-0"
				>
					<PinIcon class="size-5 shrink-0" />
					<p>{m.pin_apps_hint()}</p>
				</div>
			{/if}
		</section>
	{/if}

	{#if searchValue || listedApps.length > 0 || hasMoreApps}
		<section class="mt-10" aria-labelledby="apps-all">
			<h2 id="apps-all" class="text-heading text-fg mb-3">{m.all_apps()}</h2>

			{#if noMatches}
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

			<ul
				aria-label={m.all_apps()}
				class="relative grid grid-cols-1 gap-x-3 sm:grid-cols-2 lg:grid-cols-3"
			>
				{#each listedApps as client (client.id)}
					<li
						in:receive={{ key: client.id }}
						out:send={{ key: client.id }}
						animate:flip={slide()}
						class="group/app hover:bg-fill-hover relative -mx-2 flex min-h-14 items-center gap-3 rounded-md px-2"
					>
						{@render appRow(client)}
					</li>
				{/each}
			</ul>
			{#if hasMoreApps}
				<Button variant="ghost" class="mt-3 -ml-3" onclick={loadMoreApps}>
					{m.show_more_apps()}
				</Button>
			{/if}
		</section>
	{/if}

	{#if otherApps.length > 0}
		<div class="mt-10">
			<AdvancedDisclosure bind:open={showOtherApps} title={m.other_apps()} titleSize="heading">
				<div>
					<p class="text-caption text-fg-muted -mt-3 mb-3">{m.other_apps_description()}</p>
					<ul
						aria-label={m.other_apps()}
						class="grid grid-cols-1 gap-x-3 sm:grid-cols-2 lg:grid-cols-3"
					>
						{#each otherApps as client (client.id)}
							<li
								class="group/app hover:bg-fill-hover relative -mx-2 flex min-h-14 items-center gap-3 rounded-md px-2"
							>
								{@render appRow(client)}
							</li>
						{/each}
					</ul>
					{#if authorizedClients.pagination.currentPage < authorizedClients.pagination.totalPages}
						<Button variant="ghost" class="mt-3 -ml-3" onclick={loadMoreOtherApps}>
							{m.show_more_apps()}
						</Button>
					{/if}
				</div>
			</AdvancedDisclosure>
		</div>
	{/if}
{/if}
