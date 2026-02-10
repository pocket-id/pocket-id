<script lang="ts">
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import ScopeList from '$lib/components/scope-list.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication, type AuthenticationResponseJSON } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import type { PageProps } from './$types';
	import ClientProviderImages from './components/client-provider-images.svelte';

	const webauthnService = new WebAuthnService();
	const oidService = new OidcService();

	let { data }: PageProps = $props();
	let { client, scope, callbackURL, nonce, codeChallenge, codeChallengeMethod, authorizeState, prompt } =
		data;

	let isLoading = $state(false);
	let success = $state(false);
	let errorMessage: string | null = $state(null);
	let authorizationRequired = $state(false);
	let authorizationConfirmed = $state(false);
	let userSignedInAt: Date | undefined;

	// Parse prompt parameter once (space-delimited per OIDC spec)
	const promptValues = prompt ? prompt.split(' ') : [];
	const hasPromptNone = promptValues.includes('none');
	const hasPromptConsent = promptValues.includes('consent');

	onMount(() => {
		// If prompt=none and user is not signed in, redirect immediately with login_required
		if (hasPromptNone && !$userStore) {
			redirectWithError('login_required');
			return;
		}

		if ($userStore) {
			authorize();
		}
	});

	async function authorize() {
		isLoading = true;

		let authResponse: AuthenticationResponseJSON | undefined;

		try {
			if (!$userStore?.id) {
				const loginOptions = await webauthnService.getLoginOptions();
				authResponse = await startAuthentication({ optionsJSON: loginOptions });
				const user = await webauthnService.finishLogin(authResponse);
				userStore.setUser(user);
				userSignedInAt = new Date();
			}

			if (!authorizationConfirmed) {
				authorizationRequired = await oidService.isAuthorizationRequired(client!.id, scope);
				
				// If prompt=consent, always show consent UI
				if (hasPromptConsent) {
					authorizationRequired = true;
				}

				// If prompt=none and consent required, redirect with error
				if (hasPromptNone && authorizationRequired) {
					redirectWithError('consent_required');
					return;
				}

				if (authorizationRequired) {
					isLoading = false;
					authorizationConfirmed = true;
					return;
				}
			}

			let reauthToken: string | undefined;
			if (client?.requiresReauthentication) {
				let authResponse;
				const signedInRecently =
					userSignedInAt && userSignedInAt.getTime() > Date.now() - 60 * 1000;
				if (!signedInRecently) {
					const loginOptions = await webauthnService.getLoginOptions();
					authResponse = await startAuthentication({ optionsJSON: loginOptions });
				}
				reauthToken = await webauthnService.reauthenticate(authResponse);
			}

			const result = await oidService
				.authorize(
					client!.id,
					scope,
					callbackURL,
					nonce,
					codeChallenge,
					codeChallengeMethod,
					reauthToken,
					prompt
				);

			// Check if backend returned a redirect error
			if (result.requiresRedirect && result.error) {
				redirectWithError(result.error);
				return;
			}

			onSuccess(result.code!, result.callbackURL, result.issuer!);
		} catch (e) {
			errorMessage = getWebauthnErrorMessage(e);
			isLoading = false;
		}
	}

	function redirectWithError(error: string) {
		const redirectURL = new URL(callbackURL);
		redirectURL.searchParams.append('error', error);
		if (authorizeState) {
			redirectURL.searchParams.append('state', authorizeState);
		}
		window.location.href = redirectURL.toString();
	}

	function onSuccess(code: string, callbackURL: string, issuer: string) {
		success = true;
		setTimeout(() => {
			const redirectURL = new URL(callbackURL);
			redirectURL.searchParams.append('code', code);
			redirectURL.searchParams.append('state', authorizeState);
			redirectURL.searchParams.append('iss', issuer);

			window.location.href = redirectURL.toString();
		}, 1000);
	}
</script>

<svelte:head>
	<title>{m.sign_in_to({ name: client.name })}</title>
</svelte:head>

{#if client == null}
	<p>{m.client_not_found()}</p>
{:else}
	<SignInWrapper showAlternativeSignInMethodButton={$userStore == null}>
		<ClientProviderImages {client} {success} error={!!errorMessage} />
		<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">
			{m.sign_in_to({ name: client.name })}
		</h1>
		{#if errorMessage}
			<p class="text-muted-foreground mt-2 mb-10">
				{errorMessage}.
			</p>
		{/if}
		{#if !authorizationRequired && !errorMessage}
			<p class="text-muted-foreground mt-2 mb-10">
				<FormattedMessage
					m={m.do_you_want_to_sign_in_to_client_with_your_app_name_account({
						client: client.name,
						appName: $appConfigStore.appName
					})}
				/>
			</p>
		{:else if authorizationRequired}
			<div class="w-full max-w-[450px]" transition:slide={{ duration: 300 }}>
				<Card.Root class="mt-6 mb-10">
					<Card.Header>
						<p class="text-muted-foreground text-start">
							<FormattedMessage
								m={m.client_wants_to_access_the_following_information({ client: client.name })}
							/>
						</p>
					</Card.Header>
					<Card.Content>
						<ScopeList {scope} />
					</Card.Content>
				</Card.Root>
			</div>
		{/if}
		<!-- Flex flow is reversed so the sign in button, which has auto-focus, is the first one in the DOM, for a11y -->
		<div class="flex w-full max-w-[450px] flex-row-reverse gap-2">
			{#if !errorMessage}
				<Button class="flex-1" {isLoading} onclick={authorize} autofocus={true}>
					{m.sign_in()}
				</Button>
			{:else}
				<Button class="flex-1" onclick={() => (errorMessage = null)}>
					{m.try_again()}
				</Button>
			{/if}
			<Button href={document.referrer || '/'} class="flex-1" variant="secondary">
				{m.cancel()}
			</Button>
		</div>
	</SignInWrapper>
{/if}
