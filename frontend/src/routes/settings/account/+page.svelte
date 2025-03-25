<script lang="ts">
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Passkey } from '$lib/types/passkey.type';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast, getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startRegistration } from '@simplewebauthn/browser';
	import {
		LucideAlertTriangle,
		Languages,
		UserCog,
		Image,
		RectangleEllipsis,
		KeyRound,
		ShieldPlus,
		Plus
	} from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import ProfilePictureSettings from '../../../lib/components/form/profile-picture-settings.svelte';
	import AccountForm from './account-form.svelte';
	import LocalePicker from './locale-picker.svelte';
	import LoginCodeModal from './login-code-modal.svelte';
	import PasskeyList from './passkey-list.svelte';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';

	let { data } = $props();
	let account = $state(data.account);
	let passkeys = $state(data.passkeys);
	let passkeyToRename: Passkey | null = $state(null);
	let showLoginCodeModal: boolean = $state(false);
	let mounted = $state(false);

	const userService = new UserService();
	const webauthnService = new WebAuthnService();

	async function resetProfilePicture() {
		await userService
			.resetCurrentUserProfilePicture()
			.then(() =>
				toast.success('Profile picture has been reset. It may take a few minutes to update.')
			)
			.catch(axiosErrorToast);
	}

	async function updateAccount(user: UserCreate) {
		let success = true;
		await userService
			.updateCurrent(user)
			.then(() => toast.success(m.account_details_updated_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
	}

	async function updateProfilePicture(image: File) {
		await userService
			.updateCurrentUsersProfilePicture(image)
			.then(() => toast.success(m.profile_picture_updated_successfully()))
			.catch(axiosErrorToast);
	}

	async function createPasskey() {
		try {
			const opts = await webauthnService.getRegistrationOptions();
			const attResp = await startRegistration(opts);
			const passkey = await webauthnService.finishRegistration(attResp);

			passkeys = await webauthnService.listCredentials();
			passkeyToRename = passkey;
		} catch (e) {
			toast.error(getWebauthnErrorMessage(e));
		}
	}
</script>

<svelte:head>
	<title>{m.account_settings()}</title>
</svelte:head>

{#if passkeys.length == 0}
	<div class="animate-fade-in" style="animation-delay: 100ms;">
		<Alert.Root variant="warning" class="flex gap-3">
			<LucideAlertTriangle class="size-4" />
			<div>
				<Alert.Title class="font-semibold">{m.passkey_missing()}</Alert.Title>
				<Alert.Description class="text-sm">
					{m.please_provide_a_passkey_to_prevent_losing_access_to_your_account()}
				</Alert.Description>
			</div>
		</Alert.Root>
	</div>
{:else if passkeys.length == 1}
	<div class="animate-fade-in" style="animation-delay: 100ms;">
		<Alert.Root variant="warning" dismissibleId="single-passkey" class="flex gap-3">
			<LucideAlertTriangle class="size-4" />
			<div>
				<Alert.Title class="font-semibold">{m.single_passkey_configured()}</Alert.Title>
				<Alert.Description class="text-sm">
					{m.it_is_recommended_to_add_more_than_one_passkey()}
				</Alert.Description>
			</div>
		</Alert.Root>
	</div>
{/if}

<!-- Account details card -->
<fieldset
	disabled={!$appConfigStore.allowOwnAccountEdit ||
		(!!account.ldapId && $appConfigStore.ldapEnabled)}
	class="animate-fade-in"
	style="animation-delay: 150ms;"
>
	<Card.Root>
		<Card.Header class="border-b">
			<Card.Title>
				<UserCog class="text-primary/80 h-5 w-5" />
				{m.account_details()}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<AccountForm
				{account}
				userId={account.id}
				callback={updateAccount}
				isLdapUser={!!account.ldapId}
				{updateProfilePicture}
				{resetProfilePicture}
			/>
		</Card.Content>
	</Card.Root>
</fieldset>

<!-- Profile picture card -->
<!-- LEAVING COMMENTED OUT TILL ELIAS CONFIRMS HE LIKES THIS -->
<!-- <div class="animate-fade-in mt-6" style="animation-delay: 200ms;">
		<Card.Root class="shadow-md transition-shadow duration-200 hover:shadow-lg">
			<Card.Header class="border-b">
				<Card.Title class="flex items-center gap-2 text-xl font-semibold">
					<Image class="text-primary/80 h-5 w-5" />
					{m.profile_picture()}
				</Card.Title>
			</Card.Header>
			<Card.Content class="pt-5">
				<ProfilePictureSettings
					userId={account.id}
					isLdapUser={!!account.ldapId}
					updateCallback={updateProfilePicture}
					resetCallback={resetProfilePicture}
				/>
			</Card.Content>
		</Card.Root>
	</div> -->

<!-- Passkey management card -->
<div class="animate-fade-in" style="animation-delay: 200ms;">
	<Card.Root>
		<Card.Header class="border-b">
			<div class="flex items-center justify-between">
				<Card.Title>
					<KeyRound class="text-primary/80 h-5 w-5" />
					{m.passkeys()}
				</Card.Title>
				<Button size="sm" variant="outline" class="ml-3 gap-1.5" on:click={createPasskey}>
					<Plus class="text-primary/80 h-5 w-5" />
					{m.add_passkey()}
				</Button>
			</div>
			<Card.Description>
				{m.manage_your_passkeys_that_you_can_use_to_authenticate_yourself()}
			</Card.Description>
		</Card.Header>
		{#if passkeys.length != 0}
			<Card.Content>
				<PasskeyList bind:passkeys />
			</Card.Content>
		{/if}
	</Card.Root>
</div>

<!-- Login code card -->
<div class="animate-fade-in" style="animation-delay: 250ms;">
	<Card.Root>
		<Card.Header>
			<div class="flex items-center justify-between">
				<Card.Title>
					<RectangleEllipsis class="text-primary/80 h-5 w-5" />
					{m.login_code()}
				</Card.Title>
				<Button
					size="sm"
					variant="outline"
					class="ml-auto gap-1.5"
					on:click={() => (showLoginCodeModal = true)}
				>
					<ShieldPlus class="text-primary/80 h-5 w-5" />
					{m.create()}
				</Button>
			</div>
			<Card.Description>
				{m.create_a_one_time_login_code_to_sign_in_from_a_different_device_without_a_passkey()}
			</Card.Description>
		</Card.Header>
	</Card.Root>
</div>

<!-- Language selection card -->
<div class="animate-fade-in" style="animation-delay: 300ms;">
	<Card.Root>
		<Card.Header>
			<div class="flex items-center justify-between">
				<Card.Title>
					<Languages class="text-primary/80 h-5 w-5" />
					{m.language()}
				</Card.Title>
				<LocalePicker />
			</div>
			<Card.Description>
				{m.select_the_language_you_want_to_use()}
			</Card.Description>
		</Card.Header>
	</Card.Root>
</div>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
<LoginCodeModal bind:show={showLoginCodeModal} />
