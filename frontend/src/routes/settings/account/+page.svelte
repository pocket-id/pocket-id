<script lang="ts">
	import EmailVerificationStateBox from '#lib/components/email-verification-state-box.svelte';
	import ProfilePictureSettings from '#lib/components/form/profile-picture-settings.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import HashTabs from '#lib/components/hash-tabs.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import * as Tabs from '@pocket-id/ui/tabs';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { isEmailVerificationPending } from '#lib/utils/email-verification-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import LanguagesIcon from '~icons/solar/translation-2-bold-duotone';
	import { formatDistanceToNow } from 'date-fns';
	import { toast } from 'svelte-sonner';
	import AccountForm from './account-form.svelte';
	import ApiKeysRow from './api-keys-row.svelte';
	import LocalePicker from './locale-picker.svelte';
	import PasskeyList from './passkey-list.svelte';
	import TaskBanner, { type AccountTask } from './task-banner.svelte';

	let { data } = $props();

	const userService = new UserService();

	let tab = $state('profile');
	let passkeys = $state(data.passkeys);
	let passkeyList = $state<ReturnType<typeof PasskeyList>>();

	const user = $derived($userStore ?? data.account);
	const emailRequired = $derived(isEmailVerificationPending(user, $appConfigStore));

	const tasks = $derived.by(() => {
		const list: AccountTask[] = [];
		if (
			$appConfigStore.emailVerification !== 'disabled' &&
			!user.emailVerified &&
			(user.email || emailRequired)
		) {
			const deadline = user.emailVerificationDeadline;
			list.push({
				id: 'email',
				tone: emailRequired ? 'destructive' : 'warning',
				title: emailRequired
					? m.email_verification_required_warning()
					: m.email_verification_warning(),
				description: emailRequired
					? [
							m.email_verification_only_page(),
							deadline &&
								m.email_verification_deadline_description({
									time: formatDistanceToNow(deadline, { addSuffix: true })
								})
						]
							.filter(Boolean)
							.join(' ')
					: m.email_verification_link_hint({ email: user.email ?? '' }),
				action: user.email ? { label: m.send_email(), run: sendEmailVerification } : undefined
			});
		}
		if (passkeys.length === 0) {
			list.push({
				id: 'passkey',
				tone: 'destructive',
				title: m.add_a_passkey(),
				description: m.auth_skip_passkey_description(),
				action: { label: m.add_passkey(), run: () => passkeyList!.createPasskey() }
			});
		} else if (passkeys.length === 1) {
			list.push({
				id: 'passkey',
				tone: 'warning',
				title: m.add_backup_passkey(),
				description: m.add_backup_passkey_description({ name: passkeys[0].name }),
				action: { label: m.add_passkey(), run: () => passkeyList!.createPasskey() }
			});
		}
		// The banner shows the first task, so blocking ones go ahead of warnings
		return list.sort((a, b) => Number(b.tone === 'destructive') - Number(a.tone === 'destructive'));
	});

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

<svelte:head>
	<title>{m.my_account()}</title>
</svelte:head>

<div class="flex flex-col gap-8">
	<header class="flex items-center gap-4 sm:gap-6">
		<ProfilePictureSettings
			userId={data.account.id}
			isLdapUser={!!data.account.ldapId}
			layout="avatar"
			class="size-20"
			updateCallback={updateProfilePicture}
			resetCallback={resetProfilePicture}
		/>
		<div class="flex min-w-0 flex-col gap-1">
			<p class="text-caption text-fg-muted">{m.my_account()}</p>
			<h1 class="font-display text-display-m text-fg max-sm:text-display-s wrap-anywhere">
				{user.displayName}
			</h1>
			<p
				class="text-control text-fg-muted flex flex-wrap items-baseline gap-x-2 gap-y-1 wrap-anywhere"
			>
				<span>@{user.username}</span>
				{#if user.email}
					<span class="min-w-0 max-sm:basis-full"
						><span aria-hidden="true" class="me-2 max-sm:hidden">·</span>{user.email}</span
					>
				{/if}
			</p>
		</div>
	</header>

	<!-- Svelte leaves anchor nodes behind, so :empty never matches and the wrapper hides when it holds no element -->
	<div class="flex flex-col gap-3 not-has-[*]:hidden">
		<EmailVerificationStateBox />
		<TaskBanner {tasks} />
	</div>

	<!-- Inactive panels stay mounted, so unsaved profile edits survive a look at another tab -->
	<HashTabs bind:value={tab} class="gap-10">
		<div class="shadow-[inset_0_-1px_0_var(--border)]">
			<div class="scroll-fade-x [scrollbar-width:none] overflow-x-auto">
				<Tabs.List variant="line" class="shadow-none max-sm:gap-4" aria-label={m.my_account()}>
					<Tabs.Trigger value="profile">{m.profile()}</Tabs.Trigger>
					<Tabs.Trigger value="passkeys">
						{m.passkeys()}
						<span class="text-fg-muted tabular-nums">{passkeys.length}</span>
					</Tabs.Trigger>
					<Tabs.Trigger value="preferences">{m.preferences()}</Tabs.Trigger>
					{#if data.features?.apiKeys.enabled}
						<Tabs.Trigger value="developers">{m.developers()}</Tabs.Trigger>
					{/if}
				</Tabs.List>
			</div>
		</div>

		<Tabs.Content value="profile">
			<AccountForm account={data.account} pendingEmailChange={data.pendingEmailChange} />
		</Tabs.Content>

		<Tabs.Content value="passkeys">
			<PasskeyList bind:this={passkeyList} bind:passkeys />
		</Tabs.Content>

		<!-- Rows bring their own vertical padding, so the first one is pulled up until its text starts at the panel top like the Profile form -->
		<Tabs.Content value="preferences" class="*:first:-mt-4">
			<SettingRow title={m.language()} icon={LanguagesIcon}>
				{#snippet description()}
					<FormattedMessage message={m.language_crowdin_description} />
				{/snippet}
				{#snippet control(props)}
					<LocalePicker {...props} />
				{/snippet}
			</SettingRow>
		</Tabs.Content>

		{#if data.features?.apiKeys.enabled}
			<Tabs.Content value="developers" class="*:first:-mt-4">
				<ApiKeysRow />
			</Tabs.Content>
		{/if}
	</HashTabs>
</div>
