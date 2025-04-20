<script lang="ts">
	import { env } from '$env/dynamic/public';
	import CheckboxWithLabel from '$lib/components/form/checkbox-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import AppConfigService from '$lib/services/app-config-service';
	import type { AllAppConfig } from '$lib/types/application-configuration';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';

	let {
		callback,
		appConfig
	}: {
		appConfig: AllAppConfig;
		callback: (appConfig: Partial<AllAppConfig>) => Promise<void>;
	} = $props();

	const appConfigService = new AppConfigService();
	const uiConfigDisabled = env.PUBLIC_UI_CONFIG_DISABLED === 'true';

	let ldapEnabled = $state(appConfig.ldapEnabled);
	let ldapSyncing = $state(false);

	const updatedAppConfig = {
		ldapEnabled: appConfig.ldapEnabled,
		ldapUrl: appConfig.ldapUrl,
		ldapBindDn: appConfig.ldapBindDn,
		ldapBindPassword: appConfig.ldapBindPassword,
		ldapBase: appConfig.ldapBase,
		ldapUserSearchFilter: appConfig.ldapUserSearchFilter,
		ldapUserGroupSearchFilter: appConfig.ldapUserGroupSearchFilter,
		ldapSkipCertVerify: appConfig.ldapSkipCertVerify,
		ldapAttributeUserUniqueIdentifier: appConfig.ldapAttributeUserUniqueIdentifier,
		ldapAttributeUserUsername: appConfig.ldapAttributeUserUsername,
		ldapAttributeUserEmail: appConfig.ldapAttributeUserEmail,
		ldapAttributeUserFirstName: appConfig.ldapAttributeUserFirstName,
		ldapAttributeUserLastName: appConfig.ldapAttributeUserLastName,
		ldapAttributeUserProfilePicture: appConfig.ldapAttributeUserProfilePicture,
		ldapAttributeGroupMember: appConfig.ldapAttributeGroupMember,
		ldapAttributeGroupUniqueIdentifier: appConfig.ldapAttributeGroupUniqueIdentifier,
		ldapAttributeGroupName: appConfig.ldapAttributeGroupName,
		ldapAttributeAdminGroup: appConfig.ldapAttributeAdminGroup,
		ldapSoftDeleteUsers: appConfig.ldapSoftDeleteUsers || true
	};

	const formSchema = z.object({
		ldapUrl: z.string().url(),
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
		ldapAttributeUserProfilePicture: z.string(),
		ldapAttributeGroupMember: z.string(),
		ldapAttributeGroupUniqueIdentifier: z.string().min(1),
		ldapAttributeGroupName: z.string().min(1),
		ldapAttributeAdminGroup: z.string(),
		ldapSoftDeleteUsers: z.boolean()
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, updatedAppConfig);

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

<form onsubmit={onSubmit}>
	<h4 class="text-lg font-semibold">{m.client_configuration()}</h4>
	<fieldset disabled={uiConfigDisabled}>
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
			<CheckboxWithLabel
				id="skip-cert-verify"
				label={m.skip_certificate_verification()}
				description={m.this_can_be_useful_for_selfsigned_certificates()}
				bind:checked={$inputs.ldapSkipCertVerify.value}
			/>
			<CheckboxWithLabel
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
				label={m.group_name_attribute()}
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
			<Button variant="secondary" onclick={onDisable} disabled={uiConfigDisabled}
				>{m.disable()}</Button
			>
			<Button variant="secondary" onclick={syncLdap} isLoading={ldapSyncing}>{m.sync_now()}</Button>
			<Button type="submit" disabled={uiConfigDisabled}>{m.save()}</Button>
		{:else}
			<Button onclick={onEnable} disabled={uiConfigDisabled}>{m.enable()}</Button>
		{/if}
	</div>
</form>
