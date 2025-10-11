<script lang="ts">
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import * as Avatar from '$lib/components/ui/avatar/index';
	import { Badge } from '$lib/components/ui/badge';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { User } from '$lib/types/user.type';
	import { cachedProfilePicture } from '$lib/utils/cached-image-util';

	let {
		selectionDisabled = false,
		selectedUserIds = $bindable()
	}: {
		selectionDisabled?: boolean;
		selectedUserIds: string[];
	} = $props();

	const userService = new UserService();

	const columns: AdvancedTableColumn<User>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.profile_picture(), key: 'profilePicture', cell: ProfilePictureCell },
		{ label: m.first_name(), column: 'firstName', sortable: true, hidden: true },
		{ label: m.last_name(), column: 'lastName', sortable: true, hidden: true },
		{ label: m.display_name(), column: 'displayName', sortable: true },
		{ label: m.email(), column: 'email', sortable: true, hidden: true },
		{ label: m.username(), column: 'username', sortable: true },
		{
			label: m.role(),
			column: 'isAdmin',
			sortable: true,
			filterableValues: [
				{ label: m.admin(), value: true },
				{ label: m.user(), value: false }
			],
			value: (item) => (item.isAdmin ? m.admin() : m.user()),
			hidden: true
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
		{ label: m.locale(), column: 'locale', hidden: true }
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

<AdvancedTable
	id="user-selection"
	fetchCallback={userService.list}
	defaultSort={{ column: 'firstName', direction: 'asc' }}
	bind:selectedIds={selectedUserIds}
	{selectionDisabled}
	{columns}
/>
