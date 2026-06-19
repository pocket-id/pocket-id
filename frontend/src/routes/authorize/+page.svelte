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
	import { getAxiosErrorMessage, getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication, type AuthenticationResponseJSON } from '@simplewebauthn/browser';
	import { goto } from '$app/navigation';
	import { needsAlternativeLogin, navigateToAlternativeLogin } from '$lib/utils/device-detect-util';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import type { PageProps } from './$types';
	import ClientProviderImages from './components/client-provider-images.svelte';

	const webauthnService = new WebAuthnService();
	const oidService = new OidcService();

	let { data }: PageProps = $props();
	let {
		client,
		callbackURL,
		nonce,
		codeChallenge,
		codeChallengeMethod,
		authorizeState,
		prompt,
		responseMode,
		requestURI
	} = data;
	let scope = $state(data.scope);
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
		void handleInitialAuthorization();
	});

	async function handleInitialAuthorization() {
		// Conflicting prompt values - none can't be combined with any interactive prompt
		if (hasPromptNone && (hasPromptConsent || hasPromptLogin || hasPromptSelectAccount)) {
			await redirectWithError('interaction_required');
			return;
		}

		// If prompt=none and user is not signed in, redirect immediately with login_required
		if (hasPromptNone && !$userStore) {
			await redirectWithError('login_required');
			return;
		}

		// Redirect limited devices / no WebAuthn to alternative login.
		// `loop=1` query param breaks the cycle if the alternative-login page sends us back here
		// while the user is still not signed in (e.g. user cancelled).
		const alreadyRedirected = new URLSearchParams(window.location.search).get('loop') === '1';
		const needsAltLogin = !$userStore && $appConfigStore.qrLoginEnabled && needsAlternativeLogin();
		if (needsAltLogin && !alreadyRedirected) {
			const currentUrl = window.location.pathname + window.location.search;
			const separator = currentUrl.includes('?') ? '&' : '?';
			const returnUrl = currentUrl + separator + 'loop=1';
			navigateToAlternativeLogin(`?redirect=${encodeURIComponent(returnUrl)}`, goto);
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
			await authorize();
		}
	}

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
				const authRequired = await oidService.isAuthorizationRequired(
					client!.id,
					scope,
					requestURI
				);
				authorizationRequired = authRequired.authorizationRequired;

				if (requestURI) {
					scope = authRequired.scope;
				}

				// If prompt=consent, always show consent UI
				if (hasPromptConsent) {
					authorizationRequired = true;
				}

				// If prompt=none and consent required, redirect with error
				if (hasPromptNone && authorizationRequired) {
					await redirectWithError('consent_required');
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
				let reauthResponse: AuthenticationResponseJSON | undefined;
				// The QR-login flow sends the user back here with `reauth=1` after a fresh sign-in on
				// another device. Only trust the marker on devices that actually took that path
				// (limited devices), so it can't be appended manually on a capable device to bypass
				// reauthentication. Consume it immediately by stripping it from the URL, so a reload
				// can never re-trust a stale marker.
				const onLimitedDevice = $appConfigStore.qrLoginEnabled && needsAlternativeLogin();
				const params = new URLSearchParams(window.location.search);
				const justReauthedViaAlt = onLimitedDevice && params.get('reauth') === '1';
				if (justReauthedViaAlt) {
					params.delete('reauth');
					const query = params.toString();
					history.replaceState(
						history.state,
						'',
						window.location.pathname + (query ? `?${query}` : '')
					);
				}
				const signedInRecently =
					(userSignedInAt && userSignedInAt.getTime() > Date.now() - 60 * 1000) ||
					justReauthedViaAlt;
				if (!signedInRecently) {
					// On limited devices (TVs etc.) a WebAuthn prompt would fail; route to QR login instead.
					if (onLimitedDevice) {
						const currentUrl = window.location.pathname + window.location.search;
						const separator = currentUrl.includes('?') ? '&' : '?';
						const returnUrl = currentUrl + separator + 'reauth=1';
						navigateToAlternativeLogin(`?redirect=${encodeURIComponent(returnUrl)}`, goto);
						return;
					}
					const loginOptions = await webauthnService.getLoginOptions();
					reauthResponse = await startAuthentication({ optionsJSON: loginOptions });
				}
				reauthToken = await webauthnService.reauthenticate(reauthResponse);
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
				prompt,
				requestURI
			);

			// Check if backend returned a redirect error
			if (result.requiresRedirect && result.error) {
				if (hasPromptNone) {
					await redirectWithError(result.error, result.callbackURL);
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

	async function redirectWithError(error: string, validatedCallbackURL?: string) {
		isLoading = true;

		try {
			const safeCallbackURL =
				validatedCallbackURL ||
				(await oidService.resolveAuthorizeCallbackURL(client!.id, callbackURL, requestURI))
					.callbackURL;
			const redirectURL = new URL(safeCallbackURL);
			if (redirectURL.protocol == 'javascript:' || redirectURL.protocol == 'data:') {
				throw new Error('Invalid redirect URL protocol');
			}

			window.location.href = createRedirectURL(safeCallbackURL, {
				error,
				state: authorizeState
			});
		} catch (e) {
			errorMessage = getAxiosErrorMessage(e);
			isLoading = false;
		}
	}

	function onSuccess(code: string, callbackURL: string, issuer: string) {
		const redirectURL = new URL(callbackURL);
		if (redirectURL.protocol == 'javascript:' || redirectURL.protocol == 'data:') {
			throw new Error('Invalid redirect URL protocol');
		}

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
				window.location.href = createRedirectURL(callbackURL, {
					code,
					state: authorizeState,
					iss: issuer
				});
			}
		}, 1000);
	}

	function createRedirectURL(url: string, params: Record<string, string>) {
		const redirectURL = new URL(url);
		const responseParams =
			responseMode === 'fragment'
				? new URLSearchParams(redirectURL.hash.slice(1))
				: redirectURL.searchParams;

		for (const [key, value] of Object.entries(params)) {
			if (value) {
				responseParams.set(key, value);
			}
		}

		if (responseMode === 'fragment') {
			redirectURL.hash = responseParams.toString();
		}

		return redirectURL.toString();
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
		<h1 class="font-gloock mt-5 text-3xl font-bold sm:text-4xl">
			{m.sign_in_to({ name: client.name })}
		</h1>
		{#if errorMessage}
			<p class="text-muted-foreground mt-2 mb-10">
				{errorMessage}.
			</p>
		{/if}
		{#if authorizationRequired}
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
		{:else if accountSelectionRequired && $userStore && !errorMessage}
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
			<Button href={'/'} class="flex-1" variant="secondary">
				{m.cancel()}
			</Button>
		</div>
	</SignInWrapper>
{/if}
