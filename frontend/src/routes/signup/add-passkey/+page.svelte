<script lang="ts">
	import { goto } from '$app/navigation';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { tryCatch } from '$lib/utils/try-catch-util';
	import { startRegistration } from '@simplewebauthn/browser';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from '../../login/components/login-logo-error-success-indicator.svelte';

	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let error: string | undefined = $state();

	async function createPasskeyAndContinue() {
		isLoading = true;
		error = undefined;

		const optsResult = await tryCatch(webauthnService.getRegistrationOptions());
		if (optsResult.error) {
			error = getWebauthnErrorMessage(optsResult.error);
			isLoading = false;
			return;
		}

		const attRespResult = await tryCatch(startRegistration({ optionsJSON: optsResult.data }));
		if (attRespResult.error) {
			error = getWebauthnErrorMessage(attRespResult.error);
			isLoading = false;
			return;
		}

		const finishResult = await tryCatch(webauthnService.finishRegistration(attRespResult.data));
		if (finishResult.error) {
			error = getWebauthnErrorMessage(finishResult.error);
			isLoading = false;
			return;
		}

		goto('/settings/account');
		isLoading = false;
	}

	function skipForNow() {
		openConfirmDialog({
			title: m.skip_passkey_setup(),
			message: m.skip_passkey_setup_description(),
			confirm: {
				label: m.skip_for_now(),
				destructive: true,
				action: () => {
					goto('/settings/account');
				}
			}
		});
	}
</script>

<svelte:head>
	<title>{m.add_passkey()}</title>
</svelte:head>

<SignInWrapper>
	<div class="w-full text-center">
		<div class="flex justify-center">
			<LoginLogoErrorSuccessIndicator error={!!error} />
		</div>
		<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">
			{m.setup_your_passkey()}
		</h1>
		<p class="text-muted-foreground mt-2" in:fade>
			{#if !error}
				{m.create_a_passkey_to_securely_access_your_account()}
			{:else}
				{error}. {m.please_try_again()}
			{/if}
		</p>
		<div class="mt-10 flex w-full justify-between gap-2">
			<Button variant="secondary" onclick={skipForNow} disabled={isLoading} class="flex-1">
				{m.skip_for_now()}
			</Button>
			<Button onclick={createPasskeyAndContinue} {isLoading} class="flex-1">
				{m.add_passkey()}
			</Button>
		</div>
	</div>
</SignInWrapper>
