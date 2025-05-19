<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserGroupCreate } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import UserGroupForm from '../user-group-form.svelte';
	import UserSelection from '../user-selection.svelte';
	import { m } from '$lib/paraglide/messages';

	let { data } = $props();
	let userGroup = $state({
		...data.userGroup,
		userIds: data.userGroup.users.map((u) => u.id)
	});

	const userGroupService = new UserGroupService();
	const customClaimService = new CustomClaimService();

	async function updateUserGroup(updatedUserGroup: UserGroupCreate) {
		let success = true;
		await userGroupService
			.update(userGroup.id, updatedUserGroup)
			.then(() => toast.success(m.user_group_updated_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
	}

	async function updateUserGroupUsers(userIds: string[]) {
		await userGroupService
			.updateUsers(userGroup.id, userIds)
			.then(() => toast.success(m.users_updated_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}

	async function updateCustomClaims() {
		await customClaimService
			.updateUserGroupCustomClaims(userGroup.id, userGroup.customClaims)
			.then(() => toast.success(m.custom_claims_updated_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}
</script>

<svelte:head>
	<title>{m.user_group_details_name({ name: userGroup.name })}</title>
</svelte:head>

<div class="flex items-center justify-between">
	<a class="text-muted-foreground flex text-sm" href="/settings/admin/user-groups"
		><LucideChevronLeft class="h-5 w-5" /> {m.back()}</a
	>
	{#if !!userGroup.ldapId}
		<Badge class="rounded-full" variant="default" class="">{m.ldap()}</Badge>
	{/if}
</div>
<Card.Root>
	<Card.Header>
		<Card.Title>{m.general()}</Card.Title>
	</Card.Header>

	<Card.Content>
		<UserGroupForm existingUserGroup={userGroup} callback={updateUserGroup} />
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.users()}</Card.Title>
		<Card.Description>{m.assign_users_to_this_group()}</Card.Description>
	</Card.Header>

	<Card.Content>
		<UserSelection
			bind:selectedUserIds={userGroup.userIds}
			selectionDisabled={!!userGroup.ldapId && $appConfigStore.ldapEnabled}
		/>
		<div class="mt-5 flex justify-end">
			<Button
				disabled={!!userGroup.ldapId && $appConfigStore.ldapEnabled}
				onclick={() => updateUserGroupUsers(userGroup.userIds)}>{m.save()}</Button
			>
		</div>
	</Card.Content>
</Card.Root>

<CollapsibleCard
	id="user-group-custom-claims"
	title={m.custom_claims()}
	description={m.custom_claims_are_key_value_pairs_that_can_be_used_to_store_additional_information_about_a_user_prioritized()}
>
	<CustomClaimsInput bind:customClaims={userGroup.customClaims} />
	<div class="mt-5 flex justify-end">
		<Button onclick={updateCustomClaims} type="submit">{m.save()}</Button>
	</div>
</CollapsibleCard>
