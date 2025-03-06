<script lang="ts">
	import { goto } from '$app/navigation';
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { Badge } from '$lib/components/ui/badge/index';
	import { buttonVariants } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Table from '$lib/components/ui/table';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { User } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideLink, LucidePencil, LucideTrash } from 'lucide-svelte';
	import Ellipsis from 'lucide-svelte/icons/ellipsis';
	import { toast } from 'svelte-sonner';
	import OneTimeLinkModal from './one-time-link-modal.svelte';

	let { users = $bindable() }: { users: Paginated<User> } = $props();

	let userIdToCreateOneTimeLink: string | null = $state(null);

	let requestOptions: SearchPaginationSortRequest | undefined = $state({
		sort: { column: 'firstName', direction: 'asc' },
		pagination: {
			page: users.pagination.currentPage,
			limit: users.pagination.itemsPerPage
		}
	});

	const userService = new UserService();

	async function deleteUser(user: User) {
		openConfirmDialog({
			title: `Delete ${user.firstName} ${user.lastName}`,
			message: 'Are you sure you want to delete this user?',
			confirm: {
				label: 'Delete',
				destructive: true,
				action: async () => {
					try {
						await userService.remove(user.id);
						users = await userService.list(requestOptions!);
					} catch (e) {
						axiosErrorToast(e);
					}
					toast.success('User deleted successfully');
				}
			}
		});
	}
</script>

<AdvancedTable
	items={users}
	{requestOptions}
	defaultSort={{ column: 'firstName', direction: 'asc' }}
	onRefresh={async (options) => (users = await userService.list(options))}
	columns={[
		{ label: 'First name', sortColumn: 'firstName' },
		{ label: 'Last name', sortColumn: 'lastName' },
		{ label: 'Email', sortColumn: 'email' },
		{ label: 'Username', sortColumn: 'username' },
		{ label: 'Role', sortColumn: 'isAdmin' },
		...($appConfigStore.ldapEnabled ? [{ label: 'Source' }] : []),
		{ label: 'Actions', hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.firstName}</Table.Cell>
		<Table.Cell>{item.lastName}</Table.Cell>
		<Table.Cell>{item.email}</Table.Cell>
		<Table.Cell>{item.username}</Table.Cell>
		<Table.Cell>
			<Badge variant="outline">{item.isAdmin ? 'Admin' : 'User'}</Badge>
		</Table.Cell>
		{#if $appConfigStore.ldapEnabled}
			<Table.Cell>
				<Badge variant={item.ldapId ? 'default' : 'outline'}>{item.ldapId ? 'LDAP' : 'Local'}</Badge
				>
			</Table.Cell>
		{/if}
		<Table.Cell>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger class={buttonVariants({ variant: 'ghost', size: 'icon' })}>
					<Ellipsis class="h-4 w-4" />
					<span class="sr-only">Toggle menu</span>
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item onclick={() => (userIdToCreateOneTimeLink = item.id)}
						><LucideLink class="mr-2 h-4 w-4" />One-time link</DropdownMenu.Item
					>
					<DropdownMenu.Item onclick={() => goto(`/settings/admin/users/${item.id}`)}
						><LucidePencil class="mr-2 h-4 w-4" /> Edit</DropdownMenu.Item
					>
					{#if !item.ldapId || !$appConfigStore.ldapEnabled}
						<DropdownMenu.Item
							class="text-red-500 focus:!text-red-700"
							onclick={() => deleteUser(item)}
							><LucideTrash class="mr-2 h-4 w-4" />Delete</DropdownMenu.Item
						>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</Table.Cell>
	{/snippet}
</AdvancedTable>

<OneTimeLinkModal userId={userIdToCreateOneTimeLink} />
