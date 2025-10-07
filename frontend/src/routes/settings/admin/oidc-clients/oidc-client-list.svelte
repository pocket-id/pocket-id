<script lang="ts">
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import ImageBox from '$lib/components/image-box.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { OidcClient, OidcClientWithAllowedUserGroupsCount } from '$lib/types/oidc.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	const oidcService = new OIDCService();
	let tableRef : AdvancedTable<OidcClientWithAllowedUserGroupsCount>;

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
						await tableRef.refresh();
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
	id="oidc-client-list"
	bind:this={tableRef}
	fetchCallback={oidcService.listClients}
	defaultSort={{ column: 'name', direction: 'asc' }}
	columns={[
		{ label: m.logo() },
		{ label: m.name(), sortColumn: 'name' },
		{ label: m.oidc_allowed_group_count(), sortColumn: 'allowedUserGroupsCount' },
		{ label: m.actions(), hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell class="w-8 font-medium">
			{#if item.hasLogo}
				<ImageBox
					class="min-h-8 min-w-8 object-contain"
					src={cachedOidcClientLogo.getUrl(item.id)}
					alt={m.name_logo({ name: item.name })}
				/>
			{/if}
		</Table.Cell>
		<Table.Cell class="font-medium">{item.name}</Table.Cell>
		<Table.Cell class="font-medium"
			>{item.allowedUserGroupsCount > 0
				? item.allowedUserGroupsCount
				: m.unrestricted()}</Table.Cell
		>
		<Table.Cell class="flex justify-end gap-1">
			<Button
				href="/settings/admin/oidc-clients/{item.id}"
				size="sm"
				variant="outline"
				aria-label={m.edit()}><LucidePencil class="size-3 " /></Button
			>
			<Button onclick={() => deleteClient(item)} size="sm" variant="outline" aria-label={m.delete()}
				><LucideTrash class="size-3 text-red-500" /></Button
			>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
