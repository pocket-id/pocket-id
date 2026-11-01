<script lang="ts">
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { Api } from '#lib/types/api.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';

	let {
		open = $bindable(),
		clientId,
		onSelect
	}: {
		open: boolean;
		clientId: string;
		onSelect: (api: Api) => void;
	} = $props();

	const apisService = new ApisService();

	const columns: AdvancedTableColumn<Api>[] = [
		{ label: m.api(), column: 'name', sortable: true, cell: ApiCell },
		{ label: m.api_permissions(), key: 'permissions', value: (item) => item.permissions.length }
	];

	function fetchCallback(options: ListRequestOptions) {
		return apisService.listAssignableApis(clientId, options);
	}
</script>

{#snippet ApiCell({ item }: { item: Api })}
	<div class="flex max-w-[min(20rem,55vw)] min-w-0 items-center gap-3">
		<OidcClientAvatar id={item.id} name={item.name} hasLogo={false} size={24} />
		<div class="flex min-w-0 flex-col">
			<span class="text-control text-fg truncate font-medium">{item.name}</span>
			<span class="text-mono-s text-fg-muted truncate font-mono" title={item.resource}
				>{item.resource}</span
			>
		</div>
	</div>
{/snippet}

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-(--dialog-lg)">
		<Dialog.Header>
			<Dialog.Title>{m.add_api()}</Dialog.Title>
			<Dialog.Description>{m.select_an_api_this_client_should_access()}</Dialog.Description>
		</Dialog.Header>

		<AdvancedTable
			id="client-api-selection"
			{fetchCallback}
			onRowClick={(item) => onSelect(item)}
			defaultSort={{ column: 'name', direction: 'asc' }}
			{columns}
		/>
	</Dialog.Content>
</Dialog.Root>
