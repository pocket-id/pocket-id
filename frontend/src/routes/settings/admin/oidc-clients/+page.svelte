<script lang="ts">
	import { goto } from '$app/navigation';
	import DocsLink from '#lib/components/docs-link.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import EmptyList from '#lib/components/empty-list.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OIDCService from '#lib/services/oidc-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import type { OidcClientWithAllowedGroups } from '#lib/types/oidc.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { LucideArrowRight, LucideArrowUpRight, LucidePlus } from '@lucide/svelte';
	import OidcClientCreateSheet from './oidc-client-create-sheet.svelte';
	import {
		copyClientIdAction,
		deleteClientAction,
		refreshClientAction,
		typeLabel
	} from './oidc-client-actions';

	let { data } = $props();

	const oidcService = new OIDCService();
	const connectDocsUrl = 'https://pocket-id.org/docs/client-examples';
	let tableRef = $state<AdvancedTable<OidcClientWithAllowedGroups>>();
	let createOpen = $state(false);
	let empty = $state(false);

	async function fetchClients(options: ListRequestOptions) {
		const res = await oidcService.listClients(options);
		const filtered = Object.values(options.filters ?? {}).some((values) => values.length);
		empty = res.pagination.totalItems === 0 && !options.search && !filtered;
		return res;
	}

	const refresh = () => tableRef?.refresh();
	const detailHref = (client: OidcClientWithAllowedGroups) =>
		`/settings/admin/oidc-clients/${encodeClientIdParam(client.id)}`;

	const onOffValues = [
		{ label: m.on(), value: true },
		{ label: m.off(), value: false }
	];
	const onOff = (value: boolean) => (value ? m.on() : m.off());

	function accessLabel(client: OidcClientWithAllowedGroups) {
		const groups = client.allowedUserGroups.length;
		if (!client.isGroupRestricted) return m.all_users();
		if (groups === 0) return m.no_groups();
		return groups === 1 ? m.one_group() : m.group_count({ count: groups });
	}

	const columns: AdvancedTableColumn<OidcClientWithAllowedGroups>[] = $derived([
		{ label: m.client(), column: 'name', sortable: true, cell: ClientCell },
		{
			label: m.who_can_sign_in(),
			column: 'isGroupRestricted',
			cell: AccessCell,
			filterableValues: [
				{ label: m.all_users(), value: false },
				{ label: m.selected_groups(), value: true }
			]
		},
		{ label: m.type(), key: 'type', value: typeLabel, hideBelow: 760 },
		...(data.features?.cimd.enabled
			? ([
					{
						label: m.registration(),
						column: 'clientType',
						hidden: true,
						value: (client) =>
							client.clientType === 'cimd' ? m.metadata_document() : m.registered_by_an_admin(),
						filterableValues: [
							{ label: m.registered_by_an_admin(), value: 'standard' },
							{ label: m.metadata_document(), value: 'cimd' }
						]
					}
				] satisfies AdvancedTableColumn<OidcClientWithAllowedGroups>[])
			: []),
		{
			label: m.pkce(),
			column: 'pkceEnabled',
			hidden: true,
			value: (client) => onOff(client.pkceEnabled),
			filterableValues: onOffValues
		},
		{
			label: m.reauthentication(),
			column: 'requiresReauthentication',
			hidden: true,
			value: (client) => onOff(client.requiresReauthentication),
			filterableValues: onOffValues
		},
		{
			label: m.par(),
			column: 'requiresPushedAuthorizationRequests',
			hidden: true,
			value: (client) => onOff(client.requiresPushedAuthorizationRequests),
			filterableValues: onOffValues
		},
		{ label: m.launch_url(), key: 'launchURL', hidden: true, cell: LaunchUrlCell },
		{
			label: m.callback_urls(),
			key: 'callbackURLs',
			hidden: true,
			value: (client) => client.callbackURLs.length
		}
	]);

	const rowActions: CreateAdvancedTableActions<OidcClientWithAllowedGroups> = (client) => [
		{
			label: m.open(),
			primary: true,
			icon: LucideArrowRight,
			onClick: (client) => goto(detailHref(client))
		},
		copyClientIdAction(),
		refreshClientAction(client, refresh),
		deleteClientAction(refresh)
	];
