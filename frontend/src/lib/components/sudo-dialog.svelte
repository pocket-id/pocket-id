<script lang="ts">
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		authPrimaryClass,
		authQuietClass,
		isPasskeyPromptClosed,
		successHold
	} from '#lib/components/auth/auth-util.ts';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import { sudoPrompt } from '#lib/stores/request-prompt-store.svelte.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import * as AlertDialog from '@pocket-id/ui/alert-dialog';
	import { Button } from '@pocket-id/ui/button';
	import { startAuthentication } from '@simplewebauthn/browser';

	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let success = $state(false);
	let failure: { message: string; closed: boolean } | undefined = $state();

	// Each time the dialog opens it starts over, but closing keeps the outcome so it doesn't flicker while fading out
	$effect(() => {
		if (sudoPrompt.open) {
			success = false;
			failure = undefined;
		}
	});

	async function confirm() {
		failure = undefined;
		isLoading = true;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication({ optionsJSON: loginOptions });
			await webauthnService.reauthenticate(authResponse);
			success = true;
			await successHold();
			sudoPrompt.finish(true);
		} catch (e) {
			failure = { message: getWebauthnErrorMessage(e), closed: isPasskeyPromptClosed(e) };
		}
		isLoading = false;
	}
</script>

<!-- It sits above the dialog or sheet the change was made in, which its scrim dims too, and only confirming or cancelling closes it -->
<AlertDialog.Root bind:open={() => sudoPrompt.open, () => {}}>
	<AlertDialog.Content class="z-9999 gap-0" overlayClass="z-9999">
		<div class="flex flex-col items-center text-center">
			<AuthTile {success} error={!!failure} />
			<AlertDialog.Title class="mb-2">
				{success ? m.sudo_confirmed() : m.auth_confirm_its_you()}
			</AlertDialog.Title>
			<AlertDialog.Description class="text-body mb-6" aria-live="polite">
				{#if success}
					{m.sudo_finishing()}
				{:else if failure}
					{failure.closed ? m.auth_passkey_prompt_closed() : failure.message}
				{:else}
					{m.sudo_description()}
				{/if}
			</AlertDialog.Description>
		</div>
		{#if $userStore}
			<div class="mb-6">
				<IdentityRow user={$userStore} />
			</div>
		{/if}
		<div class="flex flex-col items-center gap-1">
			<Button class={authPrimaryClass} isLoading={isLoading || success} onclick={confirm}>
				{failure ? m.try_again() : m.auth_continue_with_passkey()}
			</Button>
			<Button variant="link" class={authQuietClass} onclick={() => sudoPrompt.finish(false)}>
				{m.cancel()}
			</Button>
		</div>
	</AlertDialog.Content>
</AlertDialog.Root>
