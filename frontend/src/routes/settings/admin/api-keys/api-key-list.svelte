<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/confirm-dialog.svelte';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKey } from '$lib/types/api-key.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { toast } from 'svelte-sonner';
	import { LucideTrash } from 'lucide-svelte';

	let apiKeys = $state<ApiKey[]>([]);
	const apiKeyService = new ApiKeyService();

	async function loadApiKeys() {
		try {
			apiKeys = await apiKeyService.list();
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
						apiKeys = apiKeys.filter((k) => k.id !== apiKey.id);
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

{#if apiKeys.length === 0}
	<Card.Content class="text-muted-foreground py-6 text-center">
		No API keys found. Create one to get started.
	</Card.Content>
{:else}
	<div class="overflow-x-auto">
		<table class="w-full">
			<thead>
				<tr class="border-b">
					<th class="pb-2 text-left font-medium">Name</th>
					<th class="pb-2 text-left font-medium">Description</th>
					<th class="pb-2 text-left font-medium">Expires</th>
					<th class="pb-2 text-left font-medium">Last Used</th>
					<th class="pb-2 text-right font-medium">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each apiKeys as apiKey}
					<tr class="border-b">
						<td class="py-3">{apiKey.name}</td>
						<td class="text-muted-foreground py-3">{apiKey.description || '-'}</td>
						<td class="py-3">{formatDate(apiKey.expiresAt)}</td>
						<td class="py-3">{formatDate(apiKey.lastUsedAt)}</td>
						<td class="py-3 text-right">
							<Button
								on:click={() => revokeApiKey(apiKey)}
								size="sm"
								variant="ghost"
								class="text-red-500 hover:text-red-600"
							>
								<LucideTrash class="h-4 w-4" />
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
