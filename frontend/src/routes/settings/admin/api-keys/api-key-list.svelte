<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKey } from '$lib/types/api-key.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { toast } from 'svelte-sonner';
	import { LucideBan, LucideEllipsis } from 'lucide-svelte';
	import type { Paginated } from '$lib/types/pagination.type';
	import type { SearchPaginationSortRequest } from '$lib/types/sort-pagination.type';

	// Convert the apiKeys array to a Paginated object
	let apiKeys = $state<Paginated<ApiKey>>({
		data: [],
		pagination: {
			currentPage: 1,
			totalItems: 0,
			totalPages: 1,
			itemsPerPage: 10
		}
	});

	let requestOptions: SearchPaginationSortRequest | undefined = $state();
	const apiKeyService = new ApiKeyService();

	async function loadApiKeys(options?: SearchPaginationSortRequest) {
		try {
			const keys = await apiKeyService.list(options);
			// If the service doesn't support pagination yet, convert the response
			if (Array.isArray(keys)) {
				apiKeys = {
					data: keys,
					pagination: {
						currentPage: 1,
						totalItems: keys.length,
						totalPages: 1,
						itemsPerPage: keys.length || 10
					}
				};
			} else {
				apiKeys = keys;
			}
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
		loadApiKeys();
	});
</script>

<AdvancedTable
	items={apiKeys}
	{requestOptions}
	onRefresh={loadApiKeys}
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
			<DropdownMenu.Root>
				<DropdownMenu.Trigger asChild let:builder>
					<Button aria-haspopup="true" size="icon" variant="ghost" builders={[builder]}>
						<LucideEllipsis class="h-4 w-4" />
						<span class="sr-only">Toggle menu</span>
					</Button>
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item
						data-testid="actions-dropdown"
						class="text-red-500 focus:!text-red-700"
						on:click={() => revokeApiKey(item)}
					>
						<LucideBan data-testid="revoke-button" class="mr-2 h-4 w-4" />Revoke
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
