<script lang="ts">
	import ImageBox from '$lib/components/image-box.svelte';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { ListRequestOptions } from '$lib/types/list-request.type';
	import type { OidcClient } from '$lib/types/oidc.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { mode } from 'mode-watcher';

	let {
		selectedGroupIds = $bindable()
	}: {
		selectedGroupIds: string[];
	} = $props();

	const oidcClientService = new OidcService();

	const isLightMode = $derived(mode.current === 'light');

	const columns: AdvancedTableColumn<OidcClient>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.logo(), key: 'logo', cell: LogoCell },
		{ label: m.name(), column: 'name', sortable: true },
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
		const unrestrictedClientIds = clients.data.filter((c) => !c.isGroupRestricted).map((c) => c.id);
		selectedGroupIds = [...new Set([...selectedGroupIds, ...unrestrictedClientIds])];

		return clients;
	}
</script>

{#snippet LogoCell({ item }: { item: OidcClient })}
	{#if item.hasLogo}
		<ImageBox
			class="size-12 rounded-lg"
			src={cachedOidcClientLogo.getUrl(item.id, isLightMode)}
			alt={m.name_logo({ name: item.name })}
		/>
	{:else}
		<div class="bg-muted flex size-12 items-center justify-center rounded-lg text-lg font-bold">
			{item.name.charAt(0).toUpperCase()}
		</div>
	{/if}
{/snippet}

<AdvancedTable
	id="oidc-client-selection"
	{fetchCallback}
	defaultSort={{ column: 'name', direction: 'asc' }}
	bind:selectedIds={selectedGroupIds}
	rowSelectionDisabled={(item) => !item.isGroupRestricted}
	{columns}
/>
