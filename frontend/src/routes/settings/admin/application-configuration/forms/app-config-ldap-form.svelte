<script lang="ts">
	import { env } from '$env/dynamic/public';
	import CheckboxWithLabel from '$lib/components/form/checkbox-with-label.svelte';
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
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
		ldapAttributeAdminGroup: appConfig.ldapAttributeAdminGroup
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
		ldapAttributeAdminGroup: z.string()
	});

	const { inputs, ...form } = createForm<typeof formSchema>(formSchema, updatedAppConfig);

	async function onSubmit() {
		const data = form.validate();
		if (!data) return false;
		await callback({
			...data,
			ldapEnabled: true
		});
		toast.success('LDAP configuration updated successfully');
		return true;
	}

	async function onDisable() {
		ldapEnabled = false;
		await callback({ ldapEnabled });
		toast.success('LDAP disabled successfully');
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
			.then(() => toast.success('LDAP sync finished'))
			.catch(axiosErrorToast);

		ldapSyncing = false;
	}
</script>

<form onsubmit={onSubmit}>
	<h4 class="text-lg font-semibold">Client Configuration</h4>
	<fieldset disabled={uiConfigDisabled}>
		<div class="mt-4 grid grid-cols-1 items-start gap-5 md:grid-cols-2">
			<FormInput
				label="LDAP URL"
				placeholder="ldap://example.com:389"
				bind:input={$inputs.ldapUrl}
			/>
			<FormInput
				label="LDAP Bind DN"
				placeholder="cn=people,dc=example,dc=com"
				bind:input={$inputs.ldapBindDn}
			/>
			<FormInput label="LDAP Bind Password" type="password" bind:input={$inputs.ldapBindPassword} />
			<FormInput
				label="LDAP Base DN"
				placeholder="dc=example,dc=com"
				bind:input={$inputs.ldapBase}
			/>
			<FormInput
				label="User Search Filter"
				description="The Search filter to use to search/sync users."
				placeholder="(objectClass=person)"
				bind:input={$inputs.ldapUserSearchFilter}
			/>
			<FormInput
				label="Groups Search Filter"
				description="The Search filter to use to search/sync groups."
				placeholder="(objectClass=groupOfNames)"
				bind:input={$inputs.ldapUserGroupSearchFilter}
			/>
			<CheckboxWithLabel
				id="skip-cert-verify"
				label="Skip Certificate Verification"
				description="This can be useful for self-signed certificates."
				bind:checked={$inputs.ldapSkipCertVerify.value}
			/>
		</div>
		<h4 class="mt-10 text-lg font-semibold">Attribute Mapping</h4>
		<div class="mt-4 grid grid-cols-1 items-end gap-5 md:grid-cols-2">
			<FormInput
				label="User Unique Identifier Attribute"
				description="The value of this attribute should never change."
				placeholder="uuid"
				bind:input={$inputs.ldapAttributeUserUniqueIdentifier}
			/>
			<FormInput
				label="Username Attribute"
				placeholder="uid"
				bind:input={$inputs.ldapAttributeUserUsername}
			/>
			<FormInput
				label="User Mail Attribute"
				placeholder="mail"
				bind:input={$inputs.ldapAttributeUserEmail}
			/>
			<FormInput
				label="User First Name Attribute"
				placeholder="givenName"
				bind:input={$inputs.ldapAttributeUserFirstName}
			/>
			<FormInput
				label="User Last Name Attribute"
				placeholder="sn"
				bind:input={$inputs.ldapAttributeUserLastName}
			/>
			<FormInput
				label="User Profile Picture Attribute"
				description="The value of this attribute can either be a URL, a binary or a base64 encoded image."
				placeholder="jpegPhoto"
				bind:input={$inputs.ldapAttributeUserProfilePicture}
			/>
			<FormInput
				label="Group Members Attribute"
				description="The attribute to use for querying members of a group."
				placeholder="member"
				bind:input={$inputs.ldapAttributeGroupMember}
			/>
			<FormInput
				label="Group Unique Identifier Attribute"
				description="The value of this attribute should never change."
				placeholder="uuid"
				bind:input={$inputs.ldapAttributeGroupUniqueIdentifier}
			/>
			<FormInput
				label="Group Name Attribute"
				placeholder="cn"
				bind:input={$inputs.ldapAttributeGroupName}
			/>
			<FormInput
				label="Admin Group Name"
				description="Members of this group will have Admin Privileges in Pocket ID."
				placeholder="_admin_group_name"
				bind:input={$inputs.ldapAttributeAdminGroup}
			/>
		</div>
	</fieldset>

	<div class="mt-8 flex flex-wrap justify-end gap-3">
		{#if ldapEnabled}
			<Button variant="secondary" onclick={onDisable} disabled={uiConfigDisabled}>Disable</Button>
			<Button variant="secondary" onclick={syncLdap} isLoading={ldapSyncing}>Sync now</Button>
			<Button type="submit" disabled={uiConfigDisabled}>Save</Button>
		{:else}
			<Button onclick={onEnable} disabled={uiConfigDisabled}>Enable</Button>
		{/if}
	</div>
</form>
