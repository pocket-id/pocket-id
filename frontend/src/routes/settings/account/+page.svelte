<script lang="ts">
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Item from '$lib/components/ui/item/index.js';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { Passkey } from '$lib/types/passkey.type';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast, getWebauthnErrorMessage } from '$lib/utils/error-util';
	import {
		KeyRound,
		Languages,
		LucideAlertTriangle,
		RectangleEllipsis,
		UserCog
	} from '@lucide/svelte';
	import { startRegistration } from '@simplewebauthn/browser';
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

	const userInfoInputDisabled = $derived(
		!$appConfigStore.allowOwnAccountEdit || (!!account.ldapId && $appConfigStore.ldapEnabled)
	);

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
			const attResp = await startRegistration({ optionsJSON: opts });
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
		<div class="md:flex md:w-full md:place-content-between">
			<div>
				<Alert.Title class="font-semibold">{m.passkey_missing()}</Alert.Title>
				<Alert.Description class="text-sm">
					{m.please_provide_a_passkey_to_prevent_losing_access_to_your_account()}
				</Alert.Description>
			</div>
			<div>
				<Button class="mt-2 md:mt-0" onclick={createPasskey}>
					{m.add_passkey()}
				</Button>
			</div>
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

<!-- Login code card mobile -->
<div class="block sm:hidden">
	<Item.Root variant="outline">
		<Item.Media class="text-primary/80">
			<RectangleEllipsis class="size-5" />
		</Item.Media>
		<Item.Content>
			<Item.Title>{m.login_code()}</Item.Title>
			<Item.Description>
				{m.create_a_one_time_login_code_to_sign_in_from_a_different_device_without_a_passkey()}
			</Item.Description>
		</Item.Content>
		<Item.Actions class="w-full sm:w-auto">
			<Button variant="outline" class="w-full" onclick={() => (showLoginCodeModal = true)}>
				{m.create()}
			</Button>
		</Item.Actions>
	</Item.Root>
</div>

<Card.Root>
	<Card.Header>
		<Card.Title>
			<UserCog class="text-primary/80 size-5" />
			{m.account_details()}
		</Card.Title>
	</Card.Header>
	<Card.Content>
		<AccountForm
			{account}
			userId={account.id}
			callback={updateAccount}
			isLdapUser={!!account.ldapId}
			{userInfoInputDisabled}
		/>
	</Card.Content>
</Card.Root>

<Item.Group class="bg-muted/50 rounded-xl border p-4">
	<Item.Root class="border-none bg-transparent p-0">
		<Item.Media class="text-primary/80">
			<KeyRound class="size-5" />
		</Item.Media>
		<Item.Content>
			<Item.Title class="text-xl font-semibold">{m.passkeys()}</Item.Title>
			<Item.Description>
				{m.manage_your_passkeys_that_you_can_use_to_authenticate_yourself()}
			</Item.Description>
		</Item.Content>
		<Item.Actions>
			<Button variant="outline" onclick={createPasskey}>
				{m.add_passkey()}
			</Button>
		</Item.Actions>
	</Item.Root>
	{#if passkeys.length != 0}
		<Item.Separator class="my-4" />
		<PasskeyList bind:passkeys />
	{/if}
</Item.Group>

<div class="hidden sm:block">
	<Item.Root variant="muted" class="border-border">
		<Item.Media class="text-primary/80">
			<RectangleEllipsis class="size-5" />
		</Item.Media>
		<Item.Content>
			<Item.Title>{m.login_code()}</Item.Title>
			<Item.Description>
				{m.create_a_one_time_login_code_to_sign_in_from_a_different_device_without_a_passkey()}
			</Item.Description>
		</Item.Content>
		<Item.Actions>
			<Button variant="outline" onclick={() => (showLoginCodeModal = true)}>
				{m.create()}
			</Button>
		</Item.Actions>
	</Item.Root>
</div>

<div>
	<Item.Root variant="muted" class="border-border">
		<Item.Media class="text-primary/80">
			<Languages class="size-5" />
		</Item.Media>
		<Item.Content>
			<Item.Title>{m.language()}</Item.Title>
			<Item.Description>
				{m.select_the_language_you_want_to_use()}
				<br />
				<FormattedMessage m={m.contribute_to_translation()} />
			</Item.Description>
		</Item.Content>
		<Item.Actions>
			<LocalePicker />
		</Item.Actions>
	</Item.Root>
</div>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
<LoginCodeModal bind:show={showLoginCodeModal} />
