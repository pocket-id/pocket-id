<script lang="ts">
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Button } from '$lib/components/ui/button';
	import Checkbox from '$lib/components/ui/checkbox/checkbox.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { Api, ApiPermission } from '$lib/types/api.type';
	import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
	import { axiosErrorToast } from '$lib/utils/error-util';

	let {
		open = $bindable(),
		api,
		userAllowedIds,
		clientAllowedIds,
		showClientAccess,
		onSave
	}: {
		open: boolean;
		api: Api;
		userAllowedIds: string[];
		clientAllowedIds: string[];
		showClientAccess: boolean;
		onSave: (userPermissionIds: string[], clientPermissionIds: string[]) => Promise<void>;
	} = $props();

	let workingUser = $state<string[]>([]);
	let workingClient = $state<string[]>([]);
	let saving = $state(false);

	$effect(() => {
		if (open) {
			workingUser = [...userAllowedIds];
			workingClient = [...clientAllowedIds];
		}
	});

	const columns: AdvancedTableColumn<ApiPermission>[] = $derived([
		{ label: m.name(), column: 'name', sortable: true },
		{ label: m.key(), key: 'key', cell: KeyCell },
		{ label: m.description(), key: 'description', value: (p) => p.description ?? '' },
		{ label: m.user_delegated_access(), key: 'userDelegated', cell: UserDelegatedCell },
		...(showClientAccess
			? [{ label: m.client_access(), key: 'clientAccess', cell: ClientAccessCell }]
			: [])
	]);

	function toggle(ids: string[], id: string, checked: boolean) {
		if (checked) {
			return ids.includes(id) ? ids : [...ids, id];
		}
		return ids.filter((existing) => existing !== id);
	}

	function fetchCallback(options: ListRequestOptions): Promise<Paginated<ApiPermission>> {
		let data = api.permissions;

		const search = options.search?.toLowerCase();
		if (search) {
			data = data.filter(
				(p) =>
					p.key.toLowerCase().includes(search) ||
					p.name.toLowerCase().includes(search) ||
					(p.description ?? '').toLowerCase().includes(search)
			);
		}

		const column = options.sort?.column;
		if (column) {
			const direction = options.sort?.direction === 'desc' ? -1 : 1;
			data = [...data].sort(
				(a, b) =>
					String((a as Record<string, unknown>)[column] ?? '').localeCompare(
						String((b as Record<string, unknown>)[column] ?? '')
					) * direction
			);
		}

		const page = options.pagination?.page ?? 1;
		const limit = options.pagination?.limit ?? 20;
		const start = (page - 1) * limit;

		return Promise.resolve({
			data: data.slice(start, start + limit),
			pagination: {
				totalPages: Math.max(1, Math.ceil(data.length / limit)),
				totalItems: data.length,
				currentPage: page,
				itemsPerPage: limit
			}
		});
	}

	async function save() {
		saving = true;
		try {
			await onSave(workingUser, workingClient);
			open = false;
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			saving = false;
		}
	}
</script>

{#snippet KeyCell({ item }: { item: ApiPermission })}
	<span class="font-mono text-xs">{item.key}</span>
{/snippet}

{#snippet UserDelegatedCell({ item }: { item: ApiPermission })}
	<Checkbox
		aria-label={`${m.user_delegated_access()}: ${item.name}`}
		checked={workingUser.includes(item.id)}
		onCheckedChange={(checked: boolean) => (workingUser = toggle(workingUser, item.id, checked))}
	/>
{/snippet}

{#snippet ClientAccessCell({ item }: { item: ApiPermission })}
	<Checkbox
		aria-label={`${m.client_access()}: ${item.name}`}
		checked={workingClient.includes(item.id)}
		onCheckedChange={(checked: boolean) =>
			(workingClient = toggle(workingClient, item.id, checked))}
	/>
{/snippet}

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-250">
		<Dialog.Header>
			<Dialog.Title>{api.name}</Dialog.Title>
			<Dialog.Description>
				{m.select_the_permissions_this_client_may_request()}
				{#if !showClientAccess}
					{m.client_access_unavailable_for_public_clients()}
				{/if}
			</Dialog.Description>
		</Dialog.Header>

		<AdvancedTable
			id={`api-access-grants-${api.id}`}
			{columns}
			{fetchCallback}
			defaultSort={{ column: 'name', direction: 'asc' }}
		/>

		<div class="mt-4 flex justify-end gap-2">
			<Button variant="secondary" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button isLoading={saving} onclick={save}>{m.save()}</Button>
		</div>
	</Dialog.Content>
</Dialog.Root>
