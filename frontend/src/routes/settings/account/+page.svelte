<script lang="ts">
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import UserService from '$lib/services/user-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Passkey } from '$lib/types/passkey.type';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast, getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startRegistration } from '@simplewebauthn/browser';
	import { LucideAlertTriangle } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import ProfilePictureSettings from '../../../lib/components/form/profile-picture-settings.svelte';
	import AccountForm from './account-form.svelte';
	import LoginCodeModal from './login-code-modal.svelte';
	import PasskeyList from './passkey-list.svelte';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';
	import { m } from '$lib/paraglide/messages';

	let { data } = $props();
	let account = $state(data.account);
	let passkeys = $state(data.passkeys);
	let passkeyToRename: Passkey | null = $state(null);
	let showLoginCodeModal: boolean = $state(false);

	const userService = new UserService();
	const webauthnService = new WebAuthnService();

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
	<Alert.Root variant="warning">
		<LucideAlertTriangle class="size-4" />
		<Alert.Title>{m.passkey_missing()}</Alert.Title>
		<Alert.Description
			>{m.please_provide_a_passkey_to_prevent_losing_access_to_your_account()}</Alert.Description
		>
	</Alert.Root>
{:else if passkeys.length == 1}
	<Alert.Root variant="warning" dismissibleId="single-passkey">
		<LucideAlertTriangle class="size-4" />
		<Alert.Title>{m.single_passkey_configured()}</Alert.Title>
		<Alert.Description
			>{m.it_is_recommended_to_add_more_than_one_passkey()}</Alert.Description
		>
	</Alert.Root>
{/if}

<fieldset
	disabled={!$appConfigStore.allowOwnAccountEdit ||
		(!!account.ldapId && $appConfigStore.ldapEnabled)}
>
	<Card.Root>
		<Card.Header>
			<Card.Title>{m.account_details()}</Card.Title>
		</Card.Header>
		<Card.Content>
			<AccountForm {account} callback={updateAccount} />
		</Card.Content>
	</Card.Root>
</fieldset>

<Card.Root>
	<Card.Content class="pt-6">
		<ProfilePictureSettings
			userId={account.id}
			isLdapUser={!!account.ldapId}
			callback={updateProfilePicture}
		/>
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>{m.passkeys()}</Card.Title>
				<Card.Description class="mt-1">
					{m.manage_your_passkeys_that_you_can_use_to_authenticate_yourself()}
				</Card.Description>
			</div>
			<Button size="sm" class="ml-3" on:click={createPasskey}>{m.add_passkey()}</Button>
		</div>
	</Card.Header>
	{#if passkeys.length != 0}
		<Card.Content>
			<PasskeyList bind:passkeys />
		</Card.Content>
	{/if}
</Card.Root>

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>{m.login_code()}</Card.Title>
				<Card.Description class="mt-1">
					{m.create_a_one_time_login_code_to_sign_in_from_a_different_device_without_a_passkey()}
				</Card.Description>
			</div>
			<Button size="sm" class="ml-auto" on:click={() => (showLoginCodeModal = true)}>{m.create()}</Button>
		</div>
	</Card.Header>
</Card.Root>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
<LoginCodeModal bind:show={showLoginCodeModal} />
