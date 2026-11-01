<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import ProfilePictureSettings from '#lib/components/form/profile-picture-settings.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Checkbox } from '#lib/components/ui/checkbox/index.ts';
	import { Switch } from '#lib/components/ui/switch/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { User } from '#lib/types/user.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { emptyToUndefined, usernameSchema } from '#lib/utils/zod-util.ts';
	import { LucideLock, LucideShieldCheck } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { get } from 'svelte/store';
	import { z } from 'zod/v4';

	let { user = $bindable(), readOnly }: { user: User; readOnly: boolean } = $props();

	const userService = new UserService();
	let displayNameEdited =
		!!user.displayName &&
		user.displayName !== [user.firstName, user.lastName].filter(Boolean).join(' ');

	const formSchema = z.object({
		firstName: z.string().max(50),
		lastName: emptyToUndefined(z.string().max(50)),
		displayName: z.string().max(100),
		username: usernameSchema,
		email: get(appConfigStore).requireUserEmail ? z.email() : emptyToUndefined(z.email()),
		emailVerified: z.boolean(),
		isAdmin: z.boolean()
	});

	const formStore = createForm(formSchema, {
		firstName: user.firstName,
		lastName: user.lastName ?? '',
		displayName: user.displayName,
		username: user.username,
		email: user.email ?? '',
		emailVerified: user.emailVerified,
		isAdmin: user.isAdmin
	});
	const { inputs } = formStore;

	function onNameInput() {
		if (displayNameEdited) return;
		$inputs.displayName.value = [$inputs.firstName.value, $inputs.lastName?.value]
			.filter(Boolean)
			.join(' ')
			.trim();
	}

	trackFormChanges(
		() => formStore,
		async (data) => {
			// Disabling is an action of its own, so the saved state is sent along unchanged
			const saved = await userService.update(user.id, { ...data, disabled: user.disabled });
			// Groups and claims are edited elsewhere on the page and may hold unsaved edits, so only the profile is taken over
			Object.assign(user, {
				...saved,
				userGroups: user.userGroups,
				customClaims: user.customClaims
			});
		}
	);

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

<SettingsSection title={m.profile()}>
	<div class="flex flex-col gap-6 pt-4">
		<ProfilePictureSettings
			userId={user.id}
			isLdapUser={!!user.ldapId}
			updateCallback={updateProfilePicture}
			resetCallback={resetProfilePicture}
		/>
		{#if readOnly}
			<p class="text-caption text-fg-muted flex items-start gap-1.5">
				<LucideLock class="mt-1 size-3 shrink-0" />
				{m.profile_managed_by_ldap()}
			</p>
		{/if}
		<div class="grid gap-4 sm:grid-cols-2">
			<FormInput
				label={m.first_name()}
				optional
				bind:input={$inputs.firstName}
				onInput={onNameInput}
				readonly={readOnly}
			/>
			<FormInput
				label={m.last_name()}
				optional
				bind:input={$inputs.lastName}
				onInput={onNameInput}
				readonly={readOnly}
			/>
			<FormInput
				label={m.display_name()}
				optional
				description={readOnly ? undefined : m.display_name_follows_names()}
				bind:input={$inputs.displayName}
				onInput={() => (displayNameEdited = true)}
				readonly={readOnly}
			/>
			<FormInput label={m.username()} bind:input={$inputs.username} readonly={readOnly} />
			<div class="flex flex-col gap-3 sm:col-span-2 sm:max-w-[calc(50%-0.5rem)]">
				<FormInput
					label={m.email()}
					optional={!$appConfigStore.requireUserEmail}
					type="email"
					bind:input={$inputs.email}
					readonly={readOnly}
				/>
				{#if $appConfigStore.emailVerification !== 'disabled'}
					<label class="text-control text-fg flex items-center gap-3 font-medium">
						<Checkbox bind:checked={$inputs.emailVerified.value} disabled={readOnly} />
						{m.email_address_is_verified()}
					</label>
				{/if}
			</div>
		</div>
	</div>
</SettingsSection>

<SettingsSection title={m.access()}>
	<SettingRow
		title={m.admin_privileges()}
		description={m.admins_have_full_access_to_the_admin_panel()}
		icon={LucideShieldCheck}
		consequence={readOnly ? m.set_by_the_ldap_admin_group() : undefined}
	>
		{#snippet control(props)}
			<Switch {...props} bind:checked={$inputs.isAdmin.value} disabled={readOnly} />
		{/snippet}
	</SettingRow>
</SettingsSection>
