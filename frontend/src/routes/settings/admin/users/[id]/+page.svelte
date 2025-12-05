<script lang="ts">
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import ProfilePictureSettings from '$lib/components/form/profile-picture-settings.svelte';
	import Badge from '$lib/components/ui/badge/badge.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import UserGroupSelection from '$lib/components/user-group-selection.svelte';
	import { m } from '$lib/paraglide/messages';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { backNavigate } from '../navigate-back-util';
	import UserForm from '../user-form.svelte';
	import * as Table from '$lib/components/ui/table/index.js';

	let { data } = $props();
	let user = $state({
		...data.user,
		userGroupIds: data.user.userGroups.map((g) => g.id)
	});

	const userService = new UserService();
	const customClaimService = new CustomClaimService();
	const backNavigation = backNavigate('/settings/admin/users');

	async function updateUserGroups(userIds: string[]) {
		await userService
			.updateUserGroups(user.id, userIds)
			.then(() => toast.success(m.user_groups_updated_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}

	async function updateUser(updatedUser: UserCreate) {
		let success = true;
		await userService
			.update(user.id, updatedUser)
			.then(() => toast.success(m.user_updated_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
	}

	async function updateCustomClaims() {
		await customClaimService
			.updateUserCustomClaims(user.id, user.customClaims)
			.then(() => toast.success(m.custom_claims_updated_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
			});
	}

	async function updateProfilePicture(image: File) {
		await userService
			.updateProfilePicture(user.id, image)
			.then(() => toast.success(m.profile_picture_updated_successfully()))
			.catch(axiosErrorToast);
	}

	async function resetProfilePicture() {
		await userService
			.resetProfilePicture(user.id)
			.then(() => toast.success(m.profile_picture_has_been_reset()))
			.catch(axiosErrorToast);
	}
</script>

<svelte:head>
	<title
		>{m.user_details_firstname_lastname({
			firstName: user.firstName,
			lastName: user.lastName ?? ''
		})}</title
	>
</svelte:head>

<div class="flex items-center justify-between">
	<button class="text-muted-foreground flex text-sm" onclick={() => backNavigation.go()}
		><LucideChevronLeft class="size-5" /> {m.back()}</button
	>
	{#if !!user.ldapId}
		<Badge class="rounded-full" variant="default">{m.ldap()}</Badge>
	{/if}
</div>
<Card.Root>
	<Card.Header>
		<Card.Title>{m.general()}</Card.Title>
	</Card.Header>
	<Card.Content>
		<UserForm existingUser={user} callback={updateUser} />
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Content>
		<ProfilePictureSettings
			userId={user.id}
			isLdapUser={!!user.ldapId}
			updateCallback={updateProfilePicture}
			resetCallback={resetProfilePicture}
		/>
	</Card.Content>
</Card.Root>

<CollapsibleCard
	id="user-groups"
	title={m.user_groups()}
	description={m.manage_which_groups_this_user_belongs_to()}
>
	<UserGroupSelection
		bind:selectedGroupIds={user.userGroupIds}
		selectionDisabled={!!user.ldapId && $appConfigStore.ldapEnabled}
	/>
	<div class="mt-5 flex justify-end">
		<Button
			onclick={() => updateUserGroups(user.userGroupIds)}
			disabled={!!user.ldapId && $appConfigStore.ldapEnabled}
			type="submit">{m.save()}</Button
		>
	</div>
</CollapsibleCard>

<CollapsibleCard
	id="user-custom-claims"
	title={m.custom_claims()}
	description={m.custom_claims_are_key_value_pairs_that_can_be_used_to_store_additional_information_about_a_user()}
>
	<CustomClaimsInput bind:customClaims={user.customClaims} />
	<div class="mt-5 flex justify-end">
		<Button onclick={updateCustomClaims} type="submit">{m.save()}</Button>
	</div>
</CollapsibleCard>

<CollapsibleCard
	id="user-ldap-claims"
	title={m.ldap_extra_attributes()}
	description={m.ldap_extra_attributes_description()}
>
  <Table.Root class="min-w-full table-auto overflow-x-auto">
    <Table.Header>
      <Table.Row>
        <Table.Head>Attribute</Table.Head>
        <Table.Head>Value(s)</Table.Head>
      </Table.Row>
    </Table.Header>
    <Table.Body>
      {#each user.ldapAttributes as attribute}
        <Table.Row>
          <Table.Cell>{attribute.key}</Table.Cell>
          <Table.Cell>{#if ["{", "["].some(p => attribute.value.startsWith(p))}
            {@html JSON.parse(attribute.value).join('<br/> ')}
            {:else}
            {attribute.value}
            {/if}</Table.Cell>
        </Table.Row>
      {/each}
    </Table.Body>
  </Table.Root>
</CollapsibleCard>