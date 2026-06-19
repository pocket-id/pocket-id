<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';

	import { needsAlternativeLogin, navigateToAlternativeLogin } from '$lib/utils/device-detect-util';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { isSafeRedirect } from '$lib/utils/redirection-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { cn } from 'tailwind-variants';
	import LoginLogoErrorSuccessIndicator from './components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let error: string | undefined = $state(undefined);

	onMount(() => {
		const params = new URLSearchParams(window.location.search);
		const method = params.get('method');

		if (method === 'code' || method === 'email') {
			params.delete('method');
			const remaining = params.toString();
			goto(`/login/alternative/${method}` + (remaining ? `?${remaining}` : ''));
			return;
		}

		if (method === 'qr' && $appConfigStore.qrLoginEnabled) {
			params.delete('method');
			const remaining = params.toString();
			navigateToAlternativeLogin(remaining ? `?${remaining}` : '', goto);
			return;
		}

		if ($appConfigStore.qrLoginEnabled && method !== 'passkey' && needsAlternativeLogin()) {
			const remaining = params.toString();
			navigateToAlternativeLogin(remaining ? `?${remaining}` : '', goto);
			return;
		}
	});

	async function authenticate() {
		error = undefined;
		isLoading = true;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication({ optionsJSON: loginOptions });
			const user = await webauthnService.finishLogin(authResponse);

			await userStore.setUser(user);
			const target = data.redirect;
			if (isSafeRedirect(target)) {
				goto(target);
			} else {
				goto('/settings');
			}
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
	<h1 class="font-gloock mt-5 text-3xl font-bold sm:text-4xl">
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
			autofocus={true}
		>
			{error ? m.try_again() : m.authenticate()}
		</Button>
	</div>
</SignInWrapper>
