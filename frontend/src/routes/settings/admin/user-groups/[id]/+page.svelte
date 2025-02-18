<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import UserGroupService from '$lib/services/user-group-service';
	import UserService from '$lib/services/user-service';
	import type { UserGroupCreate } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import UserGroupForm from '../user-group-form.svelte';
	import UserSelection from '../user-selection.svelte';
	import appConfigStore from '$lib/stores/application-configuration-store';

	let { data } = $props();
	let userGroup = $state({
		...data.userGroup,
		userIds: data.userGroup.users.map((u) => u.id)
	});

	const userGroupService = new UserGroupService();
	const userService = new UserService();
	const customClaimService = new CustomClaimService();

	async function updateUserGroup(updatedUserGroup: UserGroupCreate) {
		let success = true;
		await userGroupService
			.update(userGroup.id, updatedUserGroup)
			.then(() => toast.success('User group updated successfully'))
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
	}

	async function updateUserGroupUsers(userIds: string[]) {
		await userGroupService
			.updateUsers(userGroup.id, userIds)
			.then(() => toast.success('Users updated successfully'))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}

	async function updateCustomClaims() {
		await customClaimService
			.updateUserGroupCustomClaims(userGroup.id, userGroup.customClaims)
			.then(() => toast.success('Custom claims updated successfully'))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}
</script>

<svelte:head>
	<title>User Group Details {userGroup.name}</title>
</svelte:head>

<div class="flex items-center justify-between">
	<a class="text-muted-foreground flex text-sm" href="/settings/admin/user-groups"
		><LucideChevronLeft class="h-5 w-5" /> Back</a
	>
	{#if !!userGroup.ldapId}
		<Badge variant="default" class="">LDAP</Badge>
	{/if}
</div>
<Card.Root>
	<Card.Header>
		<Card.Title>General</Card.Title>
	</Card.Header>

	<Card.Content>
		<UserGroupForm existingUserGroup={userGroup} callback={updateUserGroup} />
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>Users</Card.Title>
		<Card.Description>Assign users to this group.</Card.Description>
	</Card.Header>

	<Card.Content>
		{#await userService.list() then users}
			<UserSelection
				{users}
				bind:selectedUserIds={userGroup.userIds}
				selectionDisabled={!!userGroup.ldapId && $appConfigStore.ldapEnabled}
			/>
		{/await}
		<div class="mt-5 flex justify-end">
			<Button disabled={!!userGroup.ldapId && $appConfigStore.ldapEnabled} on:click={() => updateUserGroupUsers(userGroup.userIds)}
				>Save</Button
			>
		</div>
	</Card.Content>
</Card.Root>

<CollapsibleCard
	id="user-group-custom-claims"
	title="Custom Claims"
	description="Custom claims are key-value pairs that can be used to store additional information about a user. These claims will be included in the ID token if the scope 'profile' is requested. Custom claims defined on the user will be prioritized if there are conflicts."
>
	<CustomClaimsInput bind:customClaims={userGroup.customClaims} />
	<div class="mt-5 flex justify-end">
		<Button onclick={updateCustomClaims} type="submit">Save</Button>
	</div>
</CollapsibleCard>
