<script lang="ts">
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { Api, ApiPermission } from '$lib/types/api.type';
	import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
	import { axiosErrorToast } from '$lib/utils/error-util';

	let {
		open = $bindable(),
		api,
		allowedIds,
		onSave
	}: {
		open: boolean;
		api: Api;
		allowedIds: string[];
		onSave: (permissionIds: string[]) => Promise<void>;
	} = $props();

	let working = $state<string[]>([]);
	let saving = $state(false);

	$effect(() => {
		if (open) {
			working = [...allowedIds];
		}
	});

	const columns: AdvancedTableColumn<ApiPermission>[] = [
		{ label: m.name(), column: 'name', sortable: true },
		{ label: m.key(), key: 'key', cell: KeyCell },
		{ label: m.description(), key: 'description', value: (p) => p.description ?? '' }
	];

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
			await onSave(working);
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

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-250">
		<Dialog.Header>
			<Dialog.Title>{api.name}</Dialog.Title>
			<Dialog.Description>{m.select_the_permissions_this_client_may_request()}</Dialog.Description>
		</Dialog.Header>

		<AdvancedTable
			id={`api-access-${api.id}`}
			{columns}
			{fetchCallback}
			bind:selectedIds={working}
			defaultSort={{ column: 'name', direction: 'asc' }}
		/>

		<div class="mt-4 flex justify-end gap-2">
			<Button variant="secondary" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button isLoading={saving} onclick={save}>{m.save()}</Button>
		</div>
	</Dialog.Content>
</Dialog.Root>
