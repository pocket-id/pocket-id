<script lang="ts">
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Button } from '$lib/components/ui/button';
	import Checkbox from '$lib/components/ui/checkbox/checkbox.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { Api, ApiClientGrant, ApiPermission } from '$lib/types/api.type';
	import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
	import { axiosErrorToast } from '$lib/utils/error-util';

	let {
		open = $bindable(),
		api,
		grant,
		implicitUserAccess = false,
		implicitUserIds = [],
		showClientAccess,
		title,
		onSave
	}: {
		open: boolean;
		api: Api;
		grant: ApiClientGrant;
		implicitUserAccess?: boolean;
		implicitUserIds?: string[];
		showClientAccess: boolean;
		title?: string;
		onSave: (grant: ApiClientGrant) => Promise<void>;
	} = $props();

	let workingUserAccess = $state(false);
	let workingClientAccess = $state(false);
	let workingUser = $state<string[]>([]);
	let workingClient = $state<string[]>([]);
	let saving = $state(false);

	$effect(() => {
		if (open) {
			workingUserAccess = grant.userDelegatedAccess;
			workingClientAccess = grant.clientAccess;
			workingUser = [...grant.userDelegatedPermissionIds];
			workingClient = [...grant.clientPermissionIds];
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

	// A permission only makes sense together with access to the API, so checking one turns the access on and turning access off drops the selection
	function toggleUserPermission(id: string, checked: boolean) {
		workingUser = toggle(workingUser, id, checked);
		if (checked) {
			workingUserAccess = true;
		}
	}

	function toggleClientPermission(id: string, checked: boolean) {
		workingClient = toggle(workingClient, id, checked);
		if (checked) {
			workingClientAccess = true;
		}
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
			await onSave({
				userDelegatedAccess: workingUserAccess,
				clientAccess: showClientAccess && workingClientAccess,
				userDelegatedPermissionIds: workingUserAccess ? workingUser : [],
				clientPermissionIds: showClientAccess && workingClientAccess ? workingClient : []
			});
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
	{@const implicit = implicitUserIds.includes(item.id)}
	<Checkbox
		aria-label={`${m.user_delegated_access()}: ${item.name}`}
		checked={implicit || workingUser.includes(item.id)}
		disabled={implicit}
		title={implicit ? m.granted_through_cimd_access() : undefined}
		onCheckedChange={(checked: boolean) => toggleUserPermission(item.id, checked)}
	/>
{/snippet}

{#snippet ClientAccessCell({ item }: { item: ApiPermission })}
	<Checkbox
		aria-label={`${m.client_access()}: ${item.name}`}
		checked={workingClient.includes(item.id)}
		onCheckedChange={(checked: boolean) => toggleClientPermission(item.id, checked)}
	/>
{/snippet}

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-250">
		<Dialog.Header>
			<Dialog.Title>{title ?? api.name}</Dialog.Title>
			<Dialog.Description>
				{m.select_the_access_this_client_may_request()}
				{#if !showClientAccess}
					{m.client_access_unavailable_for_public_clients()}
				{/if}
				{#if implicitUserAccess}
					{m.access_granted_through_cimd_access()}
				{/if}
			</Dialog.Description>
		</Dialog.Header>

		<div class="flex flex-col gap-4 sm:flex-row sm:gap-10">
			<SwitchWithLabel
				id={`api-user-access-${api.id}`}
				label={m.user_delegated_access()}
				description={m.user_delegated_access_description()}
				checked={implicitUserAccess || workingUserAccess}
				disabled={implicitUserAccess}
				onCheckedChange={(checked) => {
					workingUserAccess = checked;
					if (!checked) {
						workingUser = [];
					}
				}}
			/>
			{#if showClientAccess}
				<SwitchWithLabel
					id={`api-client-access-${api.id}`}
					label={m.client_access()}
					description={m.client_access_description()}
					bind:checked={workingClientAccess}
					onCheckedChange={(checked) => {
						if (!checked) {
							workingClient = [];
						}
					}}
				/>
			{/if}
		</div>

		{#if api.permissions.length > 0}
			<div class="overflow-auto">
				<AdvancedTable
					id={`api-access-grants-${api.id}`}
					{columns}
					{fetchCallback}
					defaultSort={{ column: 'name', direction: 'asc' }}
				/>
			</div>
		{/if}

		<div class="mt-4 flex justify-end gap-2">
			<Button variant="secondary" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button isLoading={saving} onclick={save}>{m.save()}</Button>
		</div>
	</Dialog.Content>
</Dialog.Root>
