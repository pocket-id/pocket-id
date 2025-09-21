<script lang="ts">
	import { goto } from '$app/navigation';
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { buttonVariants } from '$lib/components/ui/button';
	import OneTimeLinkModal from '$lib/components/one-time-link-modal.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { User } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { toast } from 'svelte-sonner';
	import {
		LucideLink,
		LucidePencil,
		LucideTrash,
		LucideUserCheck,
		LucideUserX
	} from '@lucide/svelte';
	import Ellipsis from '@lucide/svelte/icons/ellipsis';
	import type { ColumnSpec } from '$lib/components/pocket-id-table/pocket-id-table.types.svelte';

	let {
		users = $bindable(),
		requestOptions
	}: { users: Paginated<User>; requestOptions: SearchPaginationSortRequest } = $props();

	let userIdToCreateOneTimeLink: string | null = $state(null);
	const userService = new UserService();

	async function deleteUser(user: User) {
		openConfirmDialog({
			title: m.delete_firstname_lastname({
				firstName: user.firstName,
				lastName: user.lastName ?? ''
			}),
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
			.update(user.id, { ...user, disabled: false })
			.then(() => {
				toast.success(m.user_enabled_successfully());
				userService.list(requestOptions!).then((updatedUsers) => (users = updatedUsers));
			})
			.catch(axiosErrorToast);
	}

	async function disableUser(user: User) {
		openConfirmDialog({
			title: m.disable_firstname_lastname({
				firstName: user.firstName,
				lastName: user.lastName ?? ''
			}),
			message: m.are_you_sure_you_want_to_disable_this_user(),
			confirm: {
				label: m.disable(),
				destructive: true,
				action: async () => {
					try {
						await userService.update(user.id, { ...user, disabled: true });
						users = await userService.list(requestOptions!);
						toast.success(m.user_disabled_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	const columns = [
		{ accessorKey: 'firstName', title: m.first_name(), sortable: true },
		{ accessorKey: 'lastName', title: m.last_name(), sortable: true },
		{ accessorKey: 'displayName', title: m.display_name(), sortable: true },
		{ accessorKey: 'email', title: m.email(), sortable: true },
		{ accessorKey: 'username', title: m.username(), sortable: true },
		{
			accessorKey: 'isAdmin',
			title: m.role(),
			sortable: true,
			cell: RoleCell,
			filterFn: (row, columnId, filterValue) => {
				const selected = Array.isArray(filterValue) ? (filterValue as boolean[]) : [];
				if (selected.length === 0) return true;
				const value = Boolean(row.getValue<boolean>(columnId));
				return selected.includes(value);
			}
		},
		{
			accessorKey: 'disabled',
			title: m.status(),
			sortable: true,
			cell: StatusCell,
			filterFn: (row, columnId, filterValue) => {
				const selected = Array.isArray(filterValue) ? (filterValue as boolean[]) : [];
				if (selected.length === 0) return true;
				const value = Boolean(row.getValue<boolean>(columnId));
				return selected.includes(value);
			}
		},
		...($appConfigStore.ldapEnabled
			? ([
					{
						accessorKey: 'ldapId' as const,
						title: m.source(),
						cell: SourceCell
					}
				] as ColumnSpec<User>[])
			: [])
	] satisfies ColumnSpec<User>[];
</script>

{#snippet RoleCell({ item }: { item: User })}
	<Badge class="rounded-full" variant="outline">{item.isAdmin ? m.admin() : m.user()}</Badge>
{/snippet}

{#snippet StatusCell({ item }: { item: User })}
	<Badge class="rounded-full" variant={item.disabled ? 'destructive' : 'default'}>
		{item.disabled ? m.disabled() : m.enabled()}
	</Badge>
{/snippet}

{#snippet SourceCell({ item }: { item: User })}
	<Badge class="rounded-full" variant={item.ldapId ? 'default' : 'outline'}>
		{item.ldapId ? m.ldap() : m.local()}
	</Badge>
{/snippet}

{#snippet RowActions({ item }: { item: User })}
	<DropdownMenu.Root>
		<DropdownMenu.Trigger class={buttonVariants({ variant: 'ghost', size: 'icon' })}>
			<Ellipsis class="size-4" />
			<span class="sr-only">{m.toggle_menu()}</span>
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end">
			<DropdownMenu.Item onclick={() => (userIdToCreateOneTimeLink = item.id)}>
				<LucideLink class="mr-2 size-4" />{m.login_code()}
			</DropdownMenu.Item>
			<DropdownMenu.Item onclick={() => goto(`/settings/admin/users/${item.id}`)}>
				<LucidePencil class="mr-2 size-4" />
				{m.edit()}
			</DropdownMenu.Item>
			{#if !item.ldapId || !$appConfigStore.ldapEnabled}
				{#if item.disabled}
					<DropdownMenu.Item disabled={item.id === $userStore?.id} onclick={() => enableUser(item)}>
						<LucideUserCheck class="mr-2 size-4" />{m.enable()}
					</DropdownMenu.Item>
				{:else}
					<DropdownMenu.Item
						disabled={item.id === $userStore?.id}
						onclick={() => disableUser(item)}
					>
						<LucideUserX class="mr-2 size-4" />{m.disable()}
					</DropdownMenu.Item>
				{/if}
			{/if}
			{#if !item.ldapId || (item.ldapId && item.disabled)}
				<DropdownMenu.Item
					class="text-red-500 focus:!text-red-700"
					disabled={item.id === $userStore?.id}
					onclick={() => deleteUser(item)}
				>
					<LucideTrash class="mr-2 size-4" />{m.delete()}
				</DropdownMenu.Item>
			{/if}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/snippet}

<PocketIdTable
	items={users}
	bind:requestOptions
	onRefresh={async (opts) => (users = await userService.list(opts))}
	{columns}
	persistKey="pocket-id-users-table"
	selectionDisabled={true}
	rowActions={RowActions}
/>

<OneTimeLinkModal bind:userId={userIdToCreateOneTimeLink} />
