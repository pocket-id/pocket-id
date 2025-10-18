<script lang="ts">
	import { goto } from '$app/navigation';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import ImageBox from '$lib/components/image-box.svelte';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { OidcClient, OidcClientWithAllowedUserGroupsCount } from '$lib/types/oidc.type';
	import { cachedOidcClientLogo, cachedOidcClientDarkLogo } from '$lib/utils/cached-image-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { mode } from 'mode-watcher';

	const oidcService = new OIDCService();
	let tableRef: AdvancedTable<OidcClientWithAllowedUserGroupsCount>;

	export function refresh() {
		return tableRef?.refresh();
	}

	const isLightMode = $derived(mode.current === 'light');

	function getClientLogoUrl(client: OidcClientWithAllowedUserGroupsCount): string | null {
		if (isLightMode) {
			return client.hasLogo ? cachedOidcClientLogo.getUrl(client.id) : null;
		} else {
			return client.hasDarkLogo
				? cachedOidcClientDarkLogo.getUrl(client.id)
				: client.hasLogo
					? cachedOidcClientLogo.getUrl(client.id)
					: null;
		}
	}

	const booleanFilterValues = [
		{ label: m.enabled(), value: true },
		{ label: m.disabled(), value: false }
	];

	const columns: AdvancedTableColumn<OidcClientWithAllowedUserGroupsCount>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.logo(), key: 'logo', cell: LogoCell },
		{ label: m.name(), column: 'name', sortable: true },
		{
			label: m.oidc_allowed_group_count(),
			column: 'allowedUserGroupsCount',
			sortable: true,
			value: (item) =>
				item.allowedUserGroupsCount > 0 ? item.allowedUserGroupsCount : m.unrestricted()
		},
		{
			label: m.pkce(),
			column: 'pkceEnabled',
			sortable: true,
			hidden: true,
			filterableValues: booleanFilterValues
		},
		{
			label: m.reauthentication(),
			column: 'requiresReauthentication',
			sortable: true,
			filterableValues: booleanFilterValues
		},
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

	const actions: CreateAdvancedTableActions<OidcClientWithAllowedUserGroupsCount> = (_) => [
		{
			label: m.edit(),
			icon: LucidePencil,
			onClick: (client) => goto(`/settings/admin/oidc-clients/${client.id}`)
		},
		{
			label: m.delete(),
			icon: LucideTrash,
			variant: 'danger',
			onClick: (client) => deleteClient(client)
		}
	];

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
						await refresh();
						toast.success(m.oidc_client_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

{#snippet LogoCell({ item }: { item: OidcClientWithAllowedUserGroupsCount })}
	{@const logoUrl = getClientLogoUrl(item)}
	{#if logoUrl}
		<ImageBox class="size-12 rounded-lg" src={logoUrl} alt={m.name_logo({ name: item.name })} />
	{:else}
		<div class="bg-muted flex size-12 items-center justify-center rounded-lg text-lg font-bold">
			{item.name.charAt(0).toUpperCase()}
		</div>
	{/if}
{/snippet}

<AdvancedTable
	id="oidc-client-list"
	bind:this={tableRef}
	fetchCallback={oidcService.listClients}
	defaultSort={{ column: 'name', direction: 'asc' }}
	{columns}
	{actions}
/>
