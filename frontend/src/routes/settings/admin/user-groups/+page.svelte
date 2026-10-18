<script lang="ts">
	import { goto } from '$app/navigation';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import EmptyList from '#lib/components/empty-list.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import type { UserGroupMinimal } from '#lib/types/user-group.type.ts';
	import { LucideArrowRight, LucidePlus, LucideTrash2 } from '@lucide/svelte';
	import { confirmDeleteGroup } from './user-group-actions';
	import UserGroupCreateSheet from './user-group-create-sheet.svelte';

	let { data } = $props();

	const userGroupService = new UserGroupService();
	let tableRef = $state<AdvancedTable<UserGroupMinimal>>();
	let createOpen = $state(false);
	let empty = $state(false);

	const href = (group: UserGroupMinimal) => `/settings/admin/user-groups/${group.id}`;
	const groupSlug = (group: UserGroupMinimal) =>
		group.name !== group.friendlyName.toLowerCase() ? group.name : undefined;

	async function fetchGroups(options: ListRequestOptions) {
		const res = await userGroupService.list(options);
		empty = res.pagination.totalItems === 0 && !options.search;
		return res;
	}

	const columns: AdvancedTableColumn<UserGroupMinimal>[] = [
		{ label: m.group(), column: 'friendlyName', sortable: true, cell: GroupCell },
		{ label: m.members(), column: 'userCount', sortable: true },
		{
			label: m.created(),
			column: 'createdAt',
			sortable: true,
			hideBelow: 520,
			value: (group) =>
				new Date(group.createdAt).toLocaleDateString(getLocale(), { dateStyle: 'medium' })
		},
		{ label: m.group_id(), key: 'id', hidden: true, cell: IdCell },
		{ label: m.ldap_id(), column: 'ldapId', hidden: true }
	];

	const actions: CreateAdvancedTableActions<UserGroupMinimal> = () => [
		{ label: m.open(), primary: true, icon: LucideArrowRight, onClick: (g) => goto(href(g)) },
		{
			label: m.delete_ellipsis(),
			icon: LucideTrash2,
			variant: 'danger',
			onClick: (g) =>
				confirmDeleteGroup({ ...g, memberCount: g.userCount }, () => tableRef?.refresh())
		}
	];
</script>

<svelte:head>
	<title>{m.groups()}</title>
</svelte:head>

{#snippet GroupCell({ item }: { item: UserGroupMinimal })}
	<div class="flex max-w-(--table-id-max) min-w-0 items-center gap-3">
		<OidcClientAvatar id={item.id} name={item.friendlyName} hasLogo={false} size={24} />
		<a
			href={href(item)}
			class="text-control text-fg focus-visible:outline-ring truncate rounded-xs font-medium focus-visible:outline-2 focus-visible:outline-offset-2"
			onclick={(e) => e.stopPropagation()}>{item.friendlyName}</a
		>
		{#if groupSlug(item)}
			<span
				class="text-mono-s text-fg-muted max-w-fit min-w-[6ch] flex-1 truncate font-mono max-sm:hidden"
				>{groupSlug(item)}</span
			>
		{/if}
		{#if item.ldapId && $appConfigStore.ldapEnabled}<Badge>{m.ldap()}</Badge>{/if}
	</div>
	<span class="text-caption text-fg-muted block truncate ps-9 sm:hidden">
		{item.userCount === 1 ? m.members_count_one() : m.members_count({ count: item.userCount })}
	</span>
{/snippet}

{#snippet IdCell({ item }: { item: UserGroupMinimal })}
	<span class="text-caption font-mono">{item.id}</span>
{/snippet}

<div class="flex flex-col">
	<PageHeader title={m.groups()} description={empty ? undefined : m.groups_page_description()}>
		{#snippet actions()}
			{#if !empty}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.add_group()}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if empty}
		<EmptyList columns={[m.group(), m.members()]} title={m.add_your_first_group()}>
			{#snippet description()}{m.groups_page_description()}{/snippet}
			{#snippet action()}
				<Button onclick={() => (createOpen = true)}>
					<LucidePlus />
					{m.add_group()}
				</Button>
			{/snippet}
		</EmptyList>
	{:else}
		<div class="mt-10">
			<AdvancedTable
				id="user-group-list"
				bind:this={tableRef}
				fetchCallback={fetchGroups}
				defaultSort={{ column: 'friendlyName', direction: 'asc' }}
				onRowClick={(group) => goto(href(group))}
				{columns}
				{actions}
			/>
		</div>
	{/if}
</div>

<UserGroupCreateSheet bind:open={createOpen} />
