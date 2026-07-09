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
	import type { InteractionStep } from '$lib/types/oidc.type';
	import { cachedProfilePicture } from '$lib/utils/cached-image-util';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
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
	let currentStep = $derived(interactionSession.currentStep);

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

	async function useDifferentAccount() {
		await webauthnService.logout();
		await completeInteraction('select_account', true);
		userStore.clearUser();
		await invalidateAll();
	}

	async function handlePipeline() {
		isLoading = true;
		errorMessage = null;
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
		} finally {
			isLoading = false;
		}
	}

	async function authenticate() {
		const loginOptions = await webauthnService.getLoginOptions();
		const authResponse = await startAuthentication({ optionsJSON: loginOptions });
		const user = await webauthnService.finishLogin(authResponse);
		await userStore.setUser(user);
		await invalidateAll();
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
		const result = await oidcService.completeAuthorizeInteractionStep(interactionSession.id, step);
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
	<h1 class="font-gloock mt-5 text-3xl font-bold sm:text-4xl">
		{m.sign_in_to({ name: interactionSession.client.name })}
	</h1>
	<p class="text-muted-foreground mt-2 mb-10">
		{#if errorMessage}
			{errorMessage}.
		{:else if currentStep == 'select_account' && $userStore}
			<FormattedMessage
				m={m.account_selection_signin_confirmation({ name: interactionSession.client.name })}
			/>
		{:else}
			<FormattedMessage
				m={m.do_you_want_to_sign_in_to_client_with_your_app_name_account({
					client: interactionSession.client.name,
					appName: $appConfigStore.appName
				})}
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
				<Card.Root class="mb-2 py-4 w-sm">
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
							m={m.client_wants_to_access_the_following_information({
								client: interactionSession.client.name
							})}
						/>
					</p>
				</Card.Header>
				<Card.Content>
					<ScopeList
						scopes={(interactionSession.scopes || [])}
						scopeInfo={interactionSession.scopeInfo ?? []}
					/>
				</Card.Content>
			</Card.Root>
		</div>
	{/if}
	<div class="flex w-full max-w-md flex-row-reverse gap-2">
		<Button class="flex-1" {isLoading} onclick={handlePipeline} autofocus={true}>
			{errorMessage ? m.try_again() : m.sign_in()}
		</Button>
		<Button class="flex-1" variant="secondary" href={document.referrer || '/'}>
			{m.cancel()}
		</Button>
	</div>
</SignInWrapper>
