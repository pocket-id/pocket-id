<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from './components/login-logo-error-success-indicator.svelte';
	import { LucideKey, LucideMail, LucideRectangleEllipsis } from '@lucide/svelte';
	import { page } from '$app/state';
	import type { Component } from 'svelte';
	import LoginMethodCard from './components/login-method-card.svelte';

	let { data } = $props();

	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let error: string | undefined = $state(undefined);

	// Reactive computed methods based on configuration
	const methods = $derived.by(() => {
		const baseMethods: {
			icon: Component;
			title: string;
			description: string;
			href?: string;
			onclick?: () => void | Promise<void>;
		}[] = [
			{
				icon: LucideKey,
				title: m.passkey(),
				description: m.authenticate_with_passkey_to_access_account(),
				onclick: authenticate
			},
			{
				icon: LucideRectangleEllipsis,
				title: m.login_code(),
				description: m.enter_a_login_code_to_sign_in(),
				href: '/login/alternative/code'
			}
		];

		if ($appConfigStore.emailOneTimeAccessAsUnauthenticatedEnabled) {
			baseMethods.push({
				icon: LucideMail,
				title: m.email_login(),
				description: m.request_a_login_code_via_email(),
				href: '/login/alternative/email'
			});
		}

		return baseMethods;
	});

	const isAllMethodsEnabled = $derived($appConfigStore.allMethodsEnabled);
	const isSignupOpen = $derived($appConfigStore.allowUserSignups === 'open');
	const showAlternativeButton = $derived(!isAllMethodsEnabled);

	async function authenticate() {
		error = undefined;
		isLoading = true;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication({ optionsJSON: loginOptions });
			const user = await webauthnService.finishLogin(authResponse);

			await userStore.setUser(user);
			goto(data.redirect || '/settings');
		} catch (e) {
			error = getWebauthnErrorMessage(e);
		}
		isLoading = false;
	}
</script>

<svelte:head>
	<title>{m.sign_in()}</title>
</svelte:head>

<SignInWrapper
	animate={!$appConfigStore.disableAnimations}
	showAlternativeSignInMethodButton={showAlternativeButton}
>
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

	{#if isAllMethodsEnabled}
		<div class="mt-5 flex flex-col gap-3">
			{#each methods as method}
				<LoginMethodCard
					icon={method.icon}
					title={method.title}
					description={method.description}
					href={method.href}
					onclick={method.onclick}
					searchParams={page.url.search}
				/>
			{/each}
		</div>
	{/if}

	<div class="mt-10 flex justify-center gap-3">
		{#if isSignupOpen}
			<Button variant="secondary" href="/signup">
				{m.signup()}
			</Button>
		{/if}

		{#if showAlternativeButton}
			<Button {isLoading} onclick={authenticate} autofocus={true}>
				{error ? m.try_again() : m.authenticate()}
			</Button>
		{/if}
	</div>
</SignInWrapper>
