<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import ProfilePictureSettings from '#lib/components/form/profile-picture-settings.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { AccountUpdate, EmailChange, User } from '#lib/types/user.type.ts';
	import { isEmailVerificationPending } from '#lib/utils/email-verification-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { trackFormChanges } from '#lib/utils/unsaved-changes-util.svelte.ts';
	import { emptyToUndefined, usernameSchema } from '#lib/utils/zod-util.ts';
	import { LucideInfo, LucideSend, LucideUndo2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { get } from 'svelte/store';
	import { z } from 'zod/v4';

	let {
		account,
		pendingEmailChange: initialPendingEmailChange
	}: {
		account: User;
		pendingEmailChange: EmailChange | null;
	} = $props();

	const userService = new UserService();
	const emailId = $props.id();

	let pendingEmailChange = $state(initialPendingEmailChange);
	let hasManualDisplayNameEdit = $state(
		!!account.displayName &&
			account.displayName !== [account.firstName, account.lastName].filter(Boolean).join(' ')
	);

	const readOnlyReason = $derived(
		account.ldapId && $appConfigStore.ldapEnabled
			? m.profile_managed_by_ldap()
			: !$appConfigStore.allowOwnAccountEdit
				? m.profile_editing_disabled()
				: undefined
	);

	const emailStatus = $derived.by(() => {
		if ($appConfigStore.emailVerification === 'disabled' || !$userStore?.email) return undefined;
		if ($userStore.emailVerified)
			return { variant: 'success' as const, label: m.email_status_verified() };
		if (isEmailVerificationPending($userStore, $appConfigStore)) {
			return { variant: 'destructive' as const, label: m.email_status_verification_required() };
		}
		return { variant: 'warning' as const, label: m.email_status_unverified() };
	});

	const formSchema = z.object({
		firstName: z.string().max(50),
		lastName: emptyToUndefined(z.string().max(50).optional()),
		displayName: z.string().max(100),
		username: usernameSchema,
		email: get(appConfigStore).requireUserEmail ? z.email() : emptyToUndefined(z.email().optional())
	});
	type FormSchema = typeof formSchema;

	const formStore = createForm<FormSchema>(formSchema, {
		...account,
		email: account.email || ''
	});
	const { inputs } = formStore;

	function onNameInput() {
		if (!hasManualDisplayNameEdit) {
			$inputs.displayName.value = `${$inputs.firstName.value}${
				$inputs.lastName?.value ? ' ' + $inputs.lastName.value : ''
			}`.trim();
		}
	}

	trackFormChanges(
		() => formStore,
		async (data: AccountUpdate) => {
			const saved = await userService.updateCurrent(data);
			userStore.setUser(saved);

			// A verified email is only replaced once the new address is confirmed, so keep showing the email that is still in effect
			// The saved data becomes the new baseline, so it has to carry that email as well
			if (saved.email && data.email && saved.email !== data.email) {
				pendingEmailChange = await userService.getPendingEmailChange().catch(() => null);
				data.email = saved.email;
				formStore.setValue('email', saved.email);
			}
		}
	);

	async function cancelEmailChange() {
		await userService
			.cancelEmailChange()
			.then(() => {
				pendingEmailChange = null;
				toast.success(m.email_change_canceled());
			})
			.catch(axiosErrorToast);
	}

	async function sendEmailVerification() {
		await userService
			.sendEmailVerification()
			.then(() => toast.success(m.email_verification_sent()))
			.catch(axiosErrorToast);
	}

	async function updateProfilePicture(image: File) {
		await userService
			.updateCurrentUsersProfilePicture(image)
			.then(() => toast.success(m.profile_picture_updated_successfully()))
			.catch(axiosErrorToast);
	}

	async function resetProfilePicture() {
		await userService
			.resetCurrentUserProfilePicture()
			.then(() => toast.success(m.profile_picture_has_been_reset()))
			.catch(axiosErrorToast);
	}
</script>

<SettingsSection
	title={m.profile()}
	description={m.profile_section_description({ appName: $appConfigStore.appName })}
>
	<div class="flex flex-col gap-6 pt-4">
		<ProfilePictureSettings
			userId={account.id}
			isLdapUser={!!account.ldapId}
			updateCallback={updateProfilePicture}
			resetCallback={resetProfilePicture}
		/>

		<div class="flex flex-col gap-3">
			{#if readOnlyReason}
				<p class="text-caption text-fg-muted flex items-start gap-1.5">
					<LucideInfo class="mt-1 size-3 shrink-0" />
					{readOnlyReason}
				</p>
			{/if}
			<form class="grid gap-4 sm:grid-cols-2">
				<FormInput
					label={m.first_name()}
					optional
					bind:input={$inputs.firstName}
					onInput={onNameInput}
					readonly={!!readOnlyReason}
				/>
				<FormInput
					label={m.last_name()}
					optional
					bind:input={$inputs.lastName}
					onInput={onNameInput}
					readonly={!!readOnlyReason}
				/>
				<FormInput
					label={m.display_name()}
					optional
					description={readOnlyReason ? undefined : m.display_name_helper()}
					bind:input={$inputs.displayName}
					onInput={() => (hasManualDisplayNameEdit = true)}
					readonly={!!readOnlyReason}
				/>
				<FormInput label={m.username()} bind:input={$inputs.username} readonly={!!readOnlyReason} />
				<FormInput
					label={m.email()}
					labelFor={emailId}
					optional={!$appConfigStore.requireUserEmail}
					readonly={!!readOnlyReason}
					class="sm:col-span-2"
				>
					<!-- The status is the field's helper line, and on phones it sits between the field and its action -->
					<div class="grid gap-x-4 gap-y-1.5 sm:grid-cols-2 sm:items-center">
						<Input
							id={emailId}
							type="email"
							autocomplete="email"
							aria-invalid={!!$inputs.email.error}
							aria-describedby="{emailId}-helper"
							readonly={!!readOnlyReason}
							bind:value={
								() => $inputs.email.value,
								(value) => {
									$inputs.email.value = value;
									$inputs.email.error = null;
								}
							}
						/>
						{#if $inputs.email.error}
							<Field.Error id="{emailId}-helper" class="sm:row-start-2">
								{$inputs.email.error}
							</Field.Error>
						{:else}
							<p
								id="{emailId}-helper"
								class="text-caption text-fg-muted wrap-anywhere sm:row-start-2"
							>
								{#if pendingEmailChange}
									{m.email_change_waiting({ email: pendingEmailChange.email })}
								{:else if emailStatus}
									<Badge variant={emailStatus.variant} class="text-caption">
										{emailStatus.label}
									</Badge>
								{:else}
									{m.email_helper_purpose()}
								{/if}
							</p>
						{/if}
						{#if pendingEmailChange}
							<div class="sm:col-start-2 sm:row-start-1">
								<Button variant="secondary" size="sm" usePromiseLoading onclick={cancelEmailChange}>
									<LucideUndo2 />
									{m.cancel_email_change()}
								</Button>
							</div>
						{:else if emailStatus && !$userStore?.emailVerified}
							<!-- An edited address is not saved yet, so the button waits instead of leaving and shifting the form on phones -->
							<div class="sm:col-start-2 sm:row-start-1">
								<Button
									variant="secondary"
									size="sm"
									usePromiseLoading
									disabled={$inputs.email.value !== $userStore?.email}
									onclick={sendEmailVerification}
								>
									<LucideSend />
									{m.send_email()}
								</Button>
							</div>
						{/if}
					</div>
				</FormInput>
			</form>
		</div>
	</div>
</SettingsSection>
