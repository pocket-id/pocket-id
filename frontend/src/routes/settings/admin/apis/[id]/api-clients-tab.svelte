<script lang="ts">
	import ApiAccessCell from '$lib/components/api-access-cell.svelte';
	import ApiPermissionsModal from '$lib/components/api-permissions-modal.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import OidcClientAvatar from '$lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { Api, ApiClient, ApiClientAccess, ApiClientGrant } from '$lib/types/api.type';
	import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
	import { encodeClientIdParam } from '$lib/utils/client-id-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import ClientSelectionModal from './client-selection-modal.svelte';

	let { api }: { api: Api } = $props();

	const apisService = new ApisService();

	type ClientRow = ApiClientAccess & { id: string };

	let tableRef: AdvancedTable<ClientRow>;
	let editing = $state<ApiClientAccess | null>(null);
	let modalOpen = $state(false);
	let pickerOpen = $state(false);

	const columns: AdvancedTableColumn<ClientRow>[] = [
		{ label: m.client(), key: 'client', cell: ClientCell },
		{ label: m.user_delegated_access(), key: 'user-access', cell: UserAccessCell },
		{ label: m.client_access(), key: 'client-access', cell: ClientAccessCell },
		{ label: '', key: 'actions', cell: ActionsCell }
	];

	async function fetchCallback(options: ListRequestOptions): Promise<Paginated<ClientRow>> {
		const res = await apisService.listClients(api.id, options);
		return { ...res, data: res.data.map((entry) => ({ ...entry, id: entry.client.id })) };
	}

	export async function refresh() {
		await tableRef?.refresh();
	}

	export function openPicker() {
		pickerOpen = true;
	}

	function openEdit(entry: ApiClientAccess) {
		editing = entry;
		modalOpen = true;
	}

	function addClient(client: ApiClient) {
		editing = {
			client,
			userDelegatedAccess: true,
			clientAccess: false,
			userDelegatedPermissionIds: [],
			clientPermissionIds: [],
			cimdGrantedAccess: false,
			cimdGrantedPermissionIds: []
		};

		pickerOpen = false;
		modalOpen = true;
	}

	async function save(entry: ApiClientAccess, grant: ApiClientGrant) {
		await apisService.updateClientAccessForApi(api.id, entry.client.id, {
			...grant,
			clientAccess: entry.client.isPublic ? false : grant.clientAccess,
			clientPermissionIds: entry.client.isPublic ? [] : grant.clientPermissionIds
		});

		await tableRef?.refresh();
		toast.success(m.api_access_updated_successfully());
	}

	function userGrantedCount(entry: ApiClientAccess) {
		return new Set([...entry.userDelegatedPermissionIds, ...entry.cimdGrantedPermissionIds]).size;
	}

	function removeClient(entry: ApiClientAccess) {
		openConfirmDialog({
			title: m.revoke_access_for_name({ name: entry.client.name }),
			message: m.are_you_sure_you_want_to_revoke_the_api_access_of_this_client(),
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					try {
						await apisService.removeClientAccessForApi(api.id, entry.client.id);
						await tableRef?.refresh();
						toast.success(m.api_access_updated_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

{#snippet ClientCell({ item }: { item: ClientRow })}
	<div class="flex items-center gap-3">
		<OidcClientAvatar id={item.client.id} name={item.client.name} hasLogo={item.client.hasLogo} />
		<div class="flex flex-col gap-0.5">
			<a
				class="font-medium hover:underline"
				href={`/settings/admin/oidc-clients/${encodeClientIdParam(item.client.id)}`}
			>
				{item.client.name}
			</a>
			{#if item.client.clientType === 'cimd'}
				<span class="text-muted-foreground text-xs">{m.client_type_metadata_document()}</span>
			{/if}
		</div>
	</div>
{/snippet}

{#snippet UserAccessCell({ item }: { item: ClientRow })}
	<ApiAccessCell
		hasAccess={item.userDelegatedAccess || item.cimdGrantedAccess}
		granted={userGrantedCount(item)}
		total={api.permissions.length}
	/>
{/snippet}

{#snippet ClientAccessCell({ item }: { item: ClientRow })}
	{#if item.client.isPublic}
		<span class="text-muted-foreground text-sm">-</span>
	{:else}
		<ApiAccessCell
			hasAccess={item.clientAccess}
			granted={item.clientPermissionIds.length}
			total={api.permissions.length}
		/>
	{/if}
{/snippet}

{#snippet ActionsCell({ item }: { item: ClientRow })}
	<div class="flex justify-end gap-1">
		<Button variant="ghost" size="sm" aria-label={m.edit()} onclick={() => openEdit(item)}>
			<LucidePencil class="size-4" />
		</Button>
		<Button
			variant="ghost"
			size="sm"
			aria-label={m.revoke()}
			disabled={!item.userDelegatedAccess && !item.clientAccess}
			onclick={() => removeClient(item)}
		>
			<LucideTrash class="size-4" />
		</Button>
	</div>
{/snippet}

<AdvancedTable id={`api-clients-${api.id}`} bind:this={tableRef} {columns} {fetchCallback} />

<ClientSelectionModal bind:open={pickerOpen} apiId={api.id} onSelect={addClient} />

{#if editing}
	<ApiPermissionsModal
		bind:open={modalOpen}
		{api}
		grant={editing}
		implicitUserAccess={editing.cimdGrantedAccess}
		implicitUserIds={editing.cimdGrantedPermissionIds}
		showClientAccess={!editing.client.isPublic}
		title={editing.client.name}
		onSave={(grant) => save(editing!, grant)}
	/>
{/if}
