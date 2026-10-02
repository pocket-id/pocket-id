<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import * as Card from '#lib/components/ui/card/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as RadioGroup from '#lib/components/ui/radio-group/index.ts';
	import UserGroupSelection from '#lib/components/user-group-selection.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import type { OidcClient } from '#lib/types/oidc.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';

	type AccessMode = 'all' | 'restricted';

	let {
		client = $bindable()
	}: {
		client: OidcClient & { allowedUserGroupIds: string[] };
	} = $props();

	const oidcService = new OidcService();

	const allowedUserGroups = trackUnsavedValue(
		() => client.allowedUserGroupIds,
		(allowedUserGroupIds) => {
			client.allowedUserGroupIds = allowedUserGroupIds;
		},
		(allowedUserGroupIds) => oidcService.updateAllowedUserGroups(client.id, allowedUserGroupIds)
	);

	async function restrict() {
		try {
			await oidcService.updateClient(client.id, { ...client, isGroupRestricted: true });
			client.isGroupRestricted = true;
			toast.success(m.user_groups_restriction_updated_successfully());
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function unrestrict() {
		openConfirmDialog({
			title: m.unrestrict_oidc_client({ clientName: client.name }),
			message: {
				message: m.confirm_unrestrict_oidc_client_description,
				inputs: { clientName: client.name }
			},
			confirm: {
				label: m.unrestrict(),
				destructive: true,
				action: async () => {
					try {
						await oidcService.updateClient(client.id, { ...client, isGroupRestricted: false });
						client.allowedUserGroupIds = [];
						allowedUserGroups.markSaved();
						client.isGroupRestricted = false;
						toast.success(m.user_groups_restriction_updated_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	// The selection only follows the saved restriction, so a cancelled confirmation leaves it where it was
	function setAccessMode(accessMode: AccessMode) {
		if (accessMode === 'restricted' && !client.isGroupRestricted) {
			restrict();
		} else if (accessMode === 'all' && client.isGroupRestricted) {
			unrestrict();
		}
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.allowed_user_groups()}</Card.Title>
		<Card.Description>{m.allowed_user_groups_description()}</Card.Description>
	</Card.Header>
	<Card.Content class="flex flex-col gap-6">
		<RadioGroup.Root
			class="grid gap-3 sm:grid-cols-2"
			bind:value={
				() => (client.isGroupRestricted ? 'restricted' : 'all'),
				(value) => setAccessMode(value as AccessMode)
			}
		>
			<Field.Label for="access-mode-all">
				<Field.Field orientation="horizontal">
					<RadioGroup.Item value="all" id="access-mode-all" />
					<Field.Content>
						<Field.Title>{m.all_users()}</Field.Title>
						<Field.Description>{m.all_users_can_sign_in_to_this_client()}</Field.Description>
					</Field.Content>
				</Field.Field>
			</Field.Label>
			<Field.Label for="access-mode-restricted">
				<Field.Field orientation="horizontal">
					<RadioGroup.Item value="restricted" id="access-mode-restricted" />
					<Field.Content>
						<Field.Title>{m.selected_user_groups()}</Field.Title>
						<Field.Description>
							{m.only_members_of_the_selected_groups_can_sign_in()}
						</Field.Description>
					</Field.Content>
				</Field.Field>
			</Field.Label>
		</RadioGroup.Root>

		{#if client.isGroupRestricted}
			<div transition:slide={{ duration: 200 }}>
				<UserGroupSelection bind:selectedGroupIds={client.allowedUserGroupIds} />
			</div>
		{/if}
	</Card.Content>
</Card.Root>
