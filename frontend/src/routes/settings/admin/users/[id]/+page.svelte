<script lang="ts">
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import ProfilePictureSettings from '$lib/components/form/profile-picture-settings.svelte';
	import Badge from '$lib/components/ui/badge/badge.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as Tabs from '$lib/components/ui/tabs';
	import UserGroupSelection from '$lib/components/user-group-selection.svelte';
	import { m } from '$lib/paraglide/messages';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Passkey } from '$lib/types/passkey.type';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { KeyRound, LucideChevronLeft } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { backNavigate } from '../navigate-back-util';
	import UserForm from '../user-form.svelte';
	import AdminPasskeyList from './admin-passkey-list.svelte';

	let { data } = $props();
	let user = $state({
		...data.user,
		userGroupIds: data.user.userGroups.map((g) => g.id)
	});
	let passkeys: Passkey[] = $state(data.passkeys);

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
<Tabs.Root value="general" useHash class="gap-4">
	<div class="overflow-x-auto pb-1">
		<Tabs.List variant="line" class="min-w-max">
			<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
			<Tabs.Trigger value="groups">{m.user_groups()}</Tabs.Trigger>
			<Tabs.Trigger value="passkeys">{m.passkeys()}</Tabs.Trigger>
			<Tabs.Trigger value="custom-claims">{m.custom_claims()}</Tabs.Trigger>
		</Tabs.List>
	</div>

	<Tabs.Content value="general" class="flex flex-col gap-4">
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
	</Tabs.Content>

	<Tabs.Content value="groups" id="user-groups">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.user_groups()}</Card.Title>
				<Card.Description>{m.manage_which_groups_this_user_belongs_to()}</Card.Description>
			</Card.Header>
			<Card.Content>
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
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="passkeys">
		<Item.Group class="bg-card rounded-4xl border p-5 shadow-sm">
			<Item.Root class="border-none bg-transparent p-0">
				<Item.Media class="text-primary/80">
					<KeyRound class="size-5" />
				</Item.Media>
				<Item.Content class="min-w-52">
					<Item.Title class="text-xl font-semibold">{m.passkeys()}</Item.Title>
					<Item.Description
						>{passkeys.length > 0
							? m.manage_this_users_passkeys()
							: m.user_has_no_passkeys_yet()}</Item.Description
					>
				</Item.Content>
			</Item.Root>
			{#if passkeys.length > 0}
				<AdminPasskeyList userId={user.id} bind:passkeys />
			{/if}
		</Item.Group>
	</Tabs.Content>

	<Tabs.Content value="custom-claims" id="user-custom-claims">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.custom_claims()}</Card.Title>
				<Card.Description>
					{m.custom_claims_are_key_value_pairs_that_can_be_used_to_store_additional_information_about_a_user()}
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<CustomClaimsInput bind:customClaims={user.customClaims} />
				<div class="mt-5 flex justify-end">
					<Button onclick={updateCustomClaims} type="submit">{m.save()}</Button>
				</div>
			</Card.Content>
		</Card.Root>
	</Tabs.Content>
</Tabs.Root>
