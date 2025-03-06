<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import OIDCService from '$lib/services/oidc-service';
	import type { OidcClient } from '$lib/types/oidc.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import OneTimeLinkModal from './client-secret.svelte';

	let {
		clients: initialClients
	}: {
		clients: Paginated<OidcClient>;
	} = $props();
	let clients = $state<Paginated<OidcClient>>(initialClients);
	let oneTimeLink = $state<string | null>(null);
	let requestOptions: SearchPaginationSortRequest | undefined = $state({
		sort: { column: 'name', direction: 'asc' },
		pagination: {
			page: initialClients.pagination.currentPage,
			limit: initialClients.pagination.itemsPerPage
		}
	});

	$effect(() => {
		clients = initialClients;
	});

	const oidcService = new OIDCService();

	async function deleteClient(client: OidcClient) {
		openConfirmDialog({
			title: `Delete ${client.name}`,
			message: 'Are you sure you want to delete this OIDC client?',
			confirm: {
				label: 'Delete',
				destructive: true,
				action: async () => {
					try {
						await oidcService.removeClient(client.id);
						clients = await oidcService.listClients(requestOptions!);
						toast.success('OIDC client deleted successfully');
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<AdvancedTable
	items={clients}
	{requestOptions}
	defaultSort={{ column: 'name', direction: 'asc' }}
	onRefresh={async (o) => (clients = await oidcService.listClients(o))}
	columns={[
		{ label: 'Logo' },
		{ label: 'Name', sortColumn: 'name' },
		{ label: 'Actions', hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell class="w-8 font-medium">
			{#if item.hasLogo}
				<div class="h-8 w-8">
					<img
						class="m-auto max-h-full max-w-full object-contain"
						src="/api/oidc/clients/{item.id}/logo"
						alt="{item.name} logo"
					/>
				</div>
			{/if}
		</Table.Cell>
		<Table.Cell class="font-medium">{item.name}</Table.Cell>
		<Table.Cell class="flex justify-end gap-1">
			<Button
				href="/settings/admin/oidc-clients/{item.id}"
				size="sm"
				variant="outline"
				aria-label="Edit"><LucidePencil class="h-3 w-3 " /></Button
			>
			<Button on:click={() => deleteClient(item)} size="sm" variant="outline" aria-label="Delete"
				><LucideTrash class="h-3 w-3 text-red-500" /></Button
			>
		</Table.Cell>
	{/snippet}
</AdvancedTable>

<OneTimeLinkModal {oneTimeLink} />
