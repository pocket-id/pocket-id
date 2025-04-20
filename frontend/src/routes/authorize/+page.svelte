<script lang="ts">
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { LucideMail, LucideUser, LucideUsers } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import type { PageData } from './$types';
	import ClientProviderImages from './components/client-provider-images.svelte';
	import ScopeItem from './components/scope-item.svelte';

	const webauthnService = new WebAuthnService();
	const oidService = new OidcService();

	let isLoading = false;
	let success = false;
	let errorMessage: string | null = null;
	let authorizationRequired = false;
	let authorizationConfirmed = false;

	export let data: PageData;
	let { scope, nonce, client, state, callbackURL, codeChallenge, codeChallengeMethod } = data;

	onMount(() => {
		if ($userStore) {
			authorize();
		}
	});

	async function authorize() {
		isLoading = true;
		try {
			// Get access token if not signed in
			if (!$userStore?.id) {
				const loginOptions = await webauthnService.getLoginOptions();
				const authResponse = await startAuthentication(loginOptions);
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

			await oidService
				.authorize(client!.id, scope, callbackURL, nonce, codeChallenge, codeChallengeMethod)
				.then(async ({ code, callbackURL }) => {
					onSuccess(code, callbackURL);
				});
		} catch (e) {
			errorMessage = getWebauthnErrorMessage(e);
			isLoading = false;
		}
	}

	function onSuccess(code: string, callbackURL: string) {
		success = true;
		setTimeout(() => {
			const redirectURL = new URL(callbackURL);
			redirectURL.searchParams.append('code', code);
			redirectURL.searchParams.append('state', state);

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
	<SignInWrapper animate={!$appConfigStore.disableAnimations} showAlternativeSignInMethodButton>
		<ClientProviderImages {client} {success} error={!!errorMessage} />
		<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">
			{m.sign_in_to({ name: client.name })}
		</h1>
		{#if errorMessage}
			<p class="text-muted-foreground mb-10 mt-2">
				{errorMessage}.
			</p>
		{/if}
		{#if !authorizationRequired && !errorMessage}
			<p class="text-muted-foreground mb-10 mt-2">
				{@html m.do_you_want_to_sign_in_to_client_with_your_app_name_account({
					client: client.name,
					appName: $appConfigStore.appName
				})}
			</p>
		{:else if authorizationRequired}
			<div transition:slide={{ duration: 300 }}>
				<Card.Root class="mb-10 mt-6">
					<Card.Header class="pb-5">
						<p class="text-muted-foreground text-start">
							{@html m.client_wants_to_access_the_following_information({ client: client.name })}
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
		<div class="flex w-full justify-stretch gap-2">
			<Button onclick={() => history.back()} class="w-full" variant="secondary">{m.cancel()}</Button
			>
			{#if !errorMessage}
				<Button class="w-full" {isLoading} on:click={authorize}>{m.sign_in()}</Button>
			{:else}
				<Button class="w-full" on:click={() => (errorMessage = null)}>{m.try_again()}</Button>
			{/if}
		</div>
	</SignInWrapper>
{/if}
