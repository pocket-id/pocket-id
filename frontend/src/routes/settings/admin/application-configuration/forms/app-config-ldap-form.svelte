<script lang="ts">
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { ldap, m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const appConfigService = new AppConfigService();

	let ldapEnabled = $state(appConfig.ldapEnabled);
	let ldapSyncing = $state(false);

	// const updatedAppConfig = {
	// 	ldapEnabled: appConfig.ldapEnabled,
	// 	ldapUrl: appConfig.ldapUrl,
	// 	ldapBindDn: appConfig.ldapBindDn,
	// 	ldapBindPassword: appConfig.ldapBindPassword,
	// 	ldapBase: appConfig.ldapBase,
	// 	ldapUserSearchFilter: appConfig.ldapUserSearchFilter,
	// 	ldapUserGroupSearchFilter: appConfig.ldapUserGroupSearchFilter,
	// 	ldapSkipCertVerify: appConfig.ldapSkipCertVerify,
	// 	ldapAttributeUserUniqueIdentifier: appConfig.ldapAttributeUserUniqueIdentifier,
	// 	ldapAttributeUserUsername: appConfig.ldapAttributeUserUsername,
	// 	ldapAttributeUserEmail: appConfig.ldapAttributeUserEmail,
	// 	ldapAttributeUserFirstName: appConfig.ldapAttributeUserFirstName,
	// 	ldapAttributeUserLastName: appConfig.ldapAttributeUserLastName,
	// 	ldapAttributeUserProfilePicture: appConfig.ldapAttributeUserProfilePicture,
	// 	ldapAttributeGroupMember: appConfig.ldapAttributeGroupMember,
	// 	ldapAttributeGroupUniqueIdentifier: appConfig.ldapAttributeGroupUniqueIdentifier,
	// 	ldapAttributeGroupName: appConfig.ldapAttributeGroupName,
	// 	ldapAttributeAdminGroup: appConfig.ldapAttributeAdminGroup,
	// 	ldapSoftDeleteUsers: appConfig.ldapSoftDeleteUsers || true
	// };

	const formSchema = z.object({
		ldapUrl: z.url(),
		ldapBindDn: z.string().min(1),
		ldapBindPassword: z.string().min(1),
		ldapBase: z.string().min(1),
		ldapUserSearchFilter: z.string().min(1),
		ldapUserGroupSearchFilter: z.string().min(1),
		ldapSkipCertVerify: z.boolean(),
		ldapAttributeUserUniqueIdentifier: z.string().min(1),
		ldapAttributeUserUsername: z.string().min(1),
		ldapAttributeUserEmail: z.string().min(1),
		ldapAttributeUserFirstName: z.string().min(1),
		ldapAttributeUserLastName: z.string().min(1),
		ldapAttributeUserDisplayName: z.string().min(1),
		ldapAttributeUserProfilePicture: z.string(),
		ldapAttributeGroupMember: z.string(),
		ldapAttributeGroupUniqueIdentifier: z.string().min(1),
		ldapAttributeGroupName: z.string().min(1),
		ldapAttributeAdminGroup: z.string(),
		ldapSoftDeleteUsers: z.boolean()
	});

	let { inputs, ...form } = $derived(createForm(formSchema, appConfig));

	async function onSubmit() {
		const data = form.validate();
		if (!data) return false;
		await callback({
			...data,
			ldapEnabled: true
		});
		toast.success(m.ldap_configuration_updated_successfully());
		return true;
	}

	async function onDisable() {
		ldapEnabled = false;
		await callback({ ldapEnabled });
		toast.success(m.ldap_disabled_successfully());
	}

	async function onEnable() {
		if (await onSubmit()) {
			ldapEnabled = true;
		}
	}

	async function syncLdap() {
		ldapSyncing = true;
		await appConfigService
			.syncLdap()
			.then(() => toast.success(m.ldap_sync_finished()))
			.catch(axiosErrorToast);

		ldapSyncing = false;
	}
</script>

<form onsubmit={preventDefault(onSubmit)}>
	<h4 class="text-lg font-semibold">{m.client_configuration()}</h4>
	<fieldset disabled={$appConfigStore.uiConfigDisabled}>
		<div class="mt-4 grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput
				label={m.ldap_url()}
				placeholder="ldap://example.com:389"
				bind:input={$inputs.ldapUrl}
			/>
			<FormInput
				label={m.ldap_bind_dn()}
				placeholder="cn=people,dc=example,dc=com"
				bind:input={$inputs.ldapBindDn}
			/>
			<FormInput
				label={m.ldap_bind_password()}
				type="password"
				bind:input={$inputs.ldapBindPassword}
			/>
			<FormInput
				label={m.ldap_base_dn()}
				placeholder="dc=example,dc=com"
				bind:input={$inputs.ldapBase}
			/>
			<FormInput
				label={m.user_search_filter()}
				description={m.the_search_filter_to_use_to_search_or_sync_users()}
				placeholder="(objectClass=person)"
				bind:input={$inputs.ldapUserSearchFilter}
			/>
			<FormInput
				label={m.groups_search_filter()}
				description={m.the_search_filter_to_use_to_search_or_sync_groups()}
				placeholder="(objectClass=groupOfNames)"
				bind:input={$inputs.ldapUserGroupSearchFilter}
			/>
			<SwitchWithLabel
				id="skip-cert-verify"
				label={m.skip_certificate_verification()}
				description={m.this_can_be_useful_for_selfsigned_certificates()}
				bind:checked={$inputs.ldapSkipCertVerify.value}
			/>
			<SwitchWithLabel
				id="ldap-soft-delete-users"
				label={m.ldap_soft_delete_users()}
				description={m.ldap_soft_delete_users_description()}
				bind:checked={$inputs.ldapSoftDeleteUsers.value}
			/>
		</div>
		<h4 class="mt-10 text-lg font-semibold">{m.attribute_mapping()}</h4>
		<div class="mt-4 grid grid-cols-1 items-end gap-5 md:grid-cols-2">
			<FormInput
				label={m.user_unique_identifier_attribute()}
				description={m.the_value_of_this_attribute_should_never_change()}
				placeholder="uuid"
				bind:input={$inputs.ldapAttributeUserUniqueIdentifier}
			/>
			<FormInput
				label={m.username_attribute()}
				placeholder="uid"
				bind:input={$inputs.ldapAttributeUserUsername}
			/>
			<FormInput
				label={m.user_mail_attribute()}
				placeholder="mail"
				bind:input={$inputs.ldapAttributeUserEmail}
			/>
			<FormInput
				label={m.user_first_name_attribute()}
				placeholder="givenName"
				bind:input={$inputs.ldapAttributeUserFirstName}
			/>
			<FormInput
				label={m.user_last_name_attribute()}
				placeholder="sn"
				bind:input={$inputs.ldapAttributeUserLastName}
			/>
			<FormInput
				label={m.display_name_attribute()}
				placeholder="displayName"
				bind:input={$inputs.ldapAttributeUserDisplayName}
			/>
			<FormInput
				label={m.user_profile_picture_attribute()}
				description={m.the_value_of_this_attribute_can_either_be_a_url_binary_or_base64_encoded_image()}
				placeholder="jpegPhoto"
				bind:input={$inputs.ldapAttributeUserProfilePicture}
			/>
			<FormInput
				label={m.group_members_attribute()}
				description={m.the_attribute_to_use_for_querying_members_of_a_group()}
				placeholder="member"
				bind:input={$inputs.ldapAttributeGroupMember}
			/>
			<FormInput
				label={m.group_unique_identifier_attribute()}
				description={m.the_value_of_this_attribute_should_never_change()}
				placeholder="uuid"
				bind:input={$inputs.ldapAttributeGroupUniqueIdentifier}
			/>
			<FormInput
				label={m.group_rdn_attribute()}
				description={m.group_rdn_attribute_description()}
				placeholder="cn"
				bind:input={$inputs.ldapAttributeGroupName}
			/>
			<FormInput
				label={m.admin_group_name()}
				description={m.members_of_this_group_will_have_admin_privileges_in_pocketid()}
				placeholder="_admin_group_name"
				bind:input={$inputs.ldapAttributeAdminGroup}
			/>
		</div>
	</fieldset>

	<div class="mt-8 flex flex-wrap justify-end gap-3">
		{#if ldapEnabled}
			<Button variant="secondary" onclick={onDisable} disabled={$appConfigStore.uiConfigDisabled}
				>{m.disable()}</Button
			>
			<Button variant="secondary" onclick={syncLdap} isLoading={ldapSyncing}>{m.sync_now()}</Button>
			<Button type="submit" disabled={$appConfigStore.uiConfigDisabled}>{m.save()}</Button>
		{:else}
			<Button onclick={onEnable} disabled={$appConfigStore.uiConfigDisabled}>{m.enable()}</Button>
		{/if}
	</div>
</form>
