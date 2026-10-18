<script lang="ts">
	import { goto } from '$app/navigation';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { ScrollArea } from '#lib/components/ui/scroll-area/index.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OIDCService from '#lib/services/oidc-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { OidcClientWithAllowedGroups } from '#lib/types/oidc.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { LucidePencil } from '@lucide/svelte';
	import { deleteClientAction, refreshClientAction } from './oidc-client-actions';

	const oidcService = new OIDCService();
	let tableRef: AdvancedTable<OidcClientWithAllowedGroups>;

	export function refresh() {
		return tableRef?.refresh();
	}

	const booleanFilterValues = [
		{ label: m.yes(), value: true },
		{ label: m.no(), value: false }
	];

	const clientTypeFilterValues = [
		{ label: m.client_type_standard(), value: 'standard' },
		{ label: m.client_type_metadata_document(), value: 'cimd' }
	];

	const columns: AdvancedTableColumn<OidcClientWithAllowedGroups>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.logo(), key: 'logo', cell: LogoCell },
		{ label: m.name(), column: 'name', sortable: true },
		{
			label: m.oidc_allowed_group_count(),
			column: 'allowedUserGroups',
			sortable: true,
			cell: AllowedGroupCountCell
		},
		{
			label: m.restricted(),
			column: 'isGroupRestricted',
			sortable: true,
			filterableValues: booleanFilterValues
		},
		{
			label: m.client_type(),
			column: 'clientType',
			sortable: true,
			filterableValues: clientTypeFilterValues,
			value: (item) =>
				item.clientType === 'cimd' ? m.client_type_metadata_document() : m.client_type_standard()
		},
		{
			label: m.pkce(),
			column: 'pkceEnabled',
			sortable: true,
			hidden: true,
			filterableValues: booleanFilterValues
		},
		{
			label: m.reauthentication(),
			column: 'requiresReauthentication',
			sortable: true,
			filterableValues: booleanFilterValues
		},
		{
			label: m.par(),
			column: 'requiresPushedAuthorizationRequests',
			sortable: true,
			hidden: true,
			filterableValues: booleanFilterValues
		},
		{
			label: m.client_launch_url(),
			column: 'launchURL',
			hidden: true
		},
		{
			label: m.public_client(),
			column: 'isPublic',
			sortable: true,
			hidden: true
		}
	];

	const actions: CreateAdvancedTableActions<OidcClientWithAllowedGroups> = (client) => [
		{
			label: m.edit(),
			primary: true,
			icon: LucidePencil,
			onClick: (client) => goto(`/settings/admin/oidc-clients/${encodeClientIdParam(client.id)}`)
		},
		refreshClientAction(client, refresh),
		deleteClientAction(refresh)
	];
</script>

{#snippet AllowedGroupCountCell({ item }: { item: OidcClientWithAllowedGroups })}
	{#if !item.isGroupRestricted}
		-
	{:else if item.allowedUserGroups.length === 0}
		{item.allowedUserGroups.length}
	{:else}
		<Tooltip.Provider>
			<Tooltip.Root>
				<Tooltip.Trigger class="cursor-default underline decoration-dotted underline-offset-4">
					{item.allowedUserGroups.length}
				</Tooltip.Trigger>
				<Tooltip.Content side="right" class="flex-col items-start">
					<ScrollArea
						class="[&>[data-slot=scroll-area-viewport]]:max-h-48"
						scrollbarYClasses="[&>[data-slot=scroll-area-thumb]]:bg-background/40"
					>
						<div class="flex flex-col gap-0.5 pr-3">
							{#each item.allowedUserGroups as group (group.id)}
								<span>{group.friendlyName}</span>
							{/each}
						</div>
					</ScrollArea>
				</Tooltip.Content>
			</Tooltip.Root>
		</Tooltip.Provider>
	{/if}
{/snippet}

{#snippet LogoCell({ item }: { item: OidcClientWithAllowedGroups })}
	<OidcClientAvatar id={item.id} name={item.name} hasLogo={item.hasLogo || item.hasDarkLogo} />
{/snippet}

<AdvancedTable
	id="oidc-client-list"
	bind:this={tableRef}
	fetchCallback={oidcService.listClients}
	defaultSort={{ column: 'name', direction: 'asc' }}
	{columns}
	{actions}
/>
