<script lang="ts">
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import ScopeItem from '$lib/components/scope-item.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { LucideMail, LucideUser, LucideUsers } from '@lucide/svelte';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import type { PageProps } from './$types';
	import ClientProviderImages from './components/client-provider-images.svelte';

	const webauthnService = new WebAuthnService();
	const oidService = new OidcService();

	let { data }: PageProps = $props();
	let { client, scope, callbackURL, nonce, codeChallenge, codeChallengeMethod, authorizeState } =
		data;

	let isLoading = $state(false);
	let success = $state(false);
	let errorMessage: string | null = $state(null);
	let authorizationRequired = $state(false);
	let authorizationConfirmed = $state(false);

	onMount(() => {
		if ($userStore) {
			authorize();
		}
	});

	async function authorize() {
		isLoading = true;
		try {
			if (!$userStore?.id || client?.requiresReauthentication) {
				const loginOptions = await webauthnService.getLoginOptions();
				const authResponse = await startAuthentication({ optionsJSON: loginOptions });
				const user = await webauthnService.finishLogin(authResponse);
				userStore.setUser(user);
			}

			if (!authorizationConfirmed) {
				authorizationRequired = await oidService.isAuthorizationRequired(client!.id, scope);
				if (authorizationRequired) {
					isLoading = false;
					authorizationConfirmed = true;
					return;
				}
			}

			const reauthToken = client?.requiresReauthentication 
				? await oidService.reauthenticate()
				: undefined;

			const response = await oidService.authorize(
				client!.id,
				scope,
				callbackURL,
				nonce,
				codeChallenge,
				codeChallengeMethod,
				reauthToken
			);

			onSuccess(response.code, response.callbackURL, response.issuer);
		} catch (e) {
			errorMessage = getWebauthnErrorMessage(e);
			isLoading = false;
		}
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
					<Card.Content data-testid="scopes">
						<div class="flex flex-col gap-3">
							{#if scope!.includes('email')}
								<ScopeItem
									icon={LucideMail}
									name={m.email()}
									description={m.view_your_email_address()}
								/>
							{/if}
							{#if scope!.includes('profile')}
								<ScopeItem
									icon={LucideUser}
									name={m.profile()}
									description={m.view_your_profile_information()}
								/>
							{/if}
							{#if scope!.includes('groups')}
								<ScopeItem
									icon={LucideUsers}
									name={m.groups()}
									description={m.view_the_groups_you_are_a_member_of()}
								/>
							{/if}
						</div>
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
			<Button onclick={() => history.back()} class="flex-1" variant="secondary">
				{m.cancel()}
			</Button>
		</div>
	</SignInWrapper>
{/if}
