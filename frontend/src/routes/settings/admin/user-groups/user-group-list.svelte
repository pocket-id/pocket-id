<script lang="ts">
	import { goto } from '$app/navigation';
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Badge } from '$lib/components/ui/badge/index';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { UserGroup, UserGroupWithUserCount } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import Ellipsis from '@lucide/svelte/icons/ellipsis';
	import { toast } from 'svelte-sonner';

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
</script>

<AdvancedTable
	items={userGroups}
	onRefresh={async (o) => (userGroups = await userGroupService.list(o))}
	{requestOptions}
	columns={[
		{ label: m.friendly_name(), sortColumn: 'friendlyName' },
		{ label: m.name(), sortColumn: 'name' },
		{ label: m.user_count(), sortColumn: 'userCount' },
		...($appConfigStore.ldapEnabled ? [{ label: m.source() }] : []),
		{ label: m.actions(), hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.friendlyName}</Table.Cell>
		<Table.Cell>{item.name}</Table.Cell>
		<Table.Cell>{item.userCount}</Table.Cell>
		{#if $appConfigStore.ldapEnabled}
			<Table.Cell>
				<Badge class="rounded-full" variant={item.ldapId ? 'default' : 'outline'}
					>{item.ldapId ? m.ldap() : m.local()}</Badge
				>
			</Table.Cell>
		{/if}
		<Table.Cell class="flex justify-end">
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					<Ellipsis class="size-4" />
					<span class="sr-only">{m.toggle_menu()}</span>
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item onclick={() => goto(`/settings/admin/user-groups/${item.id}`)}
						><LucidePencil class="mr-2 size-4" /> {m.edit()}</DropdownMenu.Item
					>
					{#if !item.ldapId || !$appConfigStore.ldapEnabled}
						<DropdownMenu.Item
							class="text-red-500 focus:!text-red-700"
							onclick={() => deleteUserGroup(item)}
							><LucideTrash class="mr-2 size-4" />{m.delete()}</DropdownMenu.Item
						>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
