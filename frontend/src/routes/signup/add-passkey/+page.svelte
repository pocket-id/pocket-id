<script lang="ts">
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		authPrimaryClass,
		authQuietClass,
		isPasskeyPromptClosed,
		successHold
	} from '#lib/components/auth/auth-util.ts';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import { goto } from '$app/navigation';
	import { startRegistration } from '@simplewebauthn/browser';
	import { LucideSkipForward } from '@lucide/svelte';

	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let success = $state(false);
	let failure: { message: string; closed: boolean } | undefined = $state();

	async function createPasskeyAndContinue() {
		isLoading = true;
		failure = undefined;
		try {
			const options = await webauthnService.getRegistrationOptions();
			const response = await startRegistration({ optionsJSON: options });
			await webauthnService.finishRegistration(response);
			dismiss();
			success = true;
			await successHold();
			await goto('/settings/account');
		} catch (e) {
			failure = { message: getWebauthnErrorMessage(e), closed: isPasskeyPromptClosed(e) };
		}
		isLoading = false;
	}

	// Skipping leaves the person without a way back in once the session ends, so it asks once
	function skipForNow() {
		openConfirmDialog({
			title: m.auth_skip_passkey_title(),
			message: m.auth_skip_passkey_description(),
			confirm: {
				label: m.skip_for_now(),
				destructive: true,
				icon: LucideSkipForward,
				action: () => {
					dismiss();
					goto('/settings/account');
				}
			}
		});
	}

	function dismiss() {
		const inOneDay = Date.now() + 24 * 60 * 60 * 1000;
		localStorage.setItem('skip-passkey-setup-until', inOneDay.toString());
	}
</script>

<svelte:head>
	<title>{m.auth_add_a_passkey()}</title>
</svelte:head>

<SignInWrapper>
	<AuthTile {success} />
	<AuthHeading
		title={success ? m.auth_passkey_added() : m.auth_add_a_passkey()}
		tone={failure ? (failure.closed ? 'info' : 'danger') : undefined}
	>
		{#if success}
			{m.auth_taking_you_to_your_account()}
		{:else if failure}
			{failure.closed ? m.auth_passkey_prompt_closed() : failure.message}
		{:else}
			{m.auth_add_passkey_description({ appName: $appConfigStore.appName })}
		{/if}
	</AuthHeading>
	<div
		class="flex flex-col items-center gap-3 transition-opacity duration-(--dur-fast)"
		class:opacity-0={success}
		inert={success}
	>
		<Button class={authPrimaryClass} onclick={createPasskeyAndContinue} {isLoading}>
			{failure ? m.try_again() : m.add_passkey()}
		</Button>
		<Button variant="link" class={authQuietClass} onclick={skipForNow} disabled={isLoading}>
			{m.skip_for_now()}
		</Button>
	</div>
</SignInWrapper>
