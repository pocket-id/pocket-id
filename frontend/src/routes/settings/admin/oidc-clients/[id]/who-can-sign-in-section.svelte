<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as RadioGroup from '#lib/components/ui/radio-group/index.ts';
	import UserGroupSelection from '#lib/components/user-group-selection.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { OidcClient } from '#lib/types/oidc.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { trackUnsavedValue } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { LucideLockOpen } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

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

	const options = $derived([
		{
			value: 'all',
			title: m.all_users(),
			description: m.anyone_with_an_appname_account_can_sign_in({
				appName: $appConfigStore.appName
			})
		},
		{
			value: 'restricted',
			title: m.selected_groups(),
			description: m.only_members_of_the_groups_below_can_sign_in()
		}
	]);

	async function restrict() {
		try {
			await oidcService.updateClient(client.id, { ...client, isGroupRestricted: true });
			client.isGroupRestricted = true;
			toast.success(m.who_can_sign_in_updated());
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function unrestrict() {
		openConfirmDialog({
			title: m.allow_all_users_question({ name: client.name }),
			message: m.allow_all_users_consequence({ appName: $appConfigStore.appName }),
			confirm: {
				label: m.allow_all_users(),
				destructive: true,
				icon: LucideLockOpen,
				action: async () => {
					try {
						await oidcService.updateClient(client.id, { ...client, isGroupRestricted: false });
						client.allowedUserGroupIds = [];
						allowedUserGroups.markSaved();
						client.isGroupRestricted = false;
						toast.success(m.who_can_sign_in_updated());
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

<SettingsSection title={m.who_can_sign_in()}>
	<div class="flex flex-col gap-6 pt-2">
		<RadioGroup.Root
			class="grid gap-3 sm:grid-cols-2"
			aria-label={m.who_can_sign_in()}
			bind:value={
				() => (client.isGroupRestricted ? 'restricted' : 'all'),
				(value) => setAccessMode(value as AccessMode)
			}
		>
			{#each options as option (option.value)}
				<Field.Label for="access-mode-{option.value}">
					<Field.Field orientation="horizontal">
						<RadioGroup.Item value={option.value} id="access-mode-{option.value}" />
						<Field.Content>
							<Field.Title>{option.title}</Field.Title>
							<Field.Description>{option.description}</Field.Description>
						</Field.Content>
					</Field.Field>
				</Field.Label>
			{/each}
		</RadioGroup.Root>

		{#if client.isGroupRestricted}
			<div class="flex flex-col gap-2">
				<span class="text-control font-semibold">{m.allowed_groups()}</span>
				<UserGroupSelection bind:selectedGroupIds={client.allowedUserGroupIds} />
			</div>
		{/if}
	</div>
</SettingsSection>
