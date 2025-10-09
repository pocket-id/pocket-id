<script lang="ts">
	import { goto } from '$app/navigation';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import OneTimeLinkModal from '$lib/components/one-time-link-modal.svelte';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import * as Avatar from '$lib/components/ui/avatar/index';
	import { Badge } from '$lib/components/ui/badge/index';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { User } from '$lib/types/user.type';
	import { cachedProfilePicture } from '$lib/utils/cached-image-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import {
		LucideLink,
		LucidePencil,
		LucideTrash,
		LucideUserCheck,
		LucideUserX
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let userIdToCreateOneTimeLink: string | null = $state(null);
	let tableRef: AdvancedTable<User>;

	const userService = new UserService();

	export const refresh = () => tableRef.refresh();

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
						await refresh();
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
			.update(user.id, {
				...user,
				disabled: false
			})
			.then(async () => {
				toast.success(m.user_enabled_successfully());
				await refresh();
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
						await userService.update(user.id, {
							...user,
							disabled: true
						});
						await refresh();
						toast.success(m.user_disabled_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	const columns: AdvancedTableColumn<User>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.profile_picture(), key: 'profilePicture', cell: ProfilePictureCell },
		{ label: m.first_name(), column: 'firstName', sortable: true },
		{ label: m.last_name(), column: 'lastName', sortable: true },
		{ label: m.display_name(), column: 'displayName', sortable: true },
		{ label: m.email(), column: 'email', sortable: true },
		{ label: m.username(), column: 'username', sortable: true },
		{
			label: m.role(),
			column: 'isAdmin',
			sortable: true,
			filterableValues: [
				{ label: m.admin(), value: true },
				{ label: m.user(), value: false }
			],
			value: (item) => (item.isAdmin ? m.admin() : m.user())
		},
		{
			label: m.status(),
			column: 'disabled',
			cell: StatusCell,
			sortable: true,
			filterableValues: [
				{
					label: m.enabled(),
					value: false
				},
				{
					label: m.disabled(),
					value: true
				}
			]
		},
		{ label: m.ldap_id(), column: 'ldapId', hidden: true },
		{ label: m.locale(), column: 'locale', hidden: true },
		{ label: m.source(), key: 'source', hidden: !$appConfigStore.ldapEnabled, cell: SourceCell }
	];

	const actions: CreateAdvancedTableActions<User> = (u) => [
		{
			label: m.login_code(),
			icon: LucideLink,
			onClick: (u) => (userIdToCreateOneTimeLink = u.id)
		},
		{
			label: m.edit(),
			icon: LucidePencil,
			onClick: (u) => goto(`/settings/admin/users/${u.id}`)
		},
		{
			label: u.disabled ? m.enable() : m.disable(),
			icon: u.disabled ? LucideUserCheck : LucideUserX,
			onClick: (u) => (u.disabled ? enableUser(u) : disableUser(u)),
			visible: !u.ldapId || !$appConfigStore.ldapEnabled,
			disabled: u.id === $userStore?.id
		},
		{
			label: m.delete(),
			icon: LucideTrash,
			variant: 'danger',
			onClick: (u) => deleteUser(u),
			visible: !!(!u.ldapId || (u.ldapId && u.disabled)),
			disabled: u.id === $userStore?.id
		}
	];
</script>

{#snippet ProfilePictureCell({ item }: { item: User })}
	<Avatar.Root class="size-8">
		<Avatar.Image class="object-cover" src={cachedProfilePicture.getUrl(item.id)} />
	</Avatar.Root>
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

<AdvancedTable
	id="user-list"
	bind:this={tableRef}
	fetchCallback={userService.list}
	{actions}
	{columns}
/>

<OneTimeLinkModal bind:userId={userIdToCreateOneTimeLink} />
