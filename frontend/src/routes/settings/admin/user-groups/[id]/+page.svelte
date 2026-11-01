<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import FormInput from '#lib/components/form/form-input.svelte';
	import CustomClaimsInput from '#lib/components/form/custom-claims-input.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import * as InputGroup from '#lib/components/ui/input-group/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import CustomClaimService from '#lib/services/custom-claim-service.ts';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { OidcClientWithAllowedGroups } from '#lib/types/oidc.type.ts';
	import type { User } from '#lib/types/user.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges, trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import {
		LucideCopy,
		LucideEllipsis,
		LucideInfo,
		LucideLock,
		LucidePlus,
		LucideSearch,
		LucideTrash2,
		LucideUserMinus,
		LucideX
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { backNavigate } from '../../users/navigate-back-util';
	import UserAvatar from '../../users/user-avatar.svelte';
	import UserStatus from '../../users/user-status.svelte';
	import { groupNameSchema } from '../group-name-schema';
	import PickerDialog from '../picker-dialog.svelte';
	import { confirmDeleteGroup } from '../user-group-actions';

	let { data } = $props();
	let group = $state(data.userGroup);
	let tab = $state('general');
	let memberSearch = $state('');
	let memberSearchInput = $state<HTMLInputElement | null>(null);
	let memberPickerOpen = $state(false);
	let clientPickerOpen = $state(false);

	const userGroupService = new UserGroupService();
	const userService = new UserService();
	const customClaimService = new CustomClaimService();
	const backNavigation = backNavigate('/settings/admin/user-groups');

	const readOnly = $derived(!!group.ldapId && $appConfigStore.ldapEnabled);
	const openClients = $derived(data.clients.filter((c) => !c.isGroupRestricted));
	const memberCountLabel = $derived(
		group.users.length === 1
			? m.members_count_one()
			: m.members_count({ count: group.users.length })
	);
	const filteredMembers = $derived.by(() => {
		const query = memberSearch.trim().toLowerCase();
		return group.users
			.filter((u) => [u.displayName, u.username, u.email].join(' ').toLowerCase().includes(query))
			.toSorted((a, b) => a.displayName.localeCompare(b.displayName));
	});

	const formStore = createForm(groupNameSchema, {
		friendlyName: group.friendlyName,
		name: group.name
	});
	const { inputs } = formStore;
	trackFormChanges(
		() => formStore,
		async (values) => {
			const saved = await userGroupService.update(group.id, values);
			group.friendlyName = saved.friendlyName;
			group.name = saved.name;
		}
	);

	trackUnsavedValue(
		() => group.customClaims,
		(customClaims) => (group.customClaims = customClaims),
		(customClaims) => customClaimService.updateUserGroupCustomClaims(group.id, customClaims)
	);

	// Members and allowed clients save at once, and Undo restores the previous list
	async function setMembers(userIds: string[]) {
		group.users = (await userGroupService.updateUsers(group.id, userIds)).users;
	}

	async function setClients(clientIds: string[]) {
		group.allowedOidcClients = (
			await userGroupService.updateAllowedOidcClients(group.id, clientIds)
		).allowedOidcClients;
	}

	function undoable(message: string, undo: () => Promise<void>) {
		toast.success(message, {
			action: { label: m.undo(), onClick: () => undo().catch(axiosErrorToast) }
		});
	}

	async function addMembers(userIds: string[]) {
		await setMembers([...group.users.map((u) => u.id), ...userIds]);
		toast.success(m.members_added());
	}

	async function removeMember(user: User) {
		const before = group.users.map((u) => u.id);
		try {
			await setMembers(before.filter((id) => id !== user.id));
			undoable(
				m.name_removed_from_group({ name: user.displayName, group: group.friendlyName }),
				() => setMembers(before)
			);
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	async function allowClients(clientIds: string[]) {
		await setClients([...group.allowedOidcClients.map((c) => c.id), ...clientIds]);
		toast.success(m.allowed_clients_updated());
	}

	async function removeClient(client: { id: string; name: string }) {
		const before = group.allowedOidcClients.map((c) => c.id);
		const remove = async () => {
			await setClients(before.filter((id) => id !== client.id));
			undoable(
				m.group_removed_from_client({ group: group.friendlyName, client: client.name }),
				() => setClients(before)
			);
		};

		// A client limited to groups that loses its last one lets no one sign in
		const known = data.clients.find((c) => c.id === client.id);
		if (known && known.allowedUserGroups.every((g) => g.id === group.id)) {
			openConfirmDialog({
				title: m.remove_last_group_from_client_question({ client: client.name }),
				message: m.remove_last_group_from_client_consequence({ client: client.name }),
				confirm: {
					label: m.remove_group(),
					destructive: true,
					action: () => remove().catch(axiosErrorToast)
				}
			});
		} else {
			await remove().catch(axiosErrorToast);
		}
	}

	async function copyGroupId() {
		await navigator.clipboard.writeText(group.id);
		toast.success(m.group_id_copied());
	}

	// Long lists name the first few and count the rest, so the note stays short
	const joinNames = (names: string[]) =>
		names.length > 4
			? m.names_and_last({
					names: names.slice(0, 3).join(', '),
					last: m.count_more({ count: names.length - 3 })
				})
			: m.names_and_last({ names: names.slice(0, -1).join(', '), last: names.at(-1)! });
</script>

<svelte:head>
	<title>{group.friendlyName}</title>
</svelte:head>

{#snippet count(n: number)}
	<span class="text-fg-muted tabular-nums">{n}</span>
{/snippet}

{#snippet MemberRow(user: User)}
	<UserAvatar {user} class="size-6" />
	<span class="flex min-w-0 flex-1 flex-col">
		<span class="text-control text-fg truncate font-medium">{user.displayName}</span>
		{#if user.email}<span class="text-caption text-fg-muted truncate">{user.email}</span>{/if}
	</span>
	<UserStatus {user} />
{/snippet}

{#snippet ClientRow(client: OidcClientWithAllowedGroups)}
	<OidcClientAvatar
		id={client.id}
		name={client.name}
		hasLogo={client.hasLogo || client.hasDarkLogo}
		size={24}
	/>
	<span class="flex min-w-0 flex-1 flex-col">
		<span class="text-control text-fg truncate font-medium">{client.name}</span>
		{#if client.launchURL}
			<span class="text-caption text-fg-muted truncate">{client.launchURL}</span>
		{/if}
	</span>
{/snippet}

<div class="flex flex-col gap-8">
	<PageHeader
		title={group.friendlyName}
		breadcrumbs={[
			{ label: m.administration() },
			{ label: m.groups(), href: '/settings/admin/user-groups' }
		]}
	>
		{#snippet identity()}
			<OidcClientAvatar id={group.id} name={group.friendlyName} hasLogo={false} size={40} />
		{/snippet}
		{#snippet tags()}
			<span>{memberCountLabel}</span>
			{#if group.name !== group.friendlyName.toLowerCase()}
				<Badge variant="mono" class="max-w-full"><span class="truncate">{group.name}</span></Badge>
			{/if}
			{#if readOnly}<Badge>{m.ldap()}</Badge>{/if}
		{/snippet}
		{#snippet actions()}
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="outline" size="icon" aria-label={m.more_actions()}>
							<LucideEllipsis />
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item onclick={copyGroupId}>
						<LucideCopy />
						{m.copy_group_id()}
					</DropdownMenu.Item>
					{#if !readOnly}
						<DropdownMenu.Separator />
						<DropdownMenu.Item
							variant="destructive"
							onclick={() =>
								confirmDeleteGroup(
									{ ...group, memberCount: group.users.length },
									backNavigation.leave
								)}
						>
							<LucideTrash2 />
							{m.delete_group_ellipsis()}
						</DropdownMenu.Item>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{/snippet}
	</PageHeader>

	<Tabs.Root bind:value={tab} useHash class="gap-10">
		<div class="flex shadow-[inset_0_-1px_0_var(--border)]">
			<div class="scroll-fade-x min-w-0 [scrollbar-width:none] overflow-x-auto">
				<Tabs.List variant="line" class="shadow-none max-sm:gap-4" aria-label={m.group_sections()}>
					<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
					<Tabs.Trigger value="members"
						>{m.members()} {@render count(group.users.length)}</Tabs.Trigger
					>
					<Tabs.Trigger value="clients">
						{m.allowed_clients()}
						{@render count(group.allowedOidcClients.length)}
					</Tabs.Trigger>
					{#if data.features?.customClaims.enabled}
						<Tabs.Trigger value="custom-claims">
							{m.custom_claims()}
							{@render count(group.customClaims.length)}
						</Tabs.Trigger>
					{/if}
				</Tabs.List>
			</div>
		</div>

		<Tabs.Content value="general">
			<SettingsSection title={m.names()}>
				<div class="flex max-w-100 flex-col gap-6 pt-4">
					{#if readOnly}
						<p class="text-caption text-fg-muted flex items-start gap-1.5">
							<LucideLock class="mt-1 size-3 shrink-0" />
							{m.profile_managed_by_ldap()}
						</p>
					{/if}
					<FormInput
						label={m.friendly_name()}
						bind:input={$inputs.friendlyName}
						readonly={readOnly}
					/>
					<FormInput
						label={m.name()}
						description={readOnly ? undefined : m.group_name_change_helper()}
						inputClass="font-mono"
						bind:input={$inputs.name}
						readonly={readOnly}
					/>
				</div>
			</SettingsSection>
		</Tabs.Content>

		<Tabs.Content value="members">
			<SettingsSection title={m.members()}>
				{#snippet action()}
					{#if !readOnly}
						<Button variant="secondary" size="sm" onclick={() => (memberPickerOpen = true)}>
							<LucidePlus />
							{m.add_members_ellipsis()}
						</Button>
					{/if}
				{/snippet}
				{#if readOnly}
					<p class="text-caption text-fg-muted mb-2 flex items-start gap-1.5">
						<LucideLock class="mt-1 size-3 shrink-0" />
						{m.group_members_come_from_ldap()}
					</p>
				{/if}
				{#if group.users.length > 5}
					<InputGroup.Root class="mb-3 w-full sm:w-60">
						<InputGroup.Input
							bind:ref={memberSearchInput}
							bind:value={memberSearch}
							placeholder={m.search_members()}
							aria-label={m.search_members()}
							type="search"
						/>
						<InputGroup.Addon><LucideSearch /></InputGroup.Addon>
					</InputGroup.Root>
				{/if}
				<ul class="border-t">
					{#each filteredMembers as user (user.id)}
						<li class="flex min-h-(--row-2line) items-center gap-3 border-b py-2 ps-4 pe-2">
							<UserAvatar {user} class="size-6" />
							<span class="flex min-w-0 flex-1 flex-col">
								<a
									href="/settings/admin/users/{user.id}"
									class="text-control text-fg focus-visible:outline-ring truncate rounded-xs font-medium hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
									>{user.displayName}</a
								>
								{#if user.email}
									<span class="text-caption text-fg-muted truncate">{user.email}</span>
								{/if}
							</span>
							<UserStatus {user} />
							{#if !readOnly}
								<Button
									variant="ghost"
									size="icon-xs"
									aria-label={m.remove_name_from_group({
										name: user.displayName,
										group: group.friendlyName
									})}
									onclick={() => removeMember(user)}
								>
									<LucideUserMinus />
								</Button>
							{/if}
						</li>
					{:else}
						<li
							class="text-control text-fg-muted flex min-h-(--row-2line) flex-wrap items-center gap-x-3 border-b px-4 wrap-anywhere"
						>
							{#if memberSearch.trim()}
								{m.no_matches_for_query({ query: memberSearch.trim() })}
								<Button
									variant="link"
									onclick={() => {
										memberSearch = '';
										memberSearchInput?.focus();
									}}>{m.clear_search()}</Button
								>
							{:else}
								{m.no_members_yet()}
							{/if}
						</li>
					{/each}
				</ul>
			</SettingsSection>
		</Tabs.Content>

		<Tabs.Content value="clients">
			<SettingsSection
				title={m.allowed_clients()}
				description={m.allowed_clients_section_description({ group: group.friendlyName })}
			>
				{#snippet action()}
					<Button variant="secondary" size="sm" onclick={() => (clientPickerOpen = true)}>
						<LucidePlus />
						{m.allow_clients_ellipsis()}
					</Button>
				{/snippet}
				<ul class="border-t">
					{#each group.allowedOidcClients as client (client.id)}
						<li class="flex min-h-(--row-2line) items-center gap-3 border-b py-2 ps-4 pe-2">
							<OidcClientAvatar
								id={client.id}
								name={client.name}
								hasLogo={client.hasLogo || client.hasDarkLogo}
								size={24}
							/>
							<span class="flex min-w-0 flex-1 flex-col">
								<a
									href="/settings/admin/oidc-clients/{encodeClientIdParam(client.id)}"
									class="text-control text-fg focus-visible:outline-ring truncate rounded-xs font-medium hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
									>{client.name}</a
								>
								{#if client.launchURL}
									<span class="text-caption text-fg-muted truncate">{client.launchURL}</span>
								{/if}
							</span>
							<Button
								variant="ghost"
								size="icon-xs"
								aria-label={m.remove_group_from_client({
									group: group.friendlyName,
									client: client.name
								})}
								onclick={() => removeClient(client)}
							>
								<LucideX />
							</Button>
						</li>
					{:else}
						<li
							class="text-control text-fg-muted flex min-h-(--row-2line) items-center border-b px-4"
						>
							{m.no_clients_limited_to_this_group()}
						</li>
					{/each}
				</ul>
				{#if openClients.length}
					<p class="text-caption text-fg-muted mt-3 flex items-start gap-1.5">
						<LucideInfo class="mt-[3px] size-3 shrink-0" />
						{openClients.length === 1
							? m.client_open_to_everyone({ client: openClients[0].name })
							: m.clients_open_to_everyone({ clients: joinNames(openClients.map((c) => c.name)) })}
					</p>
				{/if}
			</SettingsSection>
		</Tabs.Content>

		<Tabs.Content value="custom-claims">
			<SettingsSection title={m.custom_claims()} description={m.group_custom_claims_description()}>
				<CustomClaimsInput bind:customClaims={group.customClaims} />
			</SettingsSection>
		</Tabs.Content>
	</Tabs.Root>
</div>

<PickerDialog
	bind:open={memberPickerOpen}
	title={m.add_members_to_group({ group: group.friendlyName })}
	searchLabel={m.search_people()}
	load={(search) =>
		userService
			.list({
				search,
				pagination: { page: 1, limit: 50 },
				sort: { column: 'displayName', direction: 'asc' }
			})
			.then((res) => res.data)}
	exclude={group.users.map((u) => u.id)}
	row={MemberRow}
	confirmLabel={(n) =>
		n === 0 ? m.add_members() : n === 1 ? m.add_one_member() : m.add_count_members({ count: n })}
	onConfirm={addMembers}
/>
<PickerDialog
	bind:open={clientPickerOpen}
	title={m.allow_clients_for_group({ group: group.friendlyName })}
	searchLabel={m.search_clients()}
	load={async (search) => {
		const query = search.toLowerCase();
		return data.clients.filter((c) => c.isGroupRestricted && c.name.toLowerCase().includes(query));
	}}
	exclude={group.allowedOidcClients.map((c) => c.id)}
	row={ClientRow}
	confirmLabel={(n) =>
		n === 0
			? m.allow_clients()
			: n === 1
				? m.allow_one_client()
				: m.allow_count_clients({ count: n })}
	onConfirm={allowClients}
/>