</script>

<svelte:head>
	<title>{m.oidc_clients()}</title>
</svelte:head>

{#snippet ClientCell({ item }: { item: OidcClientWithAllowedGroups })}
	<div class="flex max-w-(--table-id-max) min-w-0 items-center gap-3">
		<OidcClientAvatar id={item.id} name={item.name} hasLogo={item.hasLogo || item.hasDarkLogo} />
		<div class="flex min-w-0 flex-col">
			<span class="flex min-w-0 items-center gap-2">
				<a
					href={detailHref(item)}
					class="text-control text-fg truncate font-medium"
					onclick={(e) => e.stopPropagation()}>{item.name}</a
				>
				{#if item.clientType === 'cimd'}
					<Badge variant="outline">{m.metadata_document()}</Badge>
				{/if}
			</span>
			<span class="text-caption text-fg-muted truncate font-mono">{item.id}</span>
			<span class="text-caption text-fg-muted truncate sm:hidden">
				{#if item.isGroupRestricted && item.allowedUserGroups.length === 0}
					<Badge variant="warning" class="text-caption text-fg-muted before:size-1.5">
						{accessLabel(item)}
					</Badge>
				{:else}
					{accessLabel(item)}
				{/if}
				· {typeLabel(item)}
			</span>
		</div>
	</div>
{/snippet}

{#snippet AccessCell({ item }: { item: OidcClientWithAllowedGroups })}
	{#if !item.isGroupRestricted}
		{accessLabel(item)}
	{:else if item.allowedUserGroups.length === 0}
		<Tooltip.Provider>
			<Tooltip.Root>
				<Tooltip.Trigger class="cursor-default">
					<Badge variant="warning">{accessLabel(item)}</Badge>
				</Tooltip.Trigger>
				<Tooltip.Content>{m.no_one_can_sign_in_until_you_allow_a_group()}</Tooltip.Content>
			</Tooltip.Root>
		</Tooltip.Provider>
	{:else}
		<Tooltip.Provider>
			<Tooltip.Root>
				<Tooltip.Trigger class="cursor-default underline decoration-dotted underline-offset-4">
					{accessLabel(item)}
				</Tooltip.Trigger>
				<Tooltip.Content class="max-w-xs">
					{item.allowedUserGroups.map((group) => group.friendlyName).join(', ')}
				</Tooltip.Content>
			</Tooltip.Root>
		</Tooltip.Provider>
	{/if}
{/snippet}

{#snippet LaunchUrlCell({ item }: { item: OidcClientWithAllowedGroups })}
	{#if item.launchURL}
		<span class="text-caption block max-w-60 truncate font-mono" title={item.launchURL}>
			{item.launchURL.replace(/^https?:\/\//, '')}
		</span>
	{:else}
		<span class="text-fg-muted">{m.none()}</span>
	{/if}
{/snippet}

{#snippet headerDescription()}
	{m.oidc_clients_page_description({ appName: $appConfigStore.appName })}
	<DocsLink href={connectDocsUrl} />
{/snippet}

<div class="flex flex-col">
	<PageHeader title={m.oidc_clients()} description={empty ? undefined : headerDescription}>
		{#snippet actions()}
			{#if !empty}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.add_oidc_client()}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if empty}
		<EmptyList columns={[m.client(), m.who_can_sign_in()]} title={m.add_your_first_oidc_client()}>
			{#snippet description()}
				{m.oidc_clients_page_description({ appName: $appConfigStore.appName })}
			{/snippet}
			{#snippet action()}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.add_oidc_client()}
				</Button>
				<Button variant="link" class="gap-1" href={connectDocsUrl} target="_blank" rel="noreferrer">
					{m.docs()}
					<LucideArrowUpRight />
				</Button>
			{/snippet}
		</EmptyList>
	{:else}
		<div class="mt-10">
			<AdvancedTable
				id="oidc-client-list"
				bind:this={tableRef}
				fetchCallback={fetchClients}
				defaultSort={{ column: 'name', direction: 'asc' }}
				{columns}
				actions={rowActions}
			/>
		</div>
	{/if}
</div>

<OidcClientCreateSheet bind:open={createOpen} />
