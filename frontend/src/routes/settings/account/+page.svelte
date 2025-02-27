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
	import AccountForm from './account-form.svelte';
	import PasskeyList from './passkey-list.svelte';
	import ProfilePictureSettings from '../../../lib/components/form/profile-picture-settings.svelte';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';
	import OneTimeLinkModal from '$lib/components/one-time-link-modal.svelte';

	let { data } = $props();
	let account = $state(data.account);
	let passkeys = $state(data.passkeys);
	let passkeyToRename: Passkey | null = $state(null);
	let showOneTimeLinkModal: boolean | null = $state(null);

	const userService = new UserService();
	const webauthnService = new WebAuthnService();

	async function updateAccount(user: UserCreate) {
		let success = true;
		await userService
			.updateCurrent(user)
			.then(() => toast.success('Account details updated successfully'))
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
	}

	async function updateProfilePicture(image: File) {
		await userService
			.updateCurrentUsersProfilePicture(image)
			.then(() => toast.success('Profile picture updated successfully'))
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
	<title>Account Settings</title>
</svelte:head>

{#if passkeys.length == 0}
	<Alert.Root variant="warning">
		<LucideAlertTriangle class="size-4" />
		<Alert.Title>Passkey missing</Alert.Title>
		<Alert.Description
			>Please add a passkey to prevent losing access to your account.</Alert.Description
		>
	</Alert.Root>
{:else if passkeys.length == 1}
	<Alert.Root variant="warning" dismissibleId="single-passkey">
		<LucideAlertTriangle class="size-4" />
		<Alert.Title>Single Passkey Configured</Alert.Title>
		<Alert.Description
			>It is recommended to add more than one passkey to avoid loosing access to your account.</Alert.Description
		>
	</Alert.Root>
{/if}

<fieldset
	disabled={!$appConfigStore.allowOwnAccountEdit ||
		(!!account.ldapId && $appConfigStore.ldapEnabled)}
>
	<Card.Root>
		<Card.Header>
			<Card.Title>Account Details</Card.Title>
		</Card.Header>
		<Card.Content>
			<AccountForm {account} callback={updateAccount} />
		</Card.Content>
	</Card.Root>
</fieldset>

<Card.Root>
	<Card.Content class="pt-6">
		<ProfilePictureSettings
			userId="me"
			isLdapUser={!!account.ldapId}
			callback={updateProfilePicture}
		/>
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>Passkeys</Card.Title>
				<Card.Description class="mt-1">
					Manage your passkeys that you can use to authenticate yourself.
				</Card.Description>
			</div>
			<Button size="sm" class="ml-auto" on:click={() => (showOneTimeLinkModal = true)}
				>One-time Link</Button
			>
			<Button size="sm" class="ml-3" on:click={createPasskey}>Add Passkey</Button>
		</div>
	</Card.Header>
	{#if passkeys.length != 0}
		<Card.Content>
			<PasskeyList bind:passkeys />
		</Card.Content>
	{/if}
</Card.Root>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
<OneTimeLinkModal show={showOneTimeLinkModal} />
