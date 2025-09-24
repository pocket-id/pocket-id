<script lang="ts">
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import ImageBox from '$lib/components/image-box.svelte';
	import { goto } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type { OidcClientWithAllowedUserGroupsCount } from '$lib/types/oidc.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import type { ColumnSpec } from '$lib/components/pocket-id-table/pocket-id-table.types.svelte';

	let {
		clients = $bindable(),
		requestOptions
	}: {
		clients: Paginated<OidcClientWithAllowedUserGroupsCount>;
		requestOptions: SearchPaginationSortRequest;
	} = $props();

	const oidcService = new OIDCService();

	async function deleteClient(client: OidcClientWithAllowedUserGroupsCount) {
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

	const columns: ColumnSpec<OidcClientWithAllowedUserGroupsCount>[] = [
		{
			id: 'logo',
			accessorFn: (row) => row.hasLogo ?? false,
			title: m.logo(),
			cell: LogoCell as any
		},
		{ accessorKey: 'name', title: m.name(), sortable: true },
		{
			accessorKey: 'allowedUserGroupsCount',
			title: m.oidc_allowed_group_count(),
			sortable: true,
			cell: AllowedCell as any
		}
	] satisfies ColumnSpec<OidcClientWithAllowedUserGroupsCount>[];
</script>

{#snippet LogoCell({ item }: { item: OidcClientWithAllowedUserGroupsCount })}
	{#if item.hasLogo}
		<ImageBox
			class="min-h-8 min-w-8 object-contain"
			src={cachedOidcClientLogo.getUrl(item.id)}
			alt={m.name_logo({ name: item.name })}
		/>
	{/if}
{/snippet}

{#snippet AllowedCell({ item }: { item: OidcClientWithAllowedUserGroupsCount })}
	{item.allowedUserGroupsCount > 0 ? item.allowedUserGroupsCount : m.unrestricted()}
{/snippet}

{#snippet RowActions({ item }: { item: OidcClientWithAllowedUserGroupsCount })}
	<div class="flex justify-end gap-1">
		<Button
			href={`/settings/admin/oidc-clients/${item.id}`}
			size="sm"
			variant="outline"
			aria-label={m.edit()}
		>
			<LucidePencil class="size-3" />
		</Button>
		<Button onclick={() => deleteClient(item)} size="sm" variant="outline" aria-label={m.delete()}>
			<LucideTrash class="size-3 text-red-500" />
		</Button>
	</div>
{/snippet}

<PocketIdTable
	items={clients}
	bind:requestOptions
	onRefresh={async (o) => (clients = await oidcService.listClients(o))}
	{columns}
	persistKey="pocket-id-oidc-clients"
	rowActions={RowActions}
/>
