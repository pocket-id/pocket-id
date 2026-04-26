<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import ScopeList from '$lib/components/scope-list.svelte';
	import * as Avatar from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { cachedProfilePicture } from '$lib/utils/cached-image-util';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication, type AuthenticationResponseJSON } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import type { PageProps } from './$types';
	import ClientProviderImages from './components/client-provider-images.svelte';

	const webauthnService = new WebAuthnService();
	const oidService = new OidcService();

	let { data }: PageProps = $props();
	let {
		client,
		scope,
		callbackURL,
		nonce,
		codeChallenge,
		codeChallengeMethod,
		authorizeState,
		prompt,
		responseMode
	} = data;

	let isLoading = $state(false);
	let success = $state(false);
	let errorMessage: string | null = $state(null);
	let authorizationRequired = $state(false);
	let authorizationConfirmed = $state(false);
	let accountSelectionRequired = $state(false);
	let userSignedInAt: Date | undefined;

	const fullName = $derived.by(() => {
		if (!$userStore) {
			return '';
		}

		if ($userStore.displayName) {
			return $userStore.displayName;
		}

		return [$userStore.firstName, $userStore.lastName].filter(Boolean).join(' ').trim();
	});
	const primaryName = $derived(fullName || $userStore?.email || '');

	// Parse prompt parameter once (space-delimited per OIDC spec)
	const promptValues = prompt ? prompt.split(' ') : [];
	const hasPromptNone = promptValues.includes('none');
	const hasPromptConsent = promptValues.includes('consent');
	const hasPromptLogin = promptValues.includes('login');
	const hasPromptSelectAccount = promptValues.includes('select_account');

	onMount(() => {
		// Conflicting prompt values - none can't be combined with any interactive prompt
		if (hasPromptNone && (hasPromptConsent || hasPromptLogin || hasPromptSelectAccount)) {
			redirectWithError('interaction_required');
			return;
		}

		// If prompt=none and user is not signed in, redirect immediately with login_required
		if (hasPromptNone && !$userStore) {
			redirectWithError('login_required');
			return;
		}

		// prompt=select_account: if the user is already signed in, pause so they can
		// confirm the current account before proceeding. If they're not signed in,
		// the normal login flow below is selection enough.
		if (hasPromptSelectAccount && $userStore) {
			accountSelectionRequired = true;
			return;
		}

		if ($userStore) {
			authorize();
		}
	});

	async function useDifferentAccount() {
		try {
			await webauthnService.logout();
		} finally {
			await invalidateAll();
		}
	}

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
			if (client?.requiresReauthentication || hasPromptLogin) {
				let authResponse;
				const signedInRecently =
					userSignedInAt && userSignedInAt.getTime() > Date.now() - 60 * 1000;
				if (!signedInRecently) {
					const loginOptions = await webauthnService.getLoginOptions();
					authResponse = await startAuthentication({ optionsJSON: loginOptions });
				}
				reauthToken = await webauthnService.reauthenticate(authResponse);
			}

			const result = await oidService.authorize(
				client!.id,
				scope,
				callbackURL,
				nonce,
				codeChallenge,
				codeChallengeMethod,
				reauthToken,
				responseMode,
				prompt
			);

			// Check if backend returned a redirect error
			if (result.requiresRedirect && result.error) {
				if (hasPromptNone) {
					redirectWithError(result.error);
				} else {
					errorMessage = result.error;
					isLoading = false;
				}
				return;
			}

			onSuccess(result.code!, result.callbackURL!, result.issuer!);
		} catch (e) {
			errorMessage = getWebauthnErrorMessage(e);
			isLoading = false;
		}
	}

	function redirectWithError(error: string) {
		const redirectURL = new URL(callbackURL);
		if (redirectURL.protocol == 'javascript:' || redirectURL.protocol == 'data:') {
			throw new Error('Invalid redirect URL protocol');
		}

		redirectURL.searchParams.append('error', error);
		if (authorizeState) {
			redirectURL.searchParams.append('state', authorizeState);
		}
		window.location.href = redirectURL.toString();
	}

	function onSuccess(code: string, callbackURL: string, issuer: string) {
		const redirectURL = new URL(callbackURL);
		if (redirectURL.protocol == 'javascript:' || redirectURL.protocol == 'data:') {
			throw new Error('Invalid redirect URL protocol');
		}

		redirectURL.searchParams.append('code', code);
		redirectURL.searchParams.append('state', authorizeState);
		redirectURL.searchParams.append('iss', issuer);

		success = true;
		setTimeout(() => {
			if (responseMode === 'form_post') {
				// Create a hidden form and submit it via POST
				const form = document.createElement('form');
				form.method = 'POST';
				form.action = callbackURL;

				// Add code parameter
				const codeInput = document.createElement('input');
				codeInput.type = 'hidden';
				codeInput.name = 'code';
				codeInput.value = code;
				form.appendChild(codeInput);

				// Add state parameter
				if (authorizeState) {
					const stateInput = document.createElement('input');
					stateInput.type = 'hidden';
					stateInput.name = 'state';
					stateInput.value = authorizeState;
					form.appendChild(stateInput);
				}

				// Add issuer parameter
				const issInput = document.createElement('input');
				issInput.type = 'hidden';
				issInput.name = 'iss';
				issInput.value = issuer;
				form.appendChild(issInput);

				document.body.appendChild(form);
				form.submit();
			} else {
				// Default query parameter redirect (response_mode=query or not specified)
				const redirectURL = new URL(callbackURL);
				redirectURL.searchParams.append('code', code);
				redirectURL.searchParams.append('state', authorizeState);
				redirectURL.searchParams.append('iss', issuer);

				window.location.href = redirectURL.toString();
			}
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
		{#if accountSelectionRequired && $userStore && !errorMessage}
			<div transition:slide={{ duration: 300 }} class="flex flex-col items-center">
				<p class="text-muted-foreground mt-2 mb-8">
					<FormattedMessage m={m.account_selection_signin_confirmation({ name: client.name })} />
				</p>
				<Card.Root class="mb-2 py-4 w-sm" data-testid="account-selection">
					<Card.Content class="flex items-center gap-4">
						<Avatar.Root class="size-11 shrink-0">
							<Avatar.Image src={cachedProfilePicture.getUrl($userStore.id)} />
						</Avatar.Root>
						<div class="flex min-w-0 flex-col text-start">
							<p class="truncate text-base leading-tight font-medium">
								{primaryName}
							</p>
							{#if fullName && $userStore.email}
								<p class="text-muted-foreground mt-1 truncate text-sm leading-tight">
									{$userStore.email}
								</p>
							{/if}
						</div>
					</Card.Content>
				</Card.Root>
				<div class="mb-10 flex justify-center">
					<button
						type="button"
						class="text-muted-foreground text-xs transition-colors hover:underline"
						onclick={useDifferentAccount}
					>
						{m.use_a_different_account()}
					</button>
				</div>
			</div>
		{:else if !authorizationRequired && !errorMessage}
			<p class="text-muted-foreground mt-2 mb-10">
				<FormattedMessage
					m={m.do_you_want_to_sign_in_to_client_with_your_app_name_account({
						client: client.name,
						appName: $appConfigStore.appName
					})}
				/>
			</p>
		{:else if authorizationRequired}
			<div class="w-full max-w-md" transition:slide={{ duration: 300 }}>
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
		<div class="flex w-full max-w-md flex-row-reverse gap-2">
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
