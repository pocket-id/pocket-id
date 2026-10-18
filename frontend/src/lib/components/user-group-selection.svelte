<script lang="ts">
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { UserGroupMinimal } from '#lib/types/user-group.type.ts';

	let { selectedGroupIds = $bindable() }: { selectedGroupIds: string[] } = $props();

	const userGroupService = new UserGroupService();

	const columns: AdvancedTableColumn<UserGroupMinimal>[] = [
		{ label: m.group(), column: 'friendlyName', sortable: true, cell: GroupCell },
		{ label: m.members(), column: 'userCount', sortable: true },
		{ label: m.name(), column: 'name', sortable: true, hidden: true }
	];
</script>

{#snippet GroupCell({ item }: { item: UserGroupMinimal })}
	<div class="flex max-w-(--table-id-max) min-w-0 items-center gap-3">
		<OidcClientAvatar id={item.id} name={item.friendlyName} hasLogo={false} size={24} />
		<span class="text-control text-fg truncate font-medium">{item.friendlyName}</span>
	</div>
{/snippet}

<AdvancedTable
	id="user-group-selection"
	fetchCallback={userGroupService.list}
	defaultSort={{ column: 'friendlyName', direction: 'asc' }}
	bind:selectedIds={selectedGroupIds}
	{columns}
/>
