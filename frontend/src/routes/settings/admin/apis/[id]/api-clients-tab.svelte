<script lang="ts">
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import ApiPermissionsModal from '#lib/components/api-permissions-modal.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as InputGroup from '#lib/components/ui/input-group/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import type { Api, ApiClient, ApiClientAccess, ApiClientGrant } from '#lib/types/api.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideBan, LucidePencil, LucidePlus, LucideSearch } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let { api, clients = $bindable() }: { api: Api; clients: ApiClientAccess[] } = $props();

	const apisService = new ApisService();

	let editing = $state<ApiClientAccess | null>(null);
	let accessOpen = $state(false);
	let pickerOpen = $state(false);
	let pickerSearch = $state('');
	let assignable = $state<ApiClient[]>();

	const pickerItems = $derived(
		assignable?.filter((c) => c.name.toLowerCase().includes(pickerSearch.trim().toLowerCase()))
	);

	export async function refresh() {
		clients = (await apisService.listClients(api.id, { pagination: { page: 1, limit: 100 } })).data;
	}

	async function openPicker() {
		pickerSearch = '';
		assignable = undefined;
		pickerOpen = true;
		try {
			assignable = (
				await apisService.listAssignableClients(api.id, {
					pagination: { page: 1, limit: 100 },
					sort: { column: 'name', direction: 'asc' }
				})
			).data;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function edit(entry: ApiClientAccess) {
		editing = entry;
		accessOpen = true;
	}

	function pick(client: ApiClient) {
		pickerOpen = false;
		edit({
			client,
			userDelegatedAccess: true,
			clientAccess: false,
			userDelegatedPermissionIds: [],
			clientPermissionIds: [],
			cimdGrantedAccess: false,
			cimdGrantedPermissionIds: []
		});
	}

	async function save(entry: ApiClientAccess, grant: ApiClientGrant) {
		await apisService.updateClientAccessForApi(api.id, entry.client.id, grant);
		await refresh();
		toast.success(m.api_access_updated());
	}

	function summary(entry: ApiClientAccess) {
		const total = api.permissions.length;
		const count = (ids: string[]) =>
			total ? m.count_of_total({ count: new Set(ids).size, total }) : m.allowed_lowercase();
		const people =
			entry.userDelegatedAccess || entry.cimdGrantedAccess
				? count([...entry.userDelegatedPermissionIds, ...entry.cimdGrantedPermissionIds])
				: m.no_access_lowercase();
		const itself = entry.client.isPublic
			? m.not_available_public_client()
			: entry.clientAccess
				? count(entry.clientPermissionIds)
				: m.no_access_lowercase();
		return m.api_client_access_summary({ people, itself });
	}

	function removeAccess(entry: ApiClientAccess) {
		openConfirmDialog({
			title: m.remove_client_access_question({ client: entry.client.name, api: api.name }),
			message: m.remove_client_access_consequence({ client: entry.client.name }),
			confirm: {
				label: m.remove_access(),
				destructive: true,
				icon: LucideBan,
				action: async () => {
					try {
						await apisService.removeClientAccessForApi(api.id, entry.client.id);
						await refresh();
						toast.success(m.api_access_updated());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<SettingsSection title={m.clients()}>
	{#snippet action()}
		<Button variant="secondary" size="sm" onclick={openPicker}>
			<LucidePlus />
			{m.add_client_ellipsis()}
		</Button>
	{/snippet}
	<ul class="border-t">
		{#each clients as entry (entry.client.id)}
			<li class="flex min-h-15 items-center gap-3 border-b py-2.5 ps-4 pe-2">
				<OidcClientAvatar
					id={entry.client.id}
					name={entry.client.name}
					hasLogo={entry.client.hasLogo || entry.client.hasDarkLogo}
					size={24}
				/>
				<span class="flex min-w-0 flex-1 flex-col gap-0.5">
					<span class="flex min-w-0 items-center gap-2">
						<a
							href="/settings/admin/oidc-clients/{encodeClientIdParam(entry.client.id)}"
							class="text-control text-fg truncate font-medium hover:underline"
						>
							{entry.client.name}
						</a>
						{#if entry.client.clientType === 'cimd'}
							<Badge variant="outline">{m.metadata_document()}</Badge>
						{/if}
					</span>
					<span class="text-caption text-fg-muted">{summary(entry)}</span>
				</span>
				<ActionsMenu
					item={entry}
					size="icon-sm"
					label={m.actions_for_name({ name: entry.client.name })}
					actions={[
						{ label: m.edit_access_ellipsis(), icon: LucidePencil, onClick: edit },
						{
							label: m.remove_access_ellipsis(),
							icon: LucideBan,
							variant: 'danger',
							// Access that only comes from the metadata document setting is changed on that tab
							disabled: !entry.userDelegatedAccess && !entry.clientAccess,
							onClick: removeAccess
						}
					]}
				/>
			</li>
		{:else}
			<li class="text-control text-fg-muted flex min-h-(--row-2line) items-center border-b px-4">
				{m.no_clients_can_request_tokens_for_this_api()}
			</li>
		{/each}
	</ul>
</SettingsSection>

<Dialog.Root bind:open={pickerOpen}>
	<Dialog.Content class="max-w-(--dialog-lg)">
		<Dialog.Header>
			<Dialog.Title>{m.add_client_to_name({ name: api.name })}</Dialog.Title>
			<Dialog.Description>{m.select_a_client_to_grant_access_to_this_api()}</Dialog.Description>
		</Dialog.Header>
		<InputGroup.Root>
			<InputGroup.Input
				bind:value={pickerSearch}
				placeholder={m.search_clients()}
				aria-label={m.search_clients()}
				type="search"
			/>
			<InputGroup.Addon><LucideSearch /></InputGroup.Addon>
		</InputGroup.Root>
		<div class="max-h-90 overflow-y-auto rounded-md border">
			{#each pickerItems ?? [] as client (client.id)}
				<button
					type="button"
					class="hover:bg-fill-row-hover focus-visible:bg-fill-row-hover flex min-h-(--row-2line) w-full items-center gap-3 border-b px-4 py-2 text-start last:border-b-0 focus-visible:outline-2 focus-visible:-outline-offset-2"
					onclick={() => pick(client)}
				>
					<OidcClientAvatar
						id={client.id}
						name={client.name}
						hasLogo={client.hasLogo || client.hasDarkLogo}
						size={24}
					/>
					<span class="text-control text-fg min-w-0 flex-1 truncate font-medium">{client.name}</span
					>
					{#if client.clientType === 'cimd'}
						<Badge variant="outline">{m.metadata_document()}</Badge>
					{/if}
				</button>
			{:else}
				<p class="text-control text-fg-muted flex min-h-(--row-2line) items-center px-4">
					{assignable ? m.every_client_already_has_access({ name: api.name }) : m.loading()}
				</p>
			{/each}
		</div>
	</Dialog.Content>
</Dialog.Root>

{#if editing}
	<ApiPermissionsModal
		bind:open={accessOpen}
		{api}
		grant={editing}
		implicitUserAccess={editing.cimdGrantedAccess}
		implicitUserIds={editing.cimdGrantedPermissionIds}
		showClientAccess={!editing.client.isPublic}
		title={m.access_for_name({ name: editing.client.name })}
		onSave={(grant) => save(editing!, grant)}
	/>
{/if}
