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
	import {
		LucideLink,
		LucidePencil,
		LucideTrash,
		LucideUserCheck,
		LucideUserX
	} from 'lucide-svelte';
	import Ellipsis from 'lucide-svelte/icons/ellipsis';
	import { toast } from 'svelte-sonner';
	import OneTimeLinkModal from '$lib/components/one-time-link-modal.svelte';
	import { m } from '$lib/paraglide/messages';

	let {
		users = $bindable(),
		requestOptions
	}: { users: Paginated<User>; requestOptions: SearchPaginationSortRequest } = $props();

	let userIdToCreateOneTimeLink: string | null = $state(null);

	const userService = new UserService();

	async function deleteUser(user: User) {
		openConfirmDialog({
			title: m.delete_firstname_lastname({ firstName: user.firstName, lastName: user.lastName }),
			message: m.are_you_sure_you_want_to_delete_this_user(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await userService.remove(user.id);
						users = await userService.list(requestOptions!);
					} catch (e) {
						axiosErrorToast(e);
					}
					toast.success(m.user_deleted_successfully());
				}
			}
		});
	}

	async function enableUser(user: User) {
		await userService
			.setUserAttribute(user.id, 'disabled', false)
			.then(() => {
				toast.success(m.user_enabled_successfully());
				// Refresh the list
				userService.list(requestOptions!).then((updatedUsers) => (users = updatedUsers));
			})
			.catch(axiosErrorToast);
	}

	async function disableUser(user: User) {
		openConfirmDialog({
			title: m.disable_firstname_lastname({ firstName: user.firstName, lastName: user.lastName }),
			message: m.are_you_sure_you_want_to_disable_this_user(),
			confirm: {
				label: m.disable(),
				destructive: true,
				action: async () => {
					try {
						await userService.setUserAttribute(user.id, 'disabled', true);
						users = await userService.list(requestOptions!);
						toast.success(m.user_disabled_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<AdvancedTable
	items={users}
	{requestOptions}
	onRefresh={async (options) => (users = await userService.list(options))}
	columns={[
		{ label: m.first_name(), sortColumn: 'firstName' },
		{ label: m.last_name(), sortColumn: 'lastName' },
		{ label: m.email(), sortColumn: 'email' },
		{ label: m.username(), sortColumn: 'username' },
		{ label: m.role(), sortColumn: 'isAdmin' },
		{ label: m.status(), sortColumn: 'disabled' },
		...($appConfigStore.ldapEnabled ? [{ label: m.source() }] : []),
		{ label: m.actions(), hidden: true }
	]}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.firstName}</Table.Cell>
		<Table.Cell>{item.lastName}</Table.Cell>
		<Table.Cell>{item.email}</Table.Cell>
		<Table.Cell>{item.username}</Table.Cell>
		<Table.Cell>
			<Badge variant="outline">{item.isAdmin ? m.admin() : m.user()}</Badge>
		</Table.Cell>
		<Table.Cell>
			<Badge variant={item.disabled ? 'destructive' : 'default'}>
				{item.disabled ? m.disabled() : m.enabled()}
			</Badge>
		</Table.Cell>
		{#if $appConfigStore.ldapEnabled}
			<Table.Cell>
				<Badge variant={item.ldapId ? 'default' : 'outline'}
					>{item.ldapId ? m.ldap() : m.local()}</Badge
				>
			</Table.Cell>
		{/if}
		<Table.Cell>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger class={buttonVariants({ variant: 'ghost', size: 'icon' })}>
					<Ellipsis class="h-4 w-4" />
					<span class="sr-only">{m.toggle_menu()}</span>
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item onclick={() => (userIdToCreateOneTimeLink = item.id)}
						><LucideLink class="mr-2 h-4 w-4" />{m.login_code()}</DropdownMenu.Item
					>
					<DropdownMenu.Item onclick={() => goto(`/settings/admin/users/${item.id}`)}
						><LucidePencil class="mr-2 h-4 w-4" /> {m.edit()}</DropdownMenu.Item
					>
					{#if !item.ldapId || !$appConfigStore.ldapEnabled}
						{#if item.disabled}
							<DropdownMenu.Item onclick={() => enableUser(item)}
								><LucideUserCheck class="mr-2 h-4 w-4" />{m.enable()}</DropdownMenu.Item
							>
						{:else}
							<DropdownMenu.Item onclick={() => disableUser(item)}
								><LucideUserX class="mr-2 h-4 w-4" />{m.disable()}</DropdownMenu.Item
							>
						{/if}
					{/if}
					{#if !item.ldapId || (item.ldapId && item.disabled)}
						<DropdownMenu.Item
							class="text-red-500 focus:!text-red-700"
							onclick={() => deleteUser(item)}
							><LucideTrash class="mr-2 h-4 w-4" />{m.delete()}</DropdownMenu.Item
						>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</Table.Cell>
	{/snippet}
</AdvancedTable>

<OneTimeLinkModal userId={userIdToCreateOneTimeLink} />
