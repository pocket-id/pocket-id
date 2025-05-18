<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { OidcClient } from '$lib/types/oidc.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let {
		clients = $bindable(),
		requestOptions
	}: {
		clients: Paginated<OidcClient>;
		requestOptions: SearchPaginationSortRequest;
	} = $props();

	const oidcService = new OIDCService();

	async function deleteClient(client: OidcClient) {
		openConfirmDialog({
			title: m.delete_name({ name: client.name }),
			message: m.are_you_sure_you_want_to_delete_this_oidc_client(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await oidcService.removeClient(client.id);
						clients = await oidcService.listClients(requestOptions!);
						toast.success(m.oidc_client_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<AdvancedTable
	items={clients}
	{requestOptions}
	onRefresh={async (o) => (clients = await oidcService.listClients(o))}
	columns={[
		{ label: m.logo() },
		{ label: m.name(), sortColumn: 'name' },
		{ label: m.actions(), hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell class="w-8 font-medium">
			{#if item.hasLogo}
				<div class="bg-secondary rounded-2xl p-3">
					<div class="h-8 w-8">
						<img
							class="m-auto max-h-full max-w-full object-contain"
							src="/api/oidc/clients/{item.id}/logo"
							alt={m.name_logo({ name: item.name })}
						/>
					</div>
				</div>
			{/if}
		</Table.Cell>
		<Table.Cell class="font-medium">{item.name}</Table.Cell>
		<Table.Cell class="flex justify-end gap-1">
			<Button
				href="/settings/admin/oidc-clients/{item.id}"
				size="sm"
				variant="outline"
				aria-label={m.edit()}><LucidePencil class="h-3 w-3 " /></Button
			>
			<Button
				on:click={() => deleteClient(item)}
				size="sm"
				variant="outline"
				aria-label={m.delete()}><LucideTrash class="h-3 w-3 text-red-500" /></Button
			>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
