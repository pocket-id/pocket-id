<script lang="ts">
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import type { OidcClient, OidcClientWithAllowedGroups } from '#lib/types/oidc.type.ts';

	let {
		selectedGroupIds = $bindable()
	}: {
		selectedGroupIds: string[];
	} = $props();

	const oidcClientService = new OidcService();

	let tableRef: AdvancedTable<OidcClientWithAllowedGroups>;
	let unrestrictedClientIds = $state<string[]>([]);

	function getTableSelectedIds() {
		return [...new Set([...selectedGroupIds, ...unrestrictedClientIds])];
	}

	function setTableSelectedIds(clientIds: string[]) {
		selectedGroupIds = clientIds.filter((id) => !unrestrictedClientIds.includes(id));
	}

	export function refresh() {
		return tableRef?.refresh();
	}

	const columns: AdvancedTableColumn<OidcClientWithAllowedGroups>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.logo(), key: 'logo', cell: LogoCell },
		{ label: m.name(), column: 'name', sortable: true },
		{
			label: m.oidc_allowed_group_count(),
			column: 'allowedUserGroups',
			sortable: true,

			value: (item) => (item.isGroupRestricted ? item.allowedUserGroups.length : '-')
		},
		{
			label: m.restricted(),
			column: 'isGroupRestricted',
			sortable: true
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

	async function fetchCallback(requestOptions: ListRequestOptions) {
		const clients = await oidcClientService.listClients(requestOptions);
		unrestrictedClientIds = clients.data.filter((c) => !c.isGroupRestricted).map((c) => c.id);

		return clients;
	}
</script>

{#snippet LogoCell({ item }: { item: OidcClient })}
	<OidcClientAvatar id={item.id} name={item.name} hasLogo={item.hasLogo || item.hasDarkLogo} />
{/snippet}

<AdvancedTable
	bind:this={tableRef}
	id="oidc-client-selection"
	{fetchCallback}
	defaultSort={{ column: 'name', direction: 'asc' }}
	bind:selectedIds={getTableSelectedIds, setTableSelectedIds}
	rowSelectionDisabled={(item) => !item.isGroupRestricted}
	{columns}
/>
