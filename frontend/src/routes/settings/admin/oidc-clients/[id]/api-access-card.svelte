<script lang="ts">
	import { goto } from '$app/navigation';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type { Api } from '$lib/types/api.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import ApiPermissionsModal from './api-permissions-modal.svelte';

	let { clientId, isPublicClient }: { clientId: string; isPublicClient: boolean } = $props();

	const apisService = new ApisService();

	let apis = $state<Api[]>([]);
	let userSelected = $state<Set<string>>(new Set());
	let clientSelected = $state<Set<string>>(new Set());
	let loading = $state(true);

	let editingApi = $state<Api | null>(null);
	let modalOpen = $state(false);

	onMount(async () => {
		try {
			const [list, access] = await Promise.all([
				apisService.listAll(),
				apisService.getClientAccess(clientId)
			]);
			apis = list;
			userSelected = new Set(access.userDelegatedPermissionIds);
			clientSelected = new Set(access.clientPermissionIds);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			loading = false;
		}
	});

	function grantedCount(api: Api, selected: Set<string>) {
		return api.permissions.filter((p) => selected.has(p.id)).length;
	}

	function openEdit(api: Api) {
		editingApi = api;
		modalOpen = true;
	}

	function allowedIdsFor(api: Api, selected: Set<string>) {
		return api.permissions.filter((p) => selected.has(p.id)).map((p) => p.id);
	}

	async function saveApi(api: Api, userIds: string[], clientIds: string[]) {
		// Grants of other APIs stay untouched, and for public clients the (never editable) client grants are sent back unchanged
		const otherUser = [...userSelected].filter((id) => !api.permissions.some((p) => p.id === id));
		const otherClient = [...clientSelected].filter(
			(id) => !api.permissions.some((p) => p.id === id)
		);
		const res = await apisService.updateClientAccess(clientId, {
			userDelegatedPermissionIds: [...otherUser, ...userIds],
			clientPermissionIds: isPublicClient ? [...clientSelected] : [...otherClient, ...clientIds]
		});
		userSelected = new Set(res.userDelegatedPermissionIds);
		clientSelected = new Set(res.clientPermissionIds);
		toast.success(m.api_access_updated_successfully());
	}
</script>

{#if loading}
	<div class="flex justify-center py-6">
		<Spinner class="size-6" />
	</div>
{:else if apis.length === 0}
	<div class="flex flex-col items-center justify-center gap-2 py-6">
		<p class="text-muted-foreground text-sm">{m.no_apis_defined_yet()}</p>
		<Button variant="outline" size="sm" onclick={() => goto('/settings/admin/apis')}>
			{m.create_api()}
		</Button>
	</div>
{:else}
	<Table.Root>
		<Table.Header>
			<Table.Row>
				<Table.Head>{m.api_name()}</Table.Head>
				<Table.Head>{m.user_delegated_access()}</Table.Head>
				{#if !isPublicClient}
					<Table.Head>{m.client_access()}</Table.Head>
				{/if}
				<Table.Head class="w-20"></Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each apis as api}
				<Table.Row>
					<Table.Cell>
						<div class="flex flex-col gap-1">
							<span class="font-medium">{api.name}</span>
							<div>
								<CopyToClipboard value={api.resource}>
									<span class="text-muted-foreground font-mono text-xs break-all"
										>{api.resource}</span
									>
								</CopyToClipboard>
							</div>
						</div>
					</Table.Cell>
					<Table.Cell class="text-muted-foreground text-sm">
						{m.permissions_granted_count({
							granted: String(grantedCount(api, userSelected)),
							total: String(api.permissions.length)
						})}
					</Table.Cell>
					{#if !isPublicClient}
						<Table.Cell class="text-muted-foreground text-sm">
							{m.permissions_granted_count({
								granted: String(grantedCount(api, clientSelected)),
								total: String(api.permissions.length)
							})}
						</Table.Cell>
					{/if}
					<Table.Cell class="text-right">
						<Button
							variant="outline"
							size="sm"
							disabled={api.permissions.length === 0}
							onclick={() => openEdit(api)}>{m.edit()}</Button
						>
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
{/if}

{#if editingApi}
	<ApiPermissionsModal
		bind:open={modalOpen}
		api={editingApi}
		userAllowedIds={allowedIdsFor(editingApi, userSelected)}
		clientAllowedIds={allowedIdsFor(editingApi, clientSelected)}
		showClientAccess={!isPublicClient}
		onSave={(userIds, clientIds) => saveApi(editingApi!, userIds, clientIds)}
	/>
{/if}
