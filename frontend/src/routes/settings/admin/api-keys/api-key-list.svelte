<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKey } from '$lib/types/api-key.type';
	import type { Paginated } from '$lib/types/pagination.type';
	import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideBan } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';

	let {
		apiKeys: initialApiKeys = {
			data: [],
			pagination: { currentPage: 1, itemsPerPage: 10, totalItems: 0, totalPages: 0 }
		}
	}: {
		apiKeys?: Paginated<ApiKey>;
	} = $props();

	let apiKeys = $state<Paginated<ApiKey>>(initialApiKeys);
	let requestOptions: SearchPaginationSortRequest | undefined = $state();
	const apiKeyService = new ApiKeyService();

	// Update the local state whenever the prop changes
	$effect(() => {
		apiKeys = initialApiKeys;
	});

	async function loadApiKeys(options?: SearchPaginationSortRequest) {
		try {
			const keys = await apiKeyService.list(options);
			apiKeys = keys;
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function formatDate(dateStr: string | undefined) {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleString();
	}

	function revokeApiKey(apiKey: ApiKey) {
		openConfirmDialog({
			title: 'Revoke API Key',
			message: `Are you sure you want to revoke the API key "${apiKey.name}"? This action cannot be undone.`,
			confirm: {
				label: 'Revoke',
				destructive: true,
				action: async () => {
					try {
						await apiKeyService.revoke(apiKey.id);
						// Reload the list after revoking
						loadApiKeys(requestOptions);
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
	columns={[
		{ label: 'Name', sortColumn: 'name' },
		{ label: 'Description', sortColumn: 'description' },
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
			<Button
				size="sm"
				variant="ghost"
				data-testid="revoke-button"
				class="text-red-500 hover:bg-red-50 hover:text-red-700"
				on:click={() => revokeApiKey(item)}
			>
				<LucideBan class="mr-2 h-4 w-4" />
				Revoke
			</Button>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
