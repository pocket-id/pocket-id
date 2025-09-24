<script lang="ts">
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import type { ColumnSpec } from '$lib/components/pocket-id-table';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKey } from '$lib/types/api-key.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideBan } from '@lucide/svelte';
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
			message: m.are_you_sure_you_want_to_revoke_the_api_key_apikeyname({
				apiKeyName: apiKey.name
			}),
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

	const columns = [
		{ title: m.name(), accessorKey: 'name', sortable: true, cell: NameCell },
		{ title: m.description(), cell: DescriptionCell },
		{ title: m.expires_at(), accessorKey: 'expiresAt', sortable: true, cell: ExpiresAtCell },
		{ title: m.last_used(), accessorKey: 'lastUsedAt', sortable: true, cell: LastUsedAtCell },
		{ title: m.actions(), hidden: true, cell: RowActions }
	] satisfies ColumnSpec<ApiKey>[];
</script>

{#snippet NameCell({ item }: { item: ApiKey })}
	{item.name}
{/snippet}

{#snippet DescriptionCell({ item }: { item: ApiKey })}
	<span class="text-muted-foreground">{item.description}</span>
{/snippet}

{#snippet ExpiresAtCell({ item }: { item: ApiKey })}
	<span class="text-muted-foreground">{formatDate(item.expiresAt)}</span>
{/snippet}

{#snippet LastUsedAtCell({ item }: { item: ApiKey })}
	<span class="text-muted-foreground">{formatDate(item.lastUsedAt)}</span>
{/snippet}

{#snippet RowActions({ item }: { item: ApiKey })}
	<Button onclick={() => revokeApiKey(item)} size="sm" variant="outline" aria-label={m.revoke()}>
		<LucideBan class="text-destructive size-3" />
	</Button>
{/snippet}

<PocketIdTable
	items={apiKeys}
	bind:requestOptions
	onRefresh={async (o) => (apiKeys = await apiKeyService.list(o))}
	{columns}
	persistKey="pocket-id-api-keys-table"
	rowActions={RowActions}
/>
