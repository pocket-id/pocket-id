<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKey } from '$lib/types/api-key.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideBan } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

	let {
		apiKeys,
		requestOptions
	}: {
		apiKeys: Paginated<ApiKey>;
		requestOptions: SearchPaginationSortRequest;
	} = $props();

	const apiKeyService = new ApiKeyService();

	function formatDate(dateStr: string | undefined) {
		if (!dateStr) return m.never();
		return new Date(dateStr).toLocaleString();
	}

	function revokeApiKey(apiKey: ApiKey) {
		openConfirmDialog({
			title: m.revoke_api_key(),
			message: m.are_you_sure_you_want_to_revoke_the_api_key_apikeyname({ apiKeyName: apiKey.name }),
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					try {
						await apiKeyService.revoke(apiKey.id);
						apiKeys = await apiKeyService.list(requestOptions);
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
	items={apiKeys}
	{requestOptions}
	onRefresh={async (o) => (apiKeys = await apiKeyService.list(o))}
	withoutSearch
	columns={[
		{ label: m.name(), sortColumn: 'name' },
		{ label: m.description() },
		{ label: m.expires_at(), sortColumn: 'expiresAt' },
		{ label: m.last_used(), sortColumn: 'lastUsedAt' },
		{ label: m.actions(), hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.name}</Table.Cell>
		<Table.Cell class="text-muted-foreground">{item.description || '-'}</Table.Cell>
		<Table.Cell>{formatDate(item.expiresAt)}</Table.Cell>
		<Table.Cell>{formatDate(item.lastUsedAt)}</Table.Cell>
		<Table.Cell class="flex justify-end">
			<Button on:click={() => revokeApiKey(item)} size="sm" variant="outline" aria-label={m.revoke()}
				><LucideBan class="h-3 w-3 text-red-500" /></Button
			>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
