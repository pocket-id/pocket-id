<script lang="ts">
	import { goto } from '$app/navigation';
	import PageHeader from '#lib/components/page-header.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as InputGroup from '#lib/components/ui/input-group/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import type { User } from '#lib/types/user.type.ts';
	import type { UserGroupMinimal } from '#lib/types/user-group.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { revealFocused } from '#lib/utils/event-util.ts';
	import { cn } from '#lib/utils/style.ts';
	import { toast } from 'svelte-sonner';
	import {
		LucideArrowRight,
		LucideKeyRound,
		LucidePlus,
		LucideSearch,
		LucideTrash2,
		LucideUserCheck,
		LucideUserX,
		LucideUsersRound
	} from '@lucide/svelte';
	import { SvelteMap } from 'svelte/reactivity';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import { openOnCreateParam } from '#lib/utils/create-param-util.ts';
	import PickerDialog, { groupsConfirmLabel } from '../user-groups/picker-dialog.svelte';
	import type { PageProps } from './$types';
	import LoginCodeDialog from './login-code-dialog.svelte';
	import SignupTokensSheet from './signup-tokens-sheet.svelte';
	import UserAvatar from './user-avatar.svelte';
	import UserCreateSheet from './user-create-sheet.svelte';
	import UserStatus from './user-status.svelte';
	import { confirmDeleteUsers, confirmDisableUsers, enableUsers } from './user-actions';
	import { countUsersByScope, scopeFilters, type Scope } from './user-scopes';

	let { data }: PageProps = $props();

	const userService = new UserService();
	const userGroupService = new UserGroupService();

	let tableRef: AdvancedTable<User>;
	let scope = $state<Scope>('all');
	let search = $state('');
	let searchInput = $state<HTMLInputElement | null>(null);
	let counts = $state(data.counts);
	let shownCount = $state(0);
	let selectedIds = $state<string[]>([]);
	let createOpen = $state(false);
	openOnCreateParam(() => (createOpen = true));
	let tokensOpen = $state(false);
	let groupPickerOpen = $state(false);
	let loginCodeUser = $state<User | null>(null);

	// Rows of every page seen so far, so a selection that spans pages can still be acted on
	const loadedUsers = new SvelteMap<string, User>();
	const selectedUsers = $derived(selectedIds.map((id) => loadedUsers.get(id)).filter((u) => !!u));

	const signupsOn = $derived($appConfigStore.allowUserSignups !== 'disabled');
	const verificationOn = $derived($appConfigStore.emailVerification !== 'disabled');
	const ldapManaged = (user: User) => !!user.ldapId && $appConfigStore.ldapEnabled;
	const isMe = (user: User) => user.id === $userStore?.id;

	const scopes = $derived(
		(
			[
				['all', m.all()],
				['admins', m.admins()],
				['disabled', m.disabled()],
				['unverified', m.unverified()]
			] as const
		).filter(([value]) => value !== 'unverified' || verificationOn)
	);

	async function fetchUsers(options: ListRequestOptions) {
		const res = await userService.list({
			...options,
			search: search.trim() || undefined,
			filters: { ...options.filters, ...scopeFilters[scope] }
		});
		for (const user of res.data) loadedUsers.set(user.id, user);
		shownCount = res.pagination.totalItems;
		return res;
	}

	async function refresh() {
		[, counts] = await Promise.all([tableRef?.refresh(), countUsersByScope()]);
	}

	async function afterBulk() {
		selectedIds = [];
		await refresh();
	}

	const onSearch = debounced(() => tableRef?.refresh(), 300);

	const countLine = $derived.by(() => {
		if (scope === 'all' && !search.trim()) {
			return counts.all === 1 ? m.users_count_one() : m.users_count({ count: counts.all });
		}
		return m.count_of_users({ count: shownCount, total: counts.all });
	});

	// Phones hide every column but the first, so its third line carries what they would have shown
	function phoneMeta(user: User) {
		const status = user.disabled
			? m.disabled()
			: !user.emailVerified && verificationOn
				? m.unverified()
				: undefined;
		return [status, user.username, user.isAdmin ? m.admin() : undefined]
			.filter(Boolean)
			.join(' · ');
	}

	const columns: AdvancedTableColumn<User>[] = [
		{ label: m.user(), column: 'displayName', sortable: true, cell: UserCell },
		{ label: m.username(), column: 'username', sortable: true, hideBelow: 900 },
		{ label: m.role(), column: 'isAdmin', sortable: true, cell: RoleCell, hideBelow: 520 },
		{ label: m.status(), key: 'status', cell: StatusCell },
		{ label: m.first_name(), column: 'firstName', sortable: true, hidden: true },
		{ label: m.last_name(), column: 'lastName', sortable: true, hidden: true },
		{ label: m.locale(), column: 'locale', hidden: true, cell: LocaleCell },
		{ label: m.user_id(), key: 'id', hidden: true, cell: IdCell },
		{ label: m.ldap_id(), key: 'ldapId', hidden: true, cell: LdapIdCell }
	];

	const actions: CreateAdvancedTableActions<User> = (user) => [
		{
			label: m.open(),
			primary: true,
			icon: LucideArrowRight,
			onClick: (u) => goto(`/settings/admin/users/${u.id}`)
		},
		{ label: m.login_code_ellipsis(), icon: LucideKeyRound, onClick: (u) => (loginCodeUser = u) },
		{
			label: user.disabled ? m.enable() : m.disable_ellipsis(),
			icon: user.disabled ? LucideUserCheck : LucideUserX,
			hidden: ldapManaged(user),
			disabled: isMe(user),
			onClick: (u) => (u.disabled ? enableUsers([u], refresh) : confirmDisableUsers([u], refresh))
		},
		{
			label: m.delete_ellipsis(),
			icon: LucideTrash2,
			variant: 'danger',
			hidden: ldapManaged(user) && !user.disabled,
			disabled: isMe(user),
			onClick: (u) => confirmDeleteUsers([u], refresh)
		}
	];

	// Your own account and people managed by LDAP are skipped by the actions that can't apply to them
	const bulkEditable = $derived(selectedUsers.filter((u) => !isMe(u) && !ldapManaged(u)));
	const bulkDeletable = $derived(
		selectedUsers.filter((u) => !isMe(u) && !(ldapManaged(u) && !u.disabled))
	);

	async function addSelectedToGroups(groupIds: string[]) {
		const userIds = selectedUsers.map((u) => u.id);
		await Promise.all(
			groupIds.map(async (id) => {
				const group = await userGroupService.get(id);
				const memberIds = group.users.map((u) => u.id);
				await userGroupService.updateUsers(id, [...new Set([...memberIds, ...userIds])]);
			})
		);
		toast.success(m.group_memberships_updated());
		await afterBulk();
	}
