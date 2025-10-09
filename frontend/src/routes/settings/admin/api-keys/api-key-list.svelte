<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { m } from '$lib/paraglide/messages';
	import ApiKeyService from '$lib/services/api-key-service';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { ApiKey } from '$lib/types/api-key.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideBan } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	const apiKeyService = new ApiKeyService();

	let tableRef: AdvancedTable<ApiKey>;

	export const refresh = () => tableRef.refresh();

	function formatDate(dateStr: string | undefined) {
		if (!dateStr) return m.never();
		return new Date(dateStr).toLocaleString();
	}

	const columns: AdvancedTableColumn<ApiKey>[] = [
		{ label: m.name(), column: 'name', sortable: true },
		{
			label: m.description(),
			column: 'description'
		},
		{
			label: m.expires_at(),
			column: 'expiresAt',
			sortable: true,
			value: (item) => formatDate(item.expiresAt)
		},
		{
			label: m.last_used(),
			column: 'lastUsedAt',
			sortable: true,
			value: (item) => formatDate(item.lastUsedAt)
		},
		{
			label: m.created(),
			column: 'createdAt',
			sortable: true,
			hidden: true,
			value: (item) => formatDate(item.createdAt)
		}
	];

	const actions: CreateAdvancedTableActions<ApiKey> = (apiKey) => [
		{
			label: m.revoke(),
			icon: LucideBan,
			variant: 'danger',
			onClick: (apiKey) => revokeApiKey(apiKey)
		}
	];

	function revokeApiKey(apiKey: ApiKey) {
		openConfirmDialog({
			title: m.revoke_api_key(),
			message: m.are_you_sure_you_want_to_revoke_the_api_key_apikeyname({
				apiKeyName: apiKey.name
			}),
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					try {
						await apiKeyService.revoke(apiKey.id);
						await refresh();
						toast.success(m.api_key_revoked_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<AdvancedTable
	id="api-key-list"
	bind:this={tableRef}
	fetchCallback={apiKeyService.list}
	defaultSort={{ column: 'lastUsedAt', direction: 'desc' }}
	withoutSearch
	{columns}
	{actions}
/>
