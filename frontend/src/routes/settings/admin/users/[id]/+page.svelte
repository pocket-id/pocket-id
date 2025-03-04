<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import ProfilePictureSettings from '$lib/components/form/profile-picture-settings.svelte';
	import Badge from '$lib/components/ui/badge/badge.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Checkbox from '$lib/components/ui/checkbox';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import UserService from '$lib/services/user-service';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import type { UserGroup, UserGroupWithUserCount } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import UserForm from '../user-form.svelte';
	import type { Paginated } from '$lib/types/pagination.type';
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';

	let { data } = $props();
	let user = $state(data.user);
	let allUserGroups = $state(data.allUserGroups);
	let requestOptions: SearchPaginationSortRequest | undefined = $state();

	// Track user's group memberships
	let userGroupIds = $state<string[]>(user.userGroups?.map((g) => g.id) || []);

	const userService = new UserService();
	const customClaimService = new CustomClaimService();
	const userGroupService = new UserGroupService();

	// Check if user is in a group
	function isUserInGroup(groupId: string): boolean {
		return userGroupIds.includes(groupId);
	}

	// Toggle user group membership
	function toggleUserGroup(group: UserGroup): void {
		if (isUserInGroup(group.id)) {
			userGroupIds = userGroupIds.filter((id) => id !== group.id);
		} else {
			userGroupIds = [...userGroupIds, group.id];
		}
	}

	// Update user's group memberships
	async function updateUserGroups() {
		try {
			// Update each selected group to include this user
			await Promise.all(
				userGroupIds.map((groupId) => {
					// Get current users in the group
					return userGroupService.get(groupId).then((group) => {
						const currentUserIds = group.users.map((u) => u.id);
						// Add current user if not already included
						if (!currentUserIds.includes(user.id)) {
							return userGroupService.updateUsers(groupId, [...currentUserIds, user.id]);
						}
						return group;
					});
				})
			);

			// Remove user from unselected groups - need to get all groups, not just visible ones
			const allGroups = await userGroupService.list();
			await Promise.all(
				allGroups.data
					.filter((g) => !userGroupIds.includes(g.id))
					.map(async (group) => {
						// Check if user is in this group
						const groupDetails = await userGroupService.get(group.id);
						const groupUserIds = groupDetails.users.map((u) => u.id);
						if (groupUserIds.includes(user.id)) {
							// Remove user from group
							return userGroupService.updateUsers(
								group.id,
								groupUserIds.filter((id) => id !== user.id)
							);
						}
					})
			);

			toast.success('User group memberships updated successfully');

			// Update the table after saving
			allUserGroups = await userGroupService.list(requestOptions);
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	// Existing functions...
	async function updateUser(updatedUser: UserCreate) {
		let success = true;
		await userService
			.update(user.id, updatedUser)
			.then(() => toast.success('User updated successfully'))
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
	}

	async function updateCustomClaims() {
		await customClaimService
			.updateUserCustomClaims(user.id, user.customClaims)
			.then(() => toast.success('Custom claims updated successfully'))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}

	async function updateProfilePicture(image: File) {
		await userService
			.updateProfilePicture(user.id, image)
			.then(() => toast.success('Profile picture updated successfully'))
			.catch(axiosErrorToast);
	}

	// Modified function to handle checkbox changes
	function onGroupCheckChange(checked: boolean, group: UserGroup): void {
		if (checked) {
			if (!userGroupIds.includes(group.id)) {
				userGroupIds = [...userGroupIds, group.id];
			}
		} else {
			userGroupIds = userGroupIds.filter((id) => id !== group.id);
		}
	}
</script>

<svelte:head>
	<title>User Details {user.firstName} {user.lastName}</title>
</svelte:head>

<div class="flex items-center justify-between">
	<a class="text-muted-foreground flex text-sm" href="/settings/admin/users"
		><LucideChevronLeft class="h-5 w-5" /> Back</a
	>
	{#if !!user.ldapId}
		<Badge variant="default" class="">LDAP</Badge>
	{/if}
</div>
<Card.Root>
	<Card.Header>
		<Card.Title>General</Card.Title>
	</Card.Header>
	<Card.Content>
		<UserForm existingUser={user} callback={updateUser} />
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Content class="pt-6">
		<ProfilePictureSettings
			userId={user.id}
			isLdapUser={!!user.ldapId}
			callback={updateProfilePicture}
		/>
	</Card.Content>
</Card.Root>

<CollapsibleCard
	id="user-groups"
	title="User Groups"
	description="Manage which groups this user belongs to. Group membership affects permissions and access to OIDC clients."
>
	<AdvancedTable
		items={allUserGroups}
		onRefresh={async (options) => {
			const groups = await userGroupService.list(options);
			// Map each group to include hasMember/selected property
			groups.data = await Promise.all(
				groups.data.map(async (group) => {
					const groupDetails = await userGroupService.get(group.id);
					return {
						...group,
						hasMember: groupDetails.users.some((u) => u.id === user.id)
					};
				})
			);
			return groups;
		}}
		columns={[
			{ label: 'Friendly Name', sortColumn: 'friendlyName' },
			{ label: 'Name', sortColumn: 'name' },
			{ label: 'User Count', sortColumn: 'userCount' },
			...($appConfigStore.ldapEnabled ? [{ label: 'Source' }] : [])
		]}
		selectedIds={userGroupIds}
		selectionDisabled={!!user.ldapId && $appConfigStore.ldapEnabled}
	>
		{#snippet rows({ item })}
			<Table.Cell>{item.friendlyName}</Table.Cell>
			<Table.Cell>{item.name}</Table.Cell>
			<Table.Cell>{item.userCount}</Table.Cell>
			{#if $appConfigStore.ldapEnabled}
				<Table.Cell>
					<Badge variant={item.ldapId ? 'default' : 'outline'}
						>{item.ldapId ? 'LDAP' : 'Local'}</Badge
					>
				</Table.Cell>
			{/if}
		{/snippet}
	</AdvancedTable>
	<div class="mt-5 flex justify-end">
		<Button
			on:click={updateUserGroups}
			disabled={!!user.ldapId && $appConfigStore.ldapEnabled}
			type="submit">Save</Button
		>
	</div>
</CollapsibleCard>

<CollapsibleCard
	id="user-custom-claims"
	title="Custom Claims"
	description="Custom claims are key-value pairs that can be used to store additional information about a user. These claims will be included in the ID token if the scope 'profile' is requested."
>
	<CustomClaimsInput bind:customClaims={user.customClaims} />
	<div class="mt-5 flex justify-end">
		<Button on:click={updateCustomClaims} type="submit">Save</Button>
	</div>
</CollapsibleCard>
