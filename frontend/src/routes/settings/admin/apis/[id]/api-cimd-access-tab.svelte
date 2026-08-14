<script lang="ts">
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import { Button } from '$lib/components/ui/button';
	import Checkbox from '$lib/components/ui/checkbox/checkbox.svelte';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages';
	import type { Api, ApiCimdAccessUpdate } from '$lib/types/api.type';

	let { api, onSave }: { api: Api; onSave: (update: ApiCimdAccessUpdate) => Promise<void> } =
		$props();

	let enabled = $state(api.allowCimdClients);
	let selectedIds = $state<string[]>(
		api.permissions.filter((p) => p.allowedForCimdClients).map((p) => p.id)
	);

	function toggle(id: string, checked: boolean) {
		selectedIds = checked
			? [...selectedIds, id]
			: selectedIds.filter((selectedId) => selectedId !== id);
	}

	async function save() {
		await onSave({ enabled, permissionIds: selectedIds });
	}
</script>

<SwitchWithLabel
	id="allow-cimd-clients"
	class="my-5"
	label={m.allow_all_metadata_document_clients()}
	description={m.allow_all_metadata_document_clients_description()}
	bind:checked={enabled}
/>

{#if enabled}
	<div class="mt-6">
		{#if api.permissions.length > 0}
			<Label class="mb-3">{m.granted_permissions()}</Label>
			<div class="flex flex-col gap-3">
				{#each api.permissions as permission (permission.id)}
					<div class="flex items-start gap-2">
						<Checkbox
							id={`cimd-permission-${permission.id}`}
							class="mt-0.5"
							checked={selectedIds.includes(permission.id)}
							onCheckedChange={(checked: boolean) => toggle(permission.id, checked)}
						/>
						<div class="grid gap-1 leading-none">
							<Label
								for={`cimd-permission-${permission.id}`}
								class="mb-0 text-sm leading-none font-medium"
							>
								{permission.name}
							</Label>
							<p class="text-muted-foreground font-mono text-xs">{permission.key}</p>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
{/if}

<div class="mt-5 flex justify-end">
	<Button usePromiseLoading onclick={save}>{m.save()}</Button>
</div>
