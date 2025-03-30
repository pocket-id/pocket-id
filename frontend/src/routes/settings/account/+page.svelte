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
		KeyRound,
		Languages,
		LucideAlertTriangle,
		RectangleEllipsis,
		UserCog
	} from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
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
	<Alert.Root variant="warning" class="flex gap-3">
		<LucideAlertTriangle class="size-4" />
		<div>
			<Alert.Title class="font-semibold">{m.passkey_missing()}</Alert.Title>
			<Alert.Description class="text-sm">
				{m.please_provide_a_passkey_to_prevent_losing_access_to_your_account()}
			</Alert.Description>
		</div>
	</Alert.Root>
{:else if passkeys.length == 1}
	<Alert.Root variant="warning" dismissibleId="single-passkey" class="flex gap-3">
		<LucideAlertTriangle class="size-4" />
		<div>
			<Alert.Title class="font-semibold">{m.single_passkey_configured()}</Alert.Title>
			<Alert.Description class="text-sm">
				{m.it_is_recommended_to_add_more_than_one_passkey()}
			</Alert.Description>
		</div>
	</Alert.Root>
{/if}

<!-- Account details card -->
<fieldset
	disabled={!$appConfigStore.allowOwnAccountEdit ||
		(!!account.ldapId && $appConfigStore.ldapEnabled)}
>
	<Card.Root>
		<Card.Header>
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
			/>
		</Card.Content>
	</Card.Root>
</fieldset>

<!-- Passkey management card -->
<div>
	<Card.Root>
		<Card.Header>
			<div class="flex items-center justify-between">
				<div>
					<Card.Title>
						<KeyRound class="text-primary/80 h-5 w-5" />
						{m.passkeys()}
					</Card.Title>
					<Card.Description>
						{m.manage_your_passkeys_that_you_can_use_to_authenticate_yourself()}
					</Card.Description>
				</div>
				<Button variant="outline" class="ml-3" on:click={createPasskey}>
					{m.add_passkey()}
				</Button>
			</div>
		</Card.Header>
		{#if passkeys.length != 0}
			<Card.Content>
				<PasskeyList bind:passkeys />
			</Card.Content>
		{/if}
	</Card.Root>
</div>

<!-- Login code card -->
<div>
	<Card.Root>
		<Card.Header>
			<div class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
				<div>
					<Card.Title>
						<RectangleEllipsis class="text-primary/80 h-5 w-5" />
						{m.login_code()}
					</Card.Title>
					<Card.Description>
						{m.create_a_one_time_login_code_to_sign_in_from_a_different_device_without_a_passkey()}
					</Card.Description>
				</div>
				<Button variant="outline" on:click={() => (showLoginCodeModal = true)}>
					{m.create()}
				</Button>
			</div>
		</Card.Header>
	</Card.Root>
</div>

<!-- Language selection card -->
<div>
	<Card.Root>
		<Card.Header>
			<div class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
				<div>
					<Card.Title>
						<Languages class="text-primary/80 h-5 w-5" />
						{m.language()}
					</Card.Title>

					<Card.Description>
						{m.select_the_language_you_want_to_use()}
					</Card.Description>
				</div>
				<LocalePicker />
			</div>
		</Card.Header>
	</Card.Root>
</div>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
<LoginCodeModal bind:show={showLoginCodeModal} />
