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
	import { onMount } from 'svelte';
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import GroupSelection from '../group-selection.svelte';

	let { data } = $props();
	let user = $state(data.user);
	let allUserGroups = $state<Paginated<UserGroupWithUserCount>>(data.allUserGroups);
	let requestOptions: SearchPaginationSortRequest | undefined = $state();

	// Initialize with the IDs from the server - no need to recalculate
	let userGroupIds = $state<string[]>(data.userGroupIds);

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
			// Get fresh data about all selected groups
			const selectedGroups = await Promise.all(
				userGroupIds.map((groupId) => userGroupService.get(groupId))
			);

			// Determine which groups to add the user to - use the UI's state (userGroupIds)
			// as the source of truth, not the backend state
			const previousUserGroups = await userService.getUserGroups(user.id);
			const previousUserGroupIds = previousUserGroups.map((g) => g.id);

			// Add user to newly selected groups
			const addPromises = userGroupIds
				.filter((groupId) => !previousUserGroupIds.includes(groupId))
				.map(async (groupId) => {
					try {
						console.log(`Adding user ${user.id} to newly selected group ${groupId}`);
						const group = await userGroupService.get(groupId);
						const currentUserIds = group.users.map((u) => u.id);
						return userGroupService.updateUsers(groupId, [...currentUserIds, user.id]);
					} catch (error) {
						console.error(`Error adding user to group ${groupId}:`, error);
						throw error;
					}
				});

			// Remove user from unselected groups
			const removePromises = previousUserGroupIds
				.filter((groupId) => !userGroupIds.includes(groupId))
				.map(async (groupId) => {
					try {
						const group = await userGroupService.get(groupId);
						const currentUserIds = group.users.map((u) => u.id);
						return userGroupService.updateUsers(
							groupId,
							currentUserIds.filter((id) => id !== user.id)
						);
					} catch (error) {
						console.error(`Error removing user from group ${groupId}:`, error);
						throw error;
					}
				});

			// Wait for all operations to complete
			await Promise.all([...addPromises, ...removePromises]);

			toast.success('User group memberships updated successfully');

			// Refresh the table
			if (requestOptions) {
				const refreshedGroups = await userGroupService.list(requestOptions);
				allUserGroups = refreshedGroups;

				// Update the user's group memberships
				const updatedUserGroups = await userService.getUserGroups(user.id);
				userGroupIds = updatedUserGroups.map((g) => g.id);
			}
		} catch (e) {
			console.error('Error in updateUserGroups:', e);
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
	<GroupSelection
		groups={allUserGroups}
		bind:selectedGroupIds={userGroupIds}
		selectionDisabled={!!user.ldapId && $appConfigStore.ldapEnabled}
	/>
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
