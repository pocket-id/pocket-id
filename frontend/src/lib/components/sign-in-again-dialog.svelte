<script lang="ts">
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		alternativeSignInHref,
		authPrimaryClass,
		authQuietClass,
		isPasskeyPromptClosed,
		successHold
	} from '#lib/components/auth/auth-util.ts';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import { signInAgain } from '#lib/stores/sign-in-again-store.svelte.ts';
	import unsavedChangesStore from '#lib/stores/unsaved-changes-store.svelte.ts';
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
		if (signInAgain.open) {
			success = false;
			failure = undefined;
		}
	});

	async function authenticate() {
		failure = undefined;
		isLoading = true;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication({ optionsJSON: loginOptions });
			const user = await webauthnService.finishLogin(authResponse);

			// The page holds the previous account's data, so another account starts over instead of replaying its requests
			// Its unsaved changes can't be saved by this account, so they're dropped rather than making the browser ask to leave
			if (user.id !== $userStore?.id) {
				unsavedChangesStore.discardAll();
				window.location.assign('/settings');
				return;
			}

			await userStore.setUser(user);
			success = true;
			await successHold();
			signInAgain.finish(true);
		} catch (e) {
			failure = { message: getWebauthnErrorMessage(e), closed: isPasskeyPromptClosed(e) };
		}
		isLoading = false;
	}
</script>

<!-- Only signing in or leaving for another way closes it, because the page can't do anything without a session -->
<AlertDialog.Root bind:open={() => signInAgain.open, () => {}}>
	<AlertDialog.Content class="z-9999 gap-0">
		<div class="flex flex-col items-center text-center">
			<AuthTile {success} error={!!failure} />
			<AlertDialog.Title class="mb-2">
				{success ? m.auth_signed_in() : m.sign_in_again_title()}
			</AlertDialog.Title>
			<AlertDialog.Description class="text-body mb-6" aria-live="polite">
				{#if success}
					{m.sign_in_again_picking_up()}
				{:else if failure}
					{failure.closed ? m.auth_passkey_prompt_closed() : failure.message}
				{:else}
					{m.sign_in_again_description()}
				{/if}
			</AlertDialog.Description>
		</div>
		{#if $userStore}
			<div class="mb-6">
				<IdentityRow user={$userStore} />
			</div>
		{/if}
		<div class="flex flex-col items-center gap-1">
			<Button class={authPrimaryClass} isLoading={isLoading || success} onclick={authenticate}>
				{failure ? m.try_again() : m.auth_sign_in_with_passkey()}
			</Button>
			<Button
				variant="link"
				class={authQuietClass}
				href={alternativeSignInHref()}
				onclick={() => signInAgain.finish(false)}
			>
				{m.auth_other_ways_to_sign_in()}
			</Button>
		</div>
	</AlertDialog.Content>
</AlertDialog.Root>
