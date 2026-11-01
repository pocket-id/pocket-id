<script lang="ts">
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import ApiPermissionsModal from '#lib/components/api-permissions-modal.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import type { Api, ApiClientGrant, ClientApiGrant } from '#lib/types/api.type.ts';
	import type { OidcClient } from '#lib/types/oidc.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideBan, LucidePencil, LucidePlus } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import ApiSelectionModal from './api-selection-modal.svelte';

	let {
		client,
		grants = $bindable()
	}: {
		client: OidcClient;
		grants: ClientApiGrant[];
	} = $props();

	const apisService = new ApisService();

	let editing = $state<ClientApiGrant | null>(null);
	let modalOpen = $state(false);
	let pickerOpen = $state(false);

	async function load() {
		grants = await apisService.listClientApis(client.id);
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
		await apisService.updateClientAccessForApi(entry.api.id, client.id, {
			...grant,
			clientAccess: client.isPublic ? false : grant.clientAccess,
			clientPermissionIds: client.isPublic ? [] : grant.clientPermissionIds
		});

		await load();
		toast.success(m.api_access_updated());
	}

	function removeApi(entry: ClientApiGrant) {
		openConfirmDialog({
			title: m.remove_client_access_question({ client: client.name, api: entry.api.name }),
			message: m.remove_client_access_consequence({ client: client.name }),
			confirm: {
				label: m.remove_access(),
				destructive: true,
				icon: LucideBan,
				action: async () => {
					try {
						await apisService.removeClientAccessForApi(entry.api.id, client.id);
						await load();
						toast.success(m.api_access_updated());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	function accessSummary(hasAccess: boolean, granted: number, total: number) {
		if (!hasAccess) return m.no_access_lowercase();
		return total ? m.count_of_total({ count: granted, total }) : m.allowed_lowercase();
	}

	function summary(entry: ClientApiGrant) {
		const total = entry.api.permissions.length;
		const delegated = new Set([
			...entry.userDelegatedPermissionIds,
			...entry.cimdGrantedPermissionIds
		]).size;
		const parts = [
			m.on_behalf_of_people_summary({
				access: accessSummary(
					entry.userDelegatedAccess || entry.cimdGrantedAccess,
					delegated,
					total
				)
			})
		];
		if (!client.isPublic) {
			parts.push(
				m.as_itself_summary({
					access: accessSummary(entry.clientAccess, entry.clientPermissionIds.length, total)
				})
			);
		}
		return parts.join(' · ');
	}
</script>

<SettingsSection
	title={m.api_access()}
	description={m.apis_name_may_request_tokens_for_description({ name: client.name })}
>
	{#snippet action()}
		{#if grants.length > 0}
			<Button variant="secondary" size="sm" onclick={() => (pickerOpen = true)}>
				<LucidePlus />
				{m.add_api_ellipsis()}
			</Button>
		{/if}
	{/snippet}
	<ul class="mt-2 flex flex-col rounded-md border">
		{#each grants as entry (entry.api.id)}
			<li
				class="flex min-h-(--row-2line) items-center gap-4 border-b py-2 ps-4 pe-2 last:border-b-0"
			>
				<div class="flex min-w-0 flex-1 flex-col">
					<span class="text-control font-medium">{entry.api.name}</span>
					<span class="text-caption text-fg-muted truncate font-mono" title={entry.api.resource}
						>{entry.api.resource}</span
					>
					{#if entry.cimdGrantedAccess}
						<span class="text-caption text-fg-muted">{m.granted_to_all_cimd_clients()}</span>
					{/if}
					<span class="text-caption text-fg-muted md:hidden">{summary(entry)}</span>
				</div>
				<span class="text-caption text-fg-muted max-md:hidden">{summary(entry)}</span>
				<ActionsMenu
					item={entry}
					label={m.actions_for_name({ name: entry.api.name })}
					size="icon-sm"
					actions={[
						{ label: m.edit_access_ellipsis(), icon: LucidePencil, onClick: openEdit },
						{
							label: m.remove_access_ellipsis(),
							icon: LucideBan,
							variant: 'danger',
							hidden: !entry.userDelegatedAccess && !entry.clientAccess,
							onClick: removeApi
						}
					]}
				/>
			</li>
		{:else}
			<li class="flex min-h-(--row-2line) flex-wrap items-center justify-between gap-2 ps-4 pe-2">
				<span class="text-control text-fg-muted min-w-0">
					{m.name_cant_request_tokens_for_any_api({ name: client.name })}
				</span>
				<Button variant="ghost" size="sm" onclick={() => (pickerOpen = true)}>
					<LucidePlus />
					{m.add_api_ellipsis()}
				</Button>
			</li>
		{/each}
	</ul>
</SettingsSection>

<ApiSelectionModal bind:open={pickerOpen} clientId={client.id} onSelect={addApi} />

{#if editing}
	<ApiPermissionsModal
		bind:open={modalOpen}
		api={editing.api}
		grant={editing}
		implicitUserAccess={editing.cimdGrantedAccess}
		implicitUserIds={editing.cimdGrantedPermissionIds}
		showClientAccess={!client.isPublic}
		onSave={(grant) => save(editing!, grant)}
	/>
{/if}
