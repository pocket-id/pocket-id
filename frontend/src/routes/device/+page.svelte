<script lang="ts">
	import { goto } from '$app/navigation';
	import CodeInput from '$lib/components/code-input.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import ScopeList from '$lib/components/scope-list.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import type { OidcDeviceCodeInfo } from '$lib/types/oidc.type';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import ClientProviderImages from '../authorize/components/client-provider-images.svelte';
	import LoginLogoErrorSuccessIndicator from '../login/components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const oidcService = new OIDCService();
	const webauthnService = new WebAuthnService();

	onMount(() => {
		if (!$appConfigStore.qrLoginEnabled) {
			goto('/');
		}
	});

	type PageState = 'enter-code' | 'confirm-code' | 'authorize-scopes' | 'success' | 'error';

	let pageState: PageState = $state(data.code ? 'confirm-code' : 'enter-code');
	let userCode = $state(data.code || '');
	let isLoading = $state(false);
	let deviceInfo: OidcDeviceCodeInfo | undefined = $state();
	let errorMessage: string | null = $state(null);

	let codeComplete = $derived(userCode.replace(/[^a-zA-Z0-9]/g, '').length >= 8);

	async function submitCode() {
		isLoading = true;
		errorMessage = null;
		try {
			if (!$userStore) {
				const loginOptions = await webauthnService.getLoginOptions();
				const authResponse = await startAuthentication({ optionsJSON: loginOptions });
				const user = await webauthnService.finishLogin(authResponse);
				await userStore.setUser(user);
			}

			deviceInfo = await oidcService.getDeviceCodeInfo(userCode);

			if (deviceInfo.authorizationRequired) {
				pageState = 'authorize-scopes';
				isLoading = false;
				return;
			}

			await oidcService.verifyDeviceCode(userCode);
			pageState = 'success';
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
			pageState = 'error';
		} finally {
			isLoading = false;
		}
	}

	async function authorizeScopes() {
		isLoading = true;
		try {
			await oidcService.verifyDeviceCode(userCode);
			pageState = 'success';
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
			pageState = 'error';
		} finally {
			isLoading = false;
		}
	}

	function reset() {
		errorMessage = null;
		pageState = 'enter-code';
		userCode = '';
		deviceInfo = undefined;
	}
</script>

<svelte:head>
	<title>{m.authorize_device()}</title>
</svelte:head>

<SignInWrapper showAlternativeSignInMethodButton={$userStore == null}>
	<div class="flex justify-center">
		{#if deviceInfo?.client}
			<ClientProviderImages
				client={deviceInfo.client}
				success={pageState === 'success'}
				error={pageState === 'error'}
			/>
		{:else}
			<LoginLogoErrorSuccessIndicator
				success={pageState === 'success'}
				error={pageState === 'error'}
			/>
		{/if}
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">{m.authorize_device()}</h1>

	{#if pageState === 'enter-code'}
		<p class="text-muted-foreground mt-2">{m.enter_code_displayed_in_previous_step()}</p>
		<div class="mt-7 flex justify-center">
			<CodeInput bind:value={userCode} autofocus onsubmit={() => codeComplete && submitCode()} />
		</div>
		<div class="mt-10 flex w-full max-w-[450px] gap-2">
			<Button href="/" class="flex-1" variant="secondary">{m.cancel()}</Button>
			<Button class="flex-1" disabled={!codeComplete} onclick={submitCode} {isLoading}
				>{m.authorize()}</Button
			>
		</div>
	{:else if pageState === 'confirm-code'}
		<p class="text-muted-foreground mt-2">{m.confirm_code_matches()}</p>
		<div class="mt-5 flex justify-center">
			<CodeInput value={userCode} disabled />
		</div>
		<div class="mt-10 flex w-full max-w-[450px] gap-2">
			<Button class="flex-1" variant="secondary" onclick={reset}>{m.cancel()}</Button>
			<Button class="flex-1" onclick={submitCode} {isLoading}>{m.authorize()}</Button>
		</div>
	{:else if pageState === 'authorize-scopes'}
		<div class="w-full max-w-[450px]" transition:slide={{ duration: 300 }}>
			<Card.Root class="mt-6">
				<Card.Header class="pb-5">
					<p class="text-muted-foreground text-start">
						<FormattedMessage
							m={m.client_wants_to_access_the_following_information({
								client: deviceInfo!.client.name
							})}
						/>
					</p>
				</Card.Header>
				<Card.Content data-testid="scopes">
					<ScopeList scope={deviceInfo!.scope} />
				</Card.Content>
			</Card.Root>
		</div>
		<div class="mt-10 flex w-full max-w-[450px] gap-2">
			<Button href="/" class="flex-1" variant="secondary">{m.cancel()}</Button>
			<Button class="flex-1" onclick={authorizeScopes} {isLoading}>{m.authorize()}</Button>
		</div>
	{:else if pageState === 'success'}
		<p class="text-muted-foreground mt-2">{m.the_device_has_been_authorized()}</p>
	{:else if pageState === 'error'}
		<p class="text-muted-foreground mt-2">
			{errorMessage}. {m.please_try_again()}
		</p>
		<div class="mt-10 flex w-full max-w-[450px] gap-2">
			<Button href="/" class="flex-1" variant="secondary">{m.cancel()}</Button>
			<Button class="flex-1" onclick={reset}>{m.try_again()}</Button>
		</div>
	{/if}
</SignInWrapper>
