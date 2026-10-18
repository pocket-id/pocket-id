<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import ScopeList from '#lib/components/scope-list.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Card from '#lib/components/ui/card/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { InteractionStep } from '#lib/types/oidc.type.ts';
	import { getClientIDHost } from '#lib/utils/client-id-util.ts';
	import { getAxiosErrorCode, getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { slide } from 'svelte/transition';
	import ClientProviderImages from '../authorize/components/client-provider-images.svelte';
	import type { PageProps } from './$types';

	const webauthnService = new WebAuthnService();
	const oidcService = new OidcService();

	let { data }: PageProps = $props();
	let interactionSession = $state(data.interactionSession);
	let isLoading = $state(false);
	let success = $state(false);
	let errorMessage: string | null = $state(null);
	let errorCode: string | undefined = $state();
	let currentStep = $derived(interactionSession.currentStep);
	// Every requested scope starts selected so the user only has to deselect the ones they do not want to grant
	let selectedScopes = $state([...(data.interactionSession.scopes ?? [])]);

	async function useDifferentAccount() {
		await webauthnService.logout();
		await completeInteraction('select_account', true);
		userStore.clearUser();
		await refreshAll();
	}

	async function handlePipeline() {
		isLoading = true;
		errorMessage = null;
		errorCode = undefined;
		try {
			if (!$userStore) {
				await authenticate();
			}

			if (currentStep === 'reauthenticate') {
				await reauthenticate();
			}

			await completeInteraction(currentStep!);
		} catch (e) {
			success = false;
			errorMessage = getWebauthnErrorMessage(e);
			errorCode = getAxiosErrorCode(e);
		} finally {
			isLoading = false;
		}
	}

	async function authenticate() {
		const loginOptions = await webauthnService.getLoginOptions();
		const authResponse = await startAuthentication({ optionsJSON: loginOptions });
		const user = await webauthnService.finishLogin(authResponse);
		await userStore.setUser(user);
		await refreshAll();
	}

	async function reauthenticate() {
		try {
			await webauthnService.reauthenticate();
		} catch {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication({ optionsJSON: loginOptions });
			await webauthnService.reauthenticate(authResponse);
		}
	}

	async function completeInteraction(step: InteractionStep, skipRedirect = false) {
		const result = await oidcService.completeAuthorizeInteractionStep(
			interactionSession.id,
			step,
			step === 'consent' ? selectedScopes : undefined
		);
		if (result.interaction) {
			interactionSession = result.interaction;
			if (interactionSession.currentStep == 'reauthenticate') {
				await handlePipeline();
			}
			return;
		}

		if (result.redirectUrl && !skipRedirect) {
			success = true;
			await new Promise((r) => setTimeout(r, 800));
			window.location.href = result.redirectUrl;
		}
	}
</script>

<svelte:head>
	<title>{m.sign_in_to({ name: interactionSession.client.name })}</title>
</svelte:head>

<SignInWrapper showAlternativeSignInMethodButton={$userStore == null}>
	<ClientProviderImages client={interactionSession.client} {success} error={!!errorMessage} />
	<h1 class="font-display mt-5 text-3xl font-bold sm:text-4xl">
		{m.sign_in_to({ name: interactionSession.client.name })}
	</h1>
	<p class="text-muted-foreground mt-2 mb-10">
		{#if errorMessage}
			{errorMessage}.
		{:else if currentStep == 'select_account' && $userStore}
			<FormattedMessage
				message={m.account_selection_signin_confirmation}
				inputs={{
					name: getClientIDHost(interactionSession.client) ?? interactionSession.client.name
				}}
			/>
		{:else}
			<FormattedMessage
				message={m.do_you_want_to_sign_in_to_client_with_your_app_name_account}
				inputs={{
					client: getClientIDHost(interactionSession.client) ?? interactionSession.client.name,
					appName: $appConfigStore.appName
				}}
			/>
		{/if}
	</p>
	{#if !$userStore || errorMessage}
		<!-- Return nothing -->
	{:else if currentStep === 'select_account'}
		<div
			transition:slide={{ duration: 300 }}
			class="flex flex-col items-center"
			data-testid="account-selection"
		>
			{#if $userStore}
				<div class="mx-5 mb-2 w-full sm:min-w-sm">
					<IdentityRow user={$userStore} />
				</div>
			{/if}
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
	{:else if currentStep === 'consent'}
		<div class="w-full max-w-md" transition:slide={{ duration: 300 }}>
			<Card.Root class="mb-10 gap-3">
				<Card.Header>
					<p class="text-muted-foreground text-start">
						<FormattedMessage
							message={m.client_wants_to_access_the_following_information}
							inputs={{
								client: getClientIDHost(interactionSession.client) ?? interactionSession.client.name
							}}
						/>
					</p>
				</Card.Header>
				<Card.Content>
					<ScopeList
						scopes={interactionSession.scopes || []}
						scopeInfo={interactionSession.scopeInfo ?? []}
						bind:selectedScopes
					/>
				</Card.Content>
			</Card.Root>
		</div>
	{/if}
	<div class="flex w-full max-w-md flex-row-reverse gap-2">
		{#if errorCode === 'email_verification_required'}
			<Button class="flex-1" href="/settings/account">{m.verify_email_address()}</Button>
		{:else}
			<Button class="flex-1" {isLoading} onclick={handlePipeline} autofocus={true}>
				{errorMessage ? m.try_again() : m.sign_in()}
			</Button>
		{/if}
		<Button class="flex-1" variant="secondary" href={document.referrer || '/'}>
			{m.cancel()}
		</Button>
	</div>
</SignInWrapper>
