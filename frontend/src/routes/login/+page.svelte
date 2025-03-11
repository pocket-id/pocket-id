<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from './components/login-logo-error-success-indicator.svelte';
	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let error: string | undefined = $state(undefined);

	async function authenticate() {
		error = undefined;
		isLoading = true;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication(loginOptions);
			const user = await webauthnService.finishLogin(authResponse);

			userStore.setUser(user);
			goto('/settings');
		} catch (e) {
			error = getWebauthnErrorMessage(e);
		}
		isLoading = false;
	}
</script>

<svelte:head>
	<title>Sign In</title>
</svelte:head>

<SignInWrapper showAlternativeSignInMethodButton={$appConfigStore.showAlternativeSignInButton}>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">
		Sign in to {$appConfigStore.appName}
	</h1>
	{#if error}
		<p class="text-muted-foreground mt-2" in:fade>
			{error}. Please try to sign in again.
		</p>
	{:else}
		<p class="text-muted-foreground mt-2" in:fade>
			Authenticate yourself with your passkey to access the admin panel.
		</p>
	{/if}
	<Button class="mt-10" {isLoading} on:click={authenticate}
		>{error ? 'Try again' : 'Authenticate'}</Button
	>
</SignInWrapper>
