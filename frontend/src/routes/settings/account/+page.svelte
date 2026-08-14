<script lang="ts">
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Item from '$lib/components/ui/item/index.js';
	import { m } from '$lib/paraglide/messages';
	import RuntimeCredentialService from '$lib/services/runtime-credential-service';
	import UserService from '$lib/services/user-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import type { Passkey } from '$lib/types/passkey.type';
	import type { RuntimeCredential } from '$lib/types/runtime-credential.type';
	import type { AccountUpdate } from '$lib/types/user.type';
	import { axiosErrorToast, getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { Bot, KeyRound, Languages, LucideAlertTriangle, UserCog } from '@lucide/svelte';
	import { startRegistration } from '@simplewebauthn/browser';
	import { toast } from 'svelte-sonner';
	import AccountForm from './account-form.svelte';
	import LocalePicker from './locale-picker.svelte';
	import PasskeyList from './passkey-list.svelte';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';
	import RenameRuntimeCredentialModal from './rename-runtime-credential-modal.svelte';
	import RuntimeCredentialList from './runtime-credential-list.svelte';

	let { data } = $props();
	let account = $state(data.account);
	let passkeys = $state(data.passkeys);
	let runtimeCredentials: RuntimeCredential[] = $state(data.runtimeCredentials);
	let runtimeCredentialToRename: RuntimeCredential | null = $state(null);
	let passkeyToRename: Passkey | null = $state(null);
	const userService = new UserService();
	const webauthnService = new WebAuthnService();

	const userInfoInputDisabled = $derived(
		!$appConfigStore.allowOwnAccountEdit || (!!account.ldapId && $appConfigStore.ldapEnabled)
	);

	async function updateAccount(user: AccountUpdate) {
		let success = true;
		await userService
			.updateCurrent(user)
			.then((user) => {
				toast.success(m.account_details_updated_successfully());
				userStore.setUser(user);
			})
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

{#if account.isAgent && !runtimeCredentials.some((credential) => !credential.revokedAt)}
	<Alert.Root variant="warning" class="flex gap-3">
		<LucideAlertTriangle class="size-4" />
		<div>
			<Alert.Title class="font-semibold">{m.runtime_credential_missing()}</Alert.Title>
			<Alert.Description class="text-sm">
				{m.request_a_new_one_time_link_to_register_a_runtime_credential()}
			</Alert.Description>
		</div>
	</Alert.Root>
{:else if !account.isAgent && passkeys.length == 0}
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
{:else if !account.isAgent && passkeys.length == 1}
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

<!-- FCA13 switches account credential management between passkeys and runtime credentials without changing account capabilities -->
<Item.Group class="bg-card border shadow-sm rounded-4xl p-5">
	<Item.Root class="border-none bg-transparent p-0">
		<Item.Media class="text-primary/80">
			{#if account.isAgent}
				<Bot class="size-5" />
			{:else}
				<KeyRound class="size-5" />
			{/if}
		</Item.Media>
		<Item.Content class="min-w-52">
			<Item.Title class="text-xl font-semibold">
				{account.isAgent ? m.runtime_credentials() : m.passkeys()}
			</Item.Title>
			<Item.Description>
				{account.isAgent
					? m.manage_your_runtime_credentials()
					: m.manage_your_passkeys_that_you_can_use_to_authenticate_yourself()}
			</Item.Description>
		</Item.Content>
		{#if !account.isAgent}
			<Item.Actions>
				<Button variant="outline" onclick={createPasskey}>
					{m.add_passkey()}
				</Button>
			</Item.Actions>
		{/if}
	</Item.Root>
	{#if account.isAgent && runtimeCredentials.length != 0}
		<RuntimeCredentialList
			bind:credentials={runtimeCredentials}
			onRename={(credential) => (runtimeCredentialToRename = credential)}
		/>
	{:else if !account.isAgent && passkeys.length != 0}
		<PasskeyList bind:passkeys />
	{/if}
</Item.Group>

<Item.Root variant="card" class="border-border mb-2">
	<Item.Media class="text-primary/80">
		<Languages class="size-5" />
	</Item.Media>
	<Item.Content class="min-w-52">
		<Item.Title>{m.language()}</Item.Title>
		<Item.Description>
			{m.select_the_language_you_want_to_use()}
			<br />
			<FormattedMessage message={m.contribute_to_translation} />
		</Item.Description>
	</Item.Content>
	<Item.Actions>
		<LocalePicker />
	</Item.Actions>
</Item.Root>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>

<RenameRuntimeCredentialModal
	bind:credential={runtimeCredentialToRename}
	callback={async () => (runtimeCredentials = await new RuntimeCredentialService().list())}
/>
