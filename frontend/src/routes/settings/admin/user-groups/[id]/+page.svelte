<script lang="ts">
	import CustomClaimsInput from '$lib/components/form/custom-claims-input.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Tabs from '$lib/components/ui/tabs';
	import { m } from '$lib/paraglide/messages';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserGroupCreate } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { backNavigate } from '../../users/navigate-back-util';
	import UserGroupForm from '../user-group-form.svelte';
	import UserSelection from '../user-selection.svelte';
	import OidcClientSelection from './oidc-client-selection.svelte';

	let { data } = $props();
	let userGroup = $state({
		...data.userGroup,
		userIds: data.userGroup.users.map((u) => u.id),
		allowedOidcClientIds: data.userGroup.allowedOidcClients.map((c) => c.id)
	});

	let oidcClientSelectionRef: OidcClientSelection;

	const userGroupService = new UserGroupService();
	const customClaimService = new CustomClaimService();
	const backNavigation = backNavigate('/settings/admin/user-groups');

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

	async function updateAllowedOidcClients(allowedClients: string[]) {
		await userGroupService
			.updateAllowedOidcClients(userGroup.id, allowedClients)
			.then(() => {
				toast.success(m.allowed_oidc_clients_updated_successfully());
				oidcClientSelectionRef.refresh();
			})
			.catch((e) => {
				axiosErrorToast(e);
			});
	}
</script>

<svelte:head>
	<title>{m.user_group_details_name({ name: userGroup.name })}</title>
</svelte:head>

<div class="flex items-center justify-between">
	<button type="button" class="text-muted-foreground flex text-sm" onclick={backNavigation.go}
		><LucideChevronLeft class="size-5" /> {m.back()}</button
	>
	{#if !!userGroup.ldapId}
		<Badge class="rounded-full" variant="default">{m.ldap()}</Badge>
	{/if}
</div>
<Tabs.Root value="general" useHash class="gap-4">
	<div class="overflow-x-auto pb-1">
		<Tabs.List variant="line" class="min-w-max">
			<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
			<Tabs.Trigger value="users">{m.users()}</Tabs.Trigger>
			<Tabs.Trigger value="clients">{m.allowed_oidc_clients()}</Tabs.Trigger>
			<Tabs.Trigger value="custom-claims">{m.custom_claims()}</Tabs.Trigger>
		</Tabs.List>
	</div>

	<Tabs.Content value="general">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.general()}</Card.Title>
			</Card.Header>

			<Card.Content>
				<UserGroupForm existingUserGroup={userGroup} callback={updateUserGroup} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="users">
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
	</Tabs.Content>

	<Tabs.Content value="clients" id="user-group-oidc-clients">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.allowed_oidc_clients()}</Card.Title>
				<Card.Description>{m.allowed_oidc_clients_description()}</Card.Description>
			</Card.Header>
			<Card.Content>
				<OidcClientSelection
					bind:this={oidcClientSelectionRef}
					bind:selectedGroupIds={userGroup.allowedOidcClientIds}
				/>
				<div class="mt-5 flex justify-end gap-3">
					<Button onclick={() => updateAllowedOidcClients(userGroup.allowedOidcClientIds)}
						>{m.save()}</Button
					>
				</div>
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="custom-claims" id="user-group-custom-claims">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.custom_claims()}</Card.Title>
				<Card.Description>
					{m.custom_claims_are_key_value_pairs_that_can_be_used_to_store_additional_information_about_a_user_prioritized()}
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<CustomClaimsInput bind:customClaims={userGroup.customClaims} />
				<div class="mt-5 flex justify-end">
					<Button onclick={updateCustomClaims} type="submit">{m.save()}</Button>
				</div>
			</Card.Content>
		</Card.Root>
	</Tabs.Content>
</Tabs.Root>
