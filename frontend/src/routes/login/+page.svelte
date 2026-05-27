<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import {
		browserSupportsWebAuthnAutofill,
		startAuthentication,
		type AuthenticationResponseJSON
	} from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { cn } from 'tailwind-variants';
	import LoginLogoErrorSuccessIndicator from './components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let error: string | undefined = $state(undefined);

	onMount(() => {
		void authenticateWithPasskeyAutofill();
	});

	async function finishAuthentication(authResponse: AuthenticationResponseJSON) {
		const user = await webauthnService.finishLogin(authResponse);

		await userStore.setUser(user);
		goto(data.redirect || '/settings');
	}

	async function authenticateWithPasskeyAutofill() {
		let supportsAutofill = false;
		try {
			supportsAutofill = await browserSupportsWebAuthnAutofill();
		} catch {
			return;
		}
		if (!supportsAutofill) return;

		let authResponse: AuthenticationResponseJSON;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			authResponse = await startAuthentication({
				optionsJSON: loginOptions,
				useBrowserAutofill: true
			});
		} catch {
			return;
		}

		try {
			await finishAuthentication(authResponse);
		} catch (e) {
			error = getWebauthnErrorMessage(e);
		}
	}

	async function authenticate() {
		error = undefined;
		isLoading = true;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication({ optionsJSON: loginOptions });
			await finishAuthentication(authResponse);
		} catch (e) {
			error = getWebauthnErrorMessage(e);
		}
		isLoading = false;
	}
</script>

<svelte:head>
	<title>{m.sign_in()}</title>
</svelte:head>

<SignInWrapper showAlternativeSignInMethodButton>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">
		{m.sign_in_to_appname({ appName: $appConfigStore.appName })}
	</h1>
	{#if error}
		<p class="text-muted-foreground mt-2" in:fade>
			{error}. {m.please_try_to_sign_in_again()}
		</p>
	{:else}
		<p class="text-muted-foreground mt-2" in:fade>
			{m.authenticate_with_passkey_to_access_account()}
		</p>
	{/if}
	<div class="mt-7 w-full max-w-[450px]">
		<Input
			type="text"
			autocomplete="username webauthn"
			placeholder={m.passkeys()}
			aria-label={m.passkeys()}
			autofocus={true}
		/>
	</div>
	<div class="mt-10 flex justify-center gap-3 w-full max-w-[450px]">
		{#if $appConfigStore.allowUserSignups === 'open'}
			<Button class="w-[50%]" variant="secondary" href="/signup">
				{m.signup()}
			</Button>
		{/if}
		<Button
			class={cn($appConfigStore.allowUserSignups === 'open' && 'w-[50%]')}
			{isLoading}
			onclick={authenticate}
		>
			{error ? m.try_again() : m.authenticate()}
		</Button>
	</div>
</SignInWrapper>
