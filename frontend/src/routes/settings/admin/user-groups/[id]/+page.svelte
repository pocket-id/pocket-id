<script lang="ts">
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import CustomClaimsInput from '#lib/components/form/custom-claims-input.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import * as Card from '#lib/components/ui/card/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import CustomClaimService from '#lib/services/custom-claim-service.ts';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { UserGroupCreate } from '#lib/types/user-group.type.ts';
	import { trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { backNavigate } from '../../users/navigate-back-util';
	import { deleteUserGroupAction } from '../user-group-actions';
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
	const actions = [deleteUserGroupAction(backNavigation.leave)];

	async function updateUserGroup(updatedUserGroup: UserGroupCreate) {
		await userGroupService.update(userGroup.id, updatedUserGroup);
	}

	trackUnsavedValue(
		() => userGroup.userIds,
		(userIds) => {
			userGroup.userIds = userIds;
		},
		(userIds) => userGroupService.updateUsers(userGroup.id, userIds)
	);

	trackUnsavedValue(
		() => userGroup.allowedOidcClientIds,
		(clientIds) => {
			userGroup.allowedOidcClientIds = clientIds;
		},
		async (clientIds) => {
			await userGroupService.updateAllowedOidcClients(userGroup.id, clientIds);
			oidcClientSelectionRef.refresh();
		}
	);

	trackUnsavedValue(
		() => userGroup.customClaims,
		(customClaims) => {
			userGroup.customClaims = customClaims;
		},
		(customClaims) => customClaimService.updateUserGroupCustomClaims(userGroup.id, customClaims)
	);
</script>

<svelte:head>
	<title>{m.user_group_details_name({ name: userGroup.name })}</title>
</svelte:head>

<div class="flex items-center justify-between">
	<button type="button" class="text-muted-foreground flex text-sm" onclick={backNavigation.go}
		><LucideChevronLeft class="size-5" /> {m.back()}</button
	>
	<div class="flex items-center gap-2">
		{#if !!userGroup.ldapId}
			<Badge class="rounded-full" variant="default">{m.ldap()}</Badge>
		{/if}
		<ActionsMenu item={userGroup} {actions} label={m.actions()} variant="outline" size="icon-sm" />
	</div>
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
			</Card.Content>
		</Card.Root>
	</Tabs.Content>
</Tabs.Root>
