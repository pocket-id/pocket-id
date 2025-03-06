<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import ProfilePictureSettings from '$lib/components/form/profile-picture-settings.svelte';
	import Badge from '$lib/components/ui/badge/badge.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import UserGroupService from '$lib/services/user-group-service';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import UserGroupSelection from '$lib/components/user-group-selection.svelte';
	import UserForm from '../user-form.svelte';

	let { data } = $props();
	let user = $state({
		...data.user,
		userGroupIds: data.user.userGroups.map((g) => g.id)
	});

	const userService = new UserService();
	const customClaimService = new CustomClaimService();
	const userGroupService = new UserGroupService();

	async function updateUserGroups(userIds: string[]) {
		await userService
			.updateUserGroups(user.id, userIds)
			.then(() => toast.success('User groups updated successfully'))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}

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
	description="Manage which groups this user belongs to."
>
	{#await userGroupService.list() then groups}
		<UserGroupSelection
			{groups}
			bind:selectedGroupIds={user.userGroupIds}
			selectionDisabled={!!user.ldapId && $appConfigStore.ldapEnabled}
		/>
	{/await}

	<div class="mt-5 flex justify-end">
		<Button
			on:click={() => updateUserGroups(user.userGroupIds)}
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
