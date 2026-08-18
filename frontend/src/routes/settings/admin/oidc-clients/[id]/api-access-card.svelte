<script lang="ts">
	import ApiAccessCell from '$lib/components/api-access-cell.svelte';
	import ApiPermissionsModal from '$lib/components/api-permissions-modal.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Spinner } from '$lib/components/ui/spinner';
	import * as Table from '$lib/components/ui/table';
	import Empty from '$lib/icons/empty.svelte';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type { Api, ApiClientGrant, ClientApiGrant } from '$lib/types/api.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import ApiSelectionModal from './api-selection-modal.svelte';

	let { clientId, isPublicClient }: { clientId: string; isPublicClient: boolean } = $props();

	const apisService = new ApisService();

	let grants = $state<ClientApiGrant[]>([]);
	let loading = $state(true);

	let editing = $state<ClientApiGrant | null>(null);
	let modalOpen = $state(false);
	let pickerOpen = $state(false);

	onMount(load);

	async function load() {
		try {
			grants = await apisService.listClientApis(clientId);
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			loading = false;
		}
	}

	function openEdit(grant: ClientApiGrant) {
		editing = grant;
		modalOpen = true;
	}

	function addApi(api: Api) {
		editing = {
			api,
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

	async function save(entry: ClientApiGrant, grant: ApiClientGrant) {
		await apisService.updateClientAccessForApi(entry.api.id, clientId, {
			...grant,
			clientAccess: isPublicClient ? false : grant.clientAccess,
			clientPermissionIds: isPublicClient ? [] : grant.clientPermissionIds
		});

		await load();
		toast.success(m.api_access_updated_successfully());
	}

	function removeApi(entry: ClientApiGrant) {
		openConfirmDialog({
			title: m.revoke_access_to_name({ name: entry.api.name }),
			message: m.are_you_sure_you_want_to_revoke_the_access_of_this_client_to_the_api(),
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					try {
						await apisService.removeClientAccessForApi(entry.api.id, clientId);
						await load();
						toast.success(m.api_access_updated_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	function userGrantedCount(entry: ClientApiGrant) {
		const granted = new Set([
			...entry.userDelegatedPermissionIds,
			...entry.cimdGrantedPermissionIds
		]);
		return granted.size;
	}
</script>

<Card.Root>
	<Card.Header>
		<div class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
			<div>
				<Card.Title>{m.api_access()}</Card.Title>
				<Card.Description>{m.api_access_description()}</Card.Description>
			</div>
			<Button onclick={() => (pickerOpen = true)}>{m.add_api()}</Button>
		</div>
	</Card.Header>
	<Card.Content>
		{#if loading}
			<div class="flex justify-center py-6">
				<Spinner class="size-6" />
			</div>
		{:else if grants.length === 0}
			<div class="my-5 flex flex-col items-center">
				<Empty class="text-muted-foreground h-20" />
				<p class="text-muted-foreground mt-3 text-sm">{m.this_client_has_no_api_access_yet()}</p>
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
					{#each grants as entry (entry.api.id)}
						<Table.Row>
							<Table.Cell>
								<div class="flex flex-col gap-1">
									<span class="font-medium">{entry.api.name}</span>
									<div>
										<CopyToClipboard value={entry.api.resource}>
											<span class="text-muted-foreground font-mono text-xs break-all"
												>{entry.api.resource}</span
											>
										</CopyToClipboard>
									</div>
									{#if entry.cimdGrantedAccess}
										<span class="text-muted-foreground text-xs"
											>{m.granted_to_all_cimd_clients()}</span
										>
									{/if}
								</div>
							</Table.Cell>
							<Table.Cell>
								<ApiAccessCell
									hasAccess={entry.userDelegatedAccess || entry.cimdGrantedAccess}
									granted={userGrantedCount(entry)}
									total={entry.api.permissions.length}
								/>
							</Table.Cell>
							{#if !isPublicClient}
								<Table.Cell>
									<ApiAccessCell
										hasAccess={entry.clientAccess}
										granted={entry.clientPermissionIds.length}
										total={entry.api.permissions.length}
									/>
								</Table.Cell>
							{/if}
							<Table.Cell class="text-right">
								<div class="flex justify-end gap-1">
									<Button
										variant="ghost"
										size="sm"
										aria-label={m.edit()}
										onclick={() => openEdit(entry)}
									>
										<LucidePencil class="size-4" />
									</Button>
									<Button
										variant="ghost"
										size="sm"
										aria-label={m.revoke()}
										disabled={!entry.userDelegatedAccess && !entry.clientAccess}
										onclick={() => removeApi(entry)}
									>
										<LucideTrash class="size-4" />
									</Button>
								</div>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</Card.Content>
</Card.Root>

<ApiSelectionModal bind:open={pickerOpen} {clientId} onSelect={addApi} />

{#if editing}
	<ApiPermissionsModal
		bind:open={modalOpen}
		api={editing.api}
		grant={editing}
		implicitUserAccess={editing.cimdGrantedAccess}
		implicitUserIds={editing.cimdGrantedPermissionIds}
		showClientAccess={!isPublicClient}
		onSave={(grant) => save(editing!, grant)}
	/>
{/if}
