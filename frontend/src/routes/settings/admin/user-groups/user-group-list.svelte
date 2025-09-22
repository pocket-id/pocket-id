<script lang="ts">
	import { goto } from '$app/navigation';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Badge } from '$lib/components/ui/badge/index';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { UserGroup, UserGroupWithUserCount } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import Ellipsis from '@lucide/svelte/icons/ellipsis';
	import { toast } from 'svelte-sonner';
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
	import type { ColumnSpec } from '$lib/components/pocket-id-table';

	let {
		userGroups,
		requestOptions
	}: {
		userGroups: Paginated<UserGroupWithUserCount>;
		requestOptions: SearchPaginationSortRequest;
	} = $props();

	const userGroupService = new UserGroupService();

	async function deleteUserGroup(userGroup: UserGroup) {
		openConfirmDialog({
			title: m.delete_name({ name: userGroup.name }),
			message: m.are_you_sure_you_want_to_delete_this_user_group(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await userGroupService.remove(userGroup.id);
						userGroups = await userGroupService.list(requestOptions!);
						toast.success(m.user_group_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	const columns = [
		{ title: m.friendly_name(), accessorKey: 'friendlyName', sortable: true },
		{ title: m.name(), accessorKey: 'name', sortable: true },
		{ title: m.user_count(), accessorKey: 'userCount', sortable: true },
		...($appConfigStore.ldapEnabled
			? [{ title: m.source(), accessorKey: 'ldapId' as const, sortable: true, cell: SourceCell }]
			: []),
		{ title: m.actions(), hidden: true }
	] satisfies ColumnSpec<UserGroupWithUserCount>[];
</script>

{#snippet SourceCell({ item }: { item: UserGroupWithUserCount })}
	<Badge class="rounded-full" variant={item.ldapId ? 'default' : 'outline'}>
		{item.ldapId ? m.ldap() : m.local()}
	</Badge>
{/snippet}

{#snippet RowActions({ item }: { item: UserGroupWithUserCount })}
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			<Ellipsis class="size-4" />
			<span class="sr-only">{m.toggle_menu()}</span>
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end">
			<DropdownMenu.Item onclick={() => goto(`/settings/admin/user-groups/${item.id}`)}>
				<LucidePencil class="mr-2 size-4" />
				{m.edit()}
			</DropdownMenu.Item>
			{#if !item.ldapId || !$appConfigStore.ldapEnabled}
				<DropdownMenu.Item
					class="text-red-500 focus:!text-red-700"
					onclick={() => deleteUserGroup(item)}
				>
					<LucideTrash class="mr-2 size-4" />{m.delete()}
				</DropdownMenu.Item>
			{/if}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/snippet}

<PocketIdTable
	items={userGroups}
	bind:requestOptions
	onRefresh={async (opts) => (userGroups = await userGroupService.list(opts))}
	{columns}
	persistKey="pocket-id-user-groups"
	rowActions={RowActions}
/>