</script>

<svelte:head>
	<title>{m.users()}</title>
</svelte:head>

{#snippet UserCell({ item }: { item: User })}
	<div class="flex max-w-(--table-id-max) min-w-0 items-center gap-3">
		<UserAvatar user={item} />
		<div class="flex min-w-0 flex-col">
			<span class="flex min-w-0 items-center gap-2">
				<a
					href="/settings/admin/users/{item.id}"
					class="text-control text-fg focus-visible:outline-ring truncate rounded-xs font-medium focus-visible:outline-2 focus-visible:outline-offset-2"
					onclick={(e) => e.stopPropagation()}>{item.displayName}</a
				>
				{#if isMe(item)}<Badge>{m.you()}</Badge>{/if}
				{#if ldapManaged(item)}<Badge>{m.ldap()}</Badge>{/if}
			</span>
			{#if item.email}
				<span class="text-caption text-fg-muted truncate">{item.email}</span>
			{/if}
			<span class="text-caption text-fg-muted truncate sm:hidden">{phoneMeta(item)}</span>
		</div>
	</div>
{/snippet}

{#snippet RoleCell({ item }: { item: User })}
	{#if item.isAdmin}{m.admin()}{:else}<span class="text-fg-muted">{m.user()}</span>{/if}
{/snippet}

{#snippet StatusCell({ item }: { item: User })}
	<UserStatus user={item} />
{/snippet}

{#snippet IdCell({ item }: { item: User })}
	<span class="text-caption font-mono">{item.id}</span>
{/snippet}

{#snippet LocaleCell({ item }: { item: User })}
	{#if item.locale}{item.locale}{:else}<span class="text-fg-muted">{m.none()}</span>{/if}
{/snippet}

{#snippet LdapIdCell({ item }: { item: User })}
	{#if item.ldapId}
		<span class="text-caption font-mono">{item.ldapId}</span>
	{:else}
		<span class="text-fg-muted">{m.none()}</span>
	{/if}
{/snippet}

{#snippet GroupRow(group: UserGroupMinimal)}
	<OidcClientAvatar id={group.id} name={group.friendlyName} hasLogo={false} size={24} />
	<span class="text-control text-fg min-w-0 flex-1 truncate font-medium">{group.friendlyName}</span>
	<span class="text-caption text-fg-muted">
		{group.userCount === 1 ? m.members_count_one() : m.members_count({ count: group.userCount })}
	</span>
{/snippet}

<div class="flex flex-col">
	<PageHeader
		title={m.users()}
		description={m.users_page_description({ appName: $appConfigStore.appName })}
	>
		{#snippet actions()}
			{#if signupsOn}
				<Button variant="secondary" onclick={() => (tokensOpen = true)}>
					{m.signup_tokens()}
				</Button>
			{/if}
			<Button onclick={() => (createOpen = true)}>
				<LucidePlus />
				{m.add_user()}
			</Button>
		{/snippet}
	</PageHeader>

	<!-- On phones the selection bar sticks below the 52px top bar, above the table's sticky action cells -->
	<div
		class={cn(
			'mt-10 flex min-h-(--h-md) flex-wrap items-center gap-x-3 gap-y-2',
			selectedIds.length > 0 &&
				'max-sm:bg-surface max-sm:sticky max-sm:top-[52px] max-sm:z-(--z-nav) max-sm:py-2'
		)}
	>
		{#if selectedIds.length > 0}
			<span class="text-control text-fg font-medium tabular-nums" aria-live="polite">
				{m.count_selected({ count: selectedIds.length })}
			</span>
			<Button variant="ghost" size="sm" class="text-fg-muted" onclick={() => (selectedIds = [])}>
				{m.clear_selection()}
			</Button>
			<Button variant="secondary" size="sm" onclick={() => (groupPickerOpen = true)}>
				<LucideUsersRound />
				{m.add_to_groups_ellipsis()}
			</Button>
			{#if bulkEditable.some((u) => !u.disabled)}
				<Button
					variant="secondary"
					size="sm"
					onclick={() =>
						confirmDisableUsers(
							bulkEditable.filter((u) => !u.disabled),
							afterBulk
						)}
				>
					<LucideUserX />
					{m.disable_ellipsis()}
				</Button>
			{/if}
			{#if bulkEditable.some((u) => u.disabled)}
				<Button
					variant="secondary"
					size="sm"
					onclick={() =>
						enableUsers(
							bulkEditable.filter((u) => u.disabled),
							afterBulk
						)}
				>
					<LucideUserCheck />
					{m.enable()}
				</Button>
			{/if}
			{#if bulkDeletable.length > 0}
				<Button
					variant="secondary"
					size="sm"
					onclick={() => confirmDeleteUsers(bulkDeletable, afterBulk)}
				>
					<LucideTrash2 />
					{m.delete_ellipsis()}
				</Button>
			{/if}
		{:else}
			<InputGroup.Root class="w-full sm:w-75">
				<InputGroup.Input
					bind:ref={searchInput}
					bind:value={search}
					placeholder={m.search_users()}
					aria-label={m.search_users()}
					type="search"
					oninput={() => onSearch()}
				/>
				<InputGroup.Addon><LucideSearch /></InputGroup.Addon>
			</InputGroup.Root>
			<!-- The padding keeps focus rings visible inside the horizontal phone scroller -->
			<div
				class="scroll-fade-x -m-1 max-w-full scroll-px-1 [scrollbar-width:none] overflow-x-auto p-1"
				onfocusin={revealFocused}
			>
				<Tabs.Root
					bind:value={scope}
					onValueChange={() => {
						selectedIds = [];
						tableRef?.refresh();
					}}
				>
					<Tabs.List aria-label={m.user_scopes()}>
						{#each scopes as [value, label] (value)}
							<Tabs.Trigger {value}>
								{label}
								<span class="text-fg-muted tabular-nums">{counts[value]}</span>
							</Tabs.Trigger>
						{/each}
					</Tabs.List>
				</Tabs.Root>
			</div>
		{/if}
	</div>

	<p class="text-caption text-fg-muted mt-4 mb-3 min-h-[18px] tabular-nums" aria-live="polite">
		{countLine}
	</p>

	<AdvancedTable
		id="user-list"
		bind:this={tableRef}
		bind:selectedIds
		withoutSearch
		query={search}
		filtered={scope !== 'all'}
		onClearQuery={() => {
			search = '';
			scope = 'all';
			searchInput?.focus();
			tableRef?.refresh();
		}}
		fetchCallback={fetchUsers}
		defaultSort={{ column: 'displayName', direction: 'asc' }}
		onRowClick={(user) => goto(`/settings/admin/users/${user.id}`)}
		{columns}
		{actions}
	/>
</div>

<UserCreateSheet bind:open={createOpen} emailsVerifiedPerDefault={data.emailsVerifiedPerDefault} />
<SignupTokensSheet bind:open={tokensOpen} />
<LoginCodeDialog bind:user={loginCodeUser} />
<PickerDialog
	bind:open={groupPickerOpen}
	title={selectedUsers.length === 1
		? m.add_name_to_groups({ name: selectedUsers[0].firstName || selectedUsers[0].displayName })
		: m.add_count_users_to_groups({ count: selectedUsers.length })}
	searchLabel={m.search_groups()}
	load={(search) =>
		userGroupService
			.list({
				search,
				pagination: { page: 1, limit: 50 },
				sort: { column: 'friendlyName', direction: 'asc' }
			})
			.then((res) => res.data)}
	row={GroupRow}
	confirmLabel={groupsConfirmLabel}
	onConfirm={addSelectedToGroups}
/>
