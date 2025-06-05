<script lang="ts">
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import ScopeList from '$lib/components/scope-list.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import type { OidcDeviceCodeInfo } from '$lib/types/oidc.type';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import ClientProviderImages from '../authorize/components/client-provider-images.svelte';
	import LoginLogoErrorSuccessIndicator from '../login/components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const oidcService = new OIDCService();
	const webauthnService = new WebAuthnService();

	let userCode = $state(data.code || '');
	let isLoading = $state(false);
	let deviceInfo: OidcDeviceCodeInfo | undefined = $state();
	let success = $state(false);
	let errorMessage: string | null = $state(null);
	let authorizationRequired = $state(false);

	onMount(() => {
		if (data.code && $userStore) {
			authorize();
		}
	});

	async function authorize() {
		isLoading = true;
		try {
			// Get access token if not signed in
			if (!$userStore) {
				const loginOptions = await webauthnService.getLoginOptions();
				const authResponse = await startAuthentication({ optionsJSON: loginOptions });
				const user = await webauthnService.finishLogin(authResponse);
				userStore.setUser(user);
			}

			const info = await oidcService.getDeviceCodeInfo(userCode);
			deviceInfo = info;

			if (info.authorizationRequired && !authorizationRequired) {
				authorizationRequired = true;
				isLoading = false;
				return;
			}

			await oidcService.verifyDeviceCode(userCode);

			success = true;
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.authorize_device()}</title>
</svelte:head>

<SignInWrapper
	animate={!$appConfigStore.disableAnimations}
	showAlternativeSignInMethodButton={$userStore == null}
>
	<div class="flex justify-center">
		{#if deviceInfo?.client}
			<ClientProviderImages client={deviceInfo.client} {success} error={!!errorMessage} />
		{:else}
			<LoginLogoErrorSuccessIndicator {success} error={!!errorMessage} />
		{/if}
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">{m.authorize_device()}</h1>
	{#if errorMessage}
		<p class="text-muted-foreground mt-2">
			{errorMessage}. {m.please_try_again()}
		</p>
	{:else if success}
		<p class="text-muted-foreground mt-2">{m.the_device_has_been_authorized()}</p>
	{:else if authorizationRequired}
		<div class="w-full max-w-[450px]" transition:slide={{ duration: 300 }}>
			<Card.Root class="mt-6">
				<Card.Header class="pb-5">
					<p class="text-muted-foreground text-start">
						{@html m.client_wants_to_access_the_following_information({
							client: deviceInfo!.client.name
						})}
					</p>
				</Card.Header>
				<Card.Content data-testid="scopes">
					<ScopeList scope={deviceInfo!.scope} />
				</Card.Content>
			</Card.Root>
		</div>
	{:else}
		<p class="text-muted-foreground mt-2">{m.enter_code_displayed_in_previous_step()}</p>
		<form id="device-code-form" onsubmit={preventDefault(authorize)} class="w-full max-w-[450px]">
			<Input id="user-code" class="mt-7" placeholder={m.code()} bind:value={userCode} type="text" />
		</form>
	{/if}
	{#if !success}
		<div class="mt-10 flex w-full max-w-[450px] gap-2">
			<Button href="/" class="flex-1" variant="secondary">{m.cancel()}</Button>
			{#if !errorMessage}
				<Button form="device-code-form" class="flex-1" onclick={authorize} {isLoading}
					>{m.authorize()}</Button
				>
			{:else}
				<Button class="flex-1" onclick={() => (errorMessage = null)}>{m.try_again()}</Button>
			{/if}
		</div>
	{/if}
</SignInWrapper>
