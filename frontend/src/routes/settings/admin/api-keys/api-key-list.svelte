<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKey } from '$lib/types/api-key.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideBan } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

	let {
		apiKeys: initialApiKeys
	}: {
		apiKeys: Paginated<ApiKey>;
	} = $props();

	let apiKeys = $state<Paginated<ApiKey>>(initialApiKeys);
	let requestOptions: SearchPaginationSortRequest | undefined = $state({
		sort: {
			column: 'lastUsedAt',
			direction: 'desc'
		}
	});
	const apiKeyService = new ApiKeyService();

	// Update the local state whenever the prop changes
	$effect(() => {
		apiKeys = initialApiKeys;
	});

	function formatDate(dateStr: string | undefined) {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleString();
	}

	function revokeApiKey(apiKey: ApiKey) {
		openConfirmDialog({
			title: 'Revoke API Key',
			message: `Are you sure you want to revoke the API key "${apiKey.name}"? This will break any integrations using this key.`,
			confirm: {
				label: 'Revoke',
				destructive: true,
				action: async () => {
					try {
						await apiKeyService.revoke(apiKey.id);
						apiKeys = await apiKeyService.list(requestOptions);
						toast.success('API key revoked successfully');
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	$effect(() => {
		// Initial load uses the server-side data
		apiKeys = initialApiKeys;
	});
</script>

<AdvancedTable
	items={apiKeys}
	{requestOptions}
	onRefresh={async (o) => (apiKeys = await apiKeyService.list(o))}
	withoutSearch
	defaultSort={{ column: 'lastUsedAt', direction: 'desc' }}
	columns={[
		{ label: 'Name', sortColumn: 'name' },
		{ label: 'Description' },
		{ label: 'Expires At', sortColumn: 'expiresAt' },
		{ label: 'Last Used', sortColumn: 'lastUsedAt' },
		{ label: 'Actions', hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.name}</Table.Cell>
		<Table.Cell class="text-muted-foreground">{item.description || '-'}</Table.Cell>
		<Table.Cell>{formatDate(item.expiresAt)}</Table.Cell>
		<Table.Cell>{formatDate(item.lastUsedAt)}</Table.Cell>
		<Table.Cell class="flex justify-end">
			<Button on:click={() => revokeApiKey(item)} size="sm" variant="outline" aria-label="Revoke"
				><LucideBan class="h-3 w-3 text-red-500" /></Button
			>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
