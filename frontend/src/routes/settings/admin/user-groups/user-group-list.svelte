<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Badge } from '$lib/components/ui/badge/index';
	import { Button } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Table from '$lib/components/ui/table';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { UserGroup, UserGroupWithUserCount } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from 'lucide-svelte';
	import Ellipsis from 'lucide-svelte/icons/ellipsis';
	import { toast } from 'svelte-sonner';
	import { onMount } from 'svelte';

	let { userGroups: initialUserGroups }: { userGroups: Paginated<UserGroupWithUserCount> } =
		$props();

	let userGroups = $state<Paginated<UserGroupWithUserCount>>(initialUserGroups);
	let requestOptions: SearchPaginationSortRequest | undefined = $state({
		sort: { column: 'friendlyName', direction: 'asc' }
	});

	// Fetch the sorted data when the component is mounted
	onMount(async () => {
		userGroups = await userGroupService.list(requestOptions!);
	});

	const userGroupService = new UserGroupService();

	async function deleteUserGroup(userGroup: UserGroup) {
		openConfirmDialog({
			title: `Delete ${userGroup.name}`,
			message: 'Are you sure you want to delete this user group?',
			confirm: {
				label: 'Delete',
				destructive: true,
				action: async () => {
					try {
						await userGroupService.remove(userGroup.id);
						userGroups = await userGroupService.list(requestOptions!);
						toast.success('User group deleted successfully');
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
	defaultSort={{ column: 'friendlyName', direction: 'asc' }}
	columns={[
		{ label: 'Friendly Name', sortColumn: 'friendlyName' },
		{ label: 'Name', sortColumn: 'name' },
		{ label: 'User Count', sortColumn: 'userCount' },
		...($appConfigStore.ldapEnabled ? [{ label: 'Source' }] : []),
		{ label: 'Actions', hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.friendlyName}</Table.Cell>
		<Table.Cell>{item.name}</Table.Cell>
		<Table.Cell>{item.userCount}</Table.Cell>
		{#if $appConfigStore.ldapEnabled}
			<Table.Cell>
				<Badge variant={item.ldapId ? 'default' : 'outline'}>{item.ldapId ? 'LDAP' : 'Local'}</Badge
				>
			</Table.Cell>
		{/if}
		<Table.Cell class="flex justify-end">
			<DropdownMenu.Root>
				<DropdownMenu.Trigger asChild let:builder>
					<Button aria-haspopup="true" size="icon" variant="ghost" builders={[builder]}>
						<Ellipsis class="h-4 w-4" />
						<span class="sr-only">Toggle menu</span>
					</Button>
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item href="/settings/admin/user-groups/{item.id}"
						><LucidePencil class="mr-2 h-4 w-4" /> Edit</DropdownMenu.Item
					>
					{#if !item.ldapId || !$appConfigStore.ldapEnabled}
						<DropdownMenu.Item
							class="text-red-500 focus:!text-red-700"
							on:click={() => deleteUserGroup(item)}
							><LucideTrash class="mr-2 h-4 w-4" />Delete</DropdownMenu.Item
						>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</Table.Cell>
	{/snippet}
</AdvancedTable>
