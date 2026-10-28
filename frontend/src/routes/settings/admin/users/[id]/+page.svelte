<script lang="ts">
	import CustomClaimsInput from '#lib/components/form/custom-claims-input.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import CustomClaimService from '#lib/services/custom-claim-service.ts';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { UserGroup, UserGroupMinimal } from '#lib/types/user-group.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import {
		LucideCopy,
		LucideEllipsis,
		LucideKeyRound,
		LucideLock,
		LucidePlus,
		LucideTrash2,
		LucideUserCheck,
		LucideUserMinus,
		LucideUserX
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import PickerDialog, { groupsConfirmLabel } from '../../user-groups/picker-dialog.svelte';
	import LoginCodeDialog from '../login-code-dialog.svelte';
	import { backNavigate } from '../navigate-back-util';
	import UserAvatar from '../user-avatar.svelte';
	import { confirmDeleteUsers, confirmDisableUsers, enableUsers } from '../user-actions';
	import UserForm from '../user-form.svelte';
	import UserStatus from '../user-status.svelte';
	import AdminPasskeyList from './admin-passkey-list.svelte';

	let { data } = $props();
	let user = $state(data.user);
	let passkeys = $state(data.passkeys);
	let tab = $state('general');
	let loginCodeUser = $state<typeof user | null>(null);
	let groupPickerOpen = $state(false);

	const userService = new UserService();
	const userGroupService = new UserGroupService();
	const customClaimService = new CustomClaimService();
	const backNavigation = backNavigate('/settings/admin/users');

	const isMe = $derived(user.id === $userStore?.id);
	const readOnly = $derived(!!user.ldapId && $appConfigStore.ldapEnabled);
	const firstName = $derived(user.firstName || user.displayName);

	trackUnsavedValue(
		() => user.customClaims,
		(customClaims) => (user.customClaims = customClaims),
		(customClaims) => customClaimService.updateUserCustomClaims(user.id, customClaims)
	);

	async function reloadUser() {
		user = await userService.get(user.id);
	}

	// Group changes save at once, and Undo puts the previous memberships back
	async function setGroups(groupIds: string[]) {
		user.userGroups = (await userService.updateUserGroups(user.id, groupIds)).userGroups;
	}

	async function addToGroups(groupIds: string[]) {
		await setGroups([...user.userGroups.map((g) => g.id), ...groupIds]);
		toast.success(m.group_memberships_updated());
	}

	async function removeFromGroup(group: UserGroup) {
		const before = user.userGroups.map((g) => g.id);
		try {
			await setGroups(before.filter((id) => id !== group.id));
			toast.success(m.name_removed_from_group({ name: firstName, group: group.friendlyName }), {
				action: { label: m.undo(), onClick: () => setGroups(before).catch(axiosErrorToast) }
			});
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	async function copyUserId() {
		await navigator.clipboard.writeText(user.id);
		toast.success(m.user_id_copied());
	}

	const groupSlug = (group: UserGroup) =>
		group.name !== group.friendlyName.toLowerCase() ? group.name : undefined;
</script>

<svelte:head>
	<title>{user.displayName}</title>
</svelte:head>

{#snippet GroupRow(group: UserGroupMinimal)}
	<OidcClientAvatar id={group.id} name={group.friendlyName} hasLogo={false} size={24} />
	<span class="text-control text-fg min-w-0 flex-1 truncate font-medium">{group.friendlyName}</span>
	<span class="text-caption text-fg-muted">
		{group.userCount === 1 ? m.members_count_one() : m.members_count({ count: group.userCount })}
	</span>
{/snippet}

{#snippet count(n: number)}
	<span class="text-fg-muted tabular-nums">{n}</span>
{/snippet}

<div class="flex flex-col gap-8">
	<PageHeader
		title={user.displayName}
		breadcrumbs={[
			{ label: m.administration() },
			{ label: m.users(), href: '/settings/admin/users' }
		]}
	>
		{#snippet identity()}
			<UserAvatar {user} />
		{/snippet}
		{#snippet status()}
			<UserStatus {user} />
		{/snippet}
		{#snippet tags()}
			<span class="min-w-0 wrap-anywhere">{user.email || m.no_email_address()}</span>
			{#if user.isAdmin}<Badge>{m.admin()}</Badge>{/if}
			{#if readOnly}<Badge>{m.ldap()}</Badge>{/if}
			{#if isMe}<Badge>{m.you()}</Badge>{/if}
		{/snippet}
		{#snippet actions()}
			{#if user.disabled}
				<Button
					variant="secondary"
					usePromiseLoading
					onclick={() => enableUsers([user], reloadUser)}
				>
					<LucideUserCheck />
					{m.enable_user()}
				</Button>
			{:else}
				<Button variant="secondary" onclick={() => (loginCodeUser = user)}>
					<LucideKeyRound />
					{m.login_code_ellipsis()}
				</Button>
			{/if}
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="outline" size="icon" aria-label={m.more_actions()}>
							<LucideEllipsis />
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end">
					<DropdownMenu.Item onclick={copyUserId}>
						<LucideCopy />
						{m.copy_user_id()}
					</DropdownMenu.Item>
					{#if !user.disabled && !readOnly}
						<DropdownMenu.Item
							disabled={isMe}
							onclick={() => confirmDisableUsers([user], reloadUser)}
						>
							<LucideUserX />
							{m.disable_user_ellipsis()}
						</DropdownMenu.Item>
					{/if}
					{#if !readOnly || user.disabled}
						<DropdownMenu.Separator />
						<DropdownMenu.Item
							variant="destructive"
							disabled={isMe}
							onclick={() => confirmDeleteUsers([user], backNavigation.leave)}
						>
							<LucideTrash2 />
							{m.delete_user_ellipsis()}
						</DropdownMenu.Item>
					{/if}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{/snippet}
	</PageHeader>

	<Tabs.Root bind:value={tab} useHash class="gap-10">
		<div class="flex shadow-[inset_0_-1px_0_var(--border)]">
			<div class="scroll-fade-x min-w-0 [scrollbar-width:none] overflow-x-auto">
				<Tabs.List variant="line" class="shadow-none max-sm:gap-4" aria-label={m.user_sections()}>
					<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
					<Tabs.Trigger value="groups"
						>{m.groups()} {@render count(user.userGroups.length)}</Tabs.Trigger
					>
					<Tabs.Trigger value="passkeys"
						>{m.passkeys()} {@render count(passkeys.length)}</Tabs.Trigger
					>
					{#if data.features?.customClaims.enabled}
						<Tabs.Trigger value="custom-claims">
							{m.custom_claims()}
							{@render count(user.customClaims.length)}
						</Tabs.Trigger>
					{/if}
				</Tabs.List>
			</div>
		</div>

		<Tabs.Content value="general" class="flex flex-col gap-10">
			<UserForm bind:user {readOnly} />
		</Tabs.Content>

		<Tabs.Content value="groups">
			<SettingsSection title={m.groups()}>
				{#snippet action()}
					{#if !readOnly}
						<Button variant="secondary" size="sm" onclick={() => (groupPickerOpen = true)}>
							<LucidePlus />
							{m.add_to_groups_ellipsis()}
						</Button>
					{/if}
				{/snippet}
				{#if readOnly}
					<p class="text-caption text-fg-muted mb-2 flex items-start gap-1.5">
						<LucideLock class="mt-1 size-3 shrink-0" />
						{m.group_memberships_come_from_ldap()}
					</p>
				{/if}
				<ul class="border-t">
					{#each user.userGroups as group (group.id)}
						<li class="flex min-h-(--row-2line) items-center gap-3 border-b py-2 ps-4 pe-2">
							<span class="flex min-w-0 flex-1 items-center gap-2">
								<a
									href="/settings/admin/user-groups/{group.id}"
									class="text-control text-fg focus-visible:outline-ring truncate rounded-xs font-medium hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
									>{group.friendlyName}</a
								>
								{#if groupSlug(group)}
									<span
										class="text-mono-s text-fg-muted max-w-fit min-w-[6ch] flex-1 truncate font-mono max-sm:hidden"
										>{groupSlug(group)}</span
									>
								{/if}
								{#if group.ldapId && $appConfigStore.ldapEnabled}<Badge>{m.ldap()}</Badge>{/if}
							</span>

							{#if !readOnly}
								<Button
									variant="ghost"
									size="icon-xs"
									aria-label={m.remove_name_from_group({
										name: firstName,
										group: group.friendlyName
									})}
									onclick={() => removeFromGroup(group)}
								>
									<LucideUserMinus />
								</Button>
							{/if}
						</li>
					{:else}
						<li
							class="text-control text-fg-muted flex min-h-(--row-2line) items-center border-b px-4"
						>
							{m.not_in_any_group()}
						</li>
					{/each}
				</ul>
			</SettingsSection>
		</Tabs.Content>

		<Tabs.Content value="passkeys">
			<SettingsSection title={m.passkeys()} description={m.user_passkeys_section_description()}>
				<AdminPasskeyList userId={user.id} {firstName} bind:passkeys />
			</SettingsSection>
		</Tabs.Content>

		<Tabs.Content value="custom-claims">
			<SettingsSection title={m.custom_claims()} description={m.user_custom_claims_description()}>
				<CustomClaimsInput bind:customClaims={user.customClaims} />
			</SettingsSection>
		</Tabs.Content>
	</Tabs.Root>
</div>

<LoginCodeDialog bind:user={loginCodeUser} />
<PickerDialog
	bind:open={groupPickerOpen}
	title={m.add_name_to_groups({ name: firstName })}
	searchLabel={m.search_groups()}
	load={(search) =>
		userGroupService
			.list({
				search,
				pagination: { page: 1, limit: 50 },
				sort: { column: 'friendlyName', direction: 'asc' }
			})
			.then((res) => res.data)}
	exclude={user.userGroups.map((g) => g.id)}
	row={GroupRow}
	confirmLabel={groupsConfirmLabel}
	onConfirm={addToGroups}
/>
