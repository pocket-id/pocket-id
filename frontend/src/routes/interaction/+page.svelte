<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import ActionPanel from '#lib/components/auth/action-panel.svelte';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import {
		alternativeSignInHref,
		authPrimaryClass,
		authQuietClass,
		isPasskeyPromptClosed,
		successHold
	} from '#lib/components/auth/auth-util.ts';
	import AuthorizeTiles from '#lib/components/auth/authorize-tiles.svelte';
	import TrustLine from '#lib/components/auth/trust-line.svelte';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import ScopeList from '#lib/components/scope-list.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Spinner } from '#lib/components/ui/spinner/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { InteractionStep } from '#lib/types/oidc.type.ts';
	import { getClientIDHost } from '#lib/utils/client-id-util.ts';
	import { getAxiosErrorCode, getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { LucideChevronRight } from '@lucide/svelte';
	import type { PageProps } from './$types';

	const webauthnService = new WebAuthnService();
	const oidcService = new OidcService();

	let { data }: PageProps = $props();
	let interactionSession = $state(data.interactionSession);
	let isLoading = $state(false);
	// Signing in keeps the sign-in step on screen until the next step is known, so the card changes size once
	let signingIn = $state(false);
	let redirectUrl: string | undefined = $state();
	let showFallback = $state(false);
	let failure: { message: string; code?: string; closed: boolean } | undefined = $state();
	let currentStep = $derived(interactionSession.currentStep);
	// Every requested scope starts selected so the user only has to deselect the ones they do not want to grant
	let selectedScopes = $state([...(data.interactionSession.scopes ?? [])]);

	const client = $derived(interactionSession.client);
	// Metadata-document clients go by their host in body copy, because their display name is self-declared
	const clientName = $derived(getClientIDHost(client) ?? client.name);
	// openid and offline_access show nothing, so a request for only those keeps the compact card
	const hasScopes = $derived(
		(interactionSession.scopes ?? []).some((s) => s !== 'openid' && s !== 'offline_access')
	);
	const emailVerificationRequired = $derived(failure?.code === 'email_verification_required');
	const view = $derived(
		redirectUrl
			? 'success'
			: !$userStore || signingIn
				? 'sign-in'
				: emailVerificationRequired
					? 'verify'
					: (currentStep ?? 'authenticate')
	);
	const cancelHref = document.referrer || '/';

	async function useDifferentAccount() {
		await webauthnService.logout();
		await completeInteraction('select_account', true);
		await refreshAll();
	}

	async function handlePipeline() {
		isLoading = true;
		signingIn = !$userStore;
		failure = undefined;
		try {
			if (!$userStore) {
				await authenticate();
			}

			if (currentStep === 'reauthenticate') {
				await reauthenticate();
			}

			await completeInteraction(currentStep!);
		} catch (e) {
			failure = {
				message: getWebauthnErrorMessage(e),
				code: getAxiosErrorCode(e),
				closed: isPasskeyPromptClosed(e)
			};
		} finally {
			isLoading = false;
			signingIn = false;
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
			redirectUrl = result.redirectUrl;
			// The way forward becomes a link when the browser has not left after a few seconds
			setTimeout(() => (showFallback = true), cssMs('--t-redirect-fallback'));
			await successHold();
			window.location.href = result.redirectUrl;
		}
	}
</script>

<svelte:head>
	<title>{m.sign_in_to({ name: client.name })}</title>
</svelte:head>

<SignInWrapper wide={view === 'consent' && hasScopes}>
	<AuthorizeTiles {client} success={!!redirectUrl} error={!!failure && !failure.closed} />

	{#if view === 'success'}
		<AuthHeading title={m.auth_signed_in()}>
			{m.auth_taking_you_to({ name: clientName })}
		</AuthHeading>
		<div
			class="transition-opacity duration-(--dur-base)"
			class:invisible={!showFallback}
			class:opacity-0={!showFallback}
		>
			<Button variant="link" class="text-body h-(--h-xl) max-w-full gap-1" href={redirectUrl}>
				<span class="truncate">{m.auth_continue_to({ name: clientName })}</span>
				<LucideChevronRight />
			</Button>
		</div>
	{:else}
		<AuthHeading
			title={view === 'select_account'
				? m.auth_choose_an_account()
				: view === 'reauthenticate'
					? m.auth_confirm_its_you()
					: client.name}
			tone={failure ? (failure.closed ? 'info' : 'danger') : undefined}
		>
			{#if failure}
				{failure.closed ? m.auth_passkey_prompt_closed() : failure.message}
			{:else if view === 'select_account'}
				{m.auth_to_continue_to({ name: clientName })}
			{:else if view === 'reauthenticate'}
				{m.auth_confirm_with_passkey({ name: clientName })}
			{:else}
				{m.auth_wants_to_sign_you_in({ appName: $appConfigStore.appName })}
			{/if}
		</AuthHeading>

		{#if view === 'consent' || view === 'authenticate'}
			<TrustLine {client} />
		{/if}

		{#if view === 'select_account'}
			<div class="mb-6" data-testid="account-selection">
				<button
					type="button"
					class="group/account focus-visible:outline-ring after:bg-fg/4 transition-motion relative block w-full rounded-md text-start after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:transition-opacity after:duration-(--dur-fast) hover:after:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 active:scale-(--press-scale)"
					aria-busy={isLoading || undefined}
					disabled={isLoading}
					onclick={handlePipeline}
				>
					<IdentityRow user={$userStore!}>
						<span class="text-fg-muted shrink-0">
							{#if isLoading}
								<Spinner class="size-4" />
							{:else}
								<LucideChevronRight
									class="size-4 transition-transform duration-(--dur-fast) group-hover/account:translate-x-(--nudge)"
								/>
							{/if}
						</span>
					</IdentityRow>
				</button>
			</div>
		{:else if $userStore && view !== 'verify' && view !== 'sign-in'}
			<div class="mb-6">
				<IdentityRow user={$userStore} />
			</div>
		{/if}

		{#if view === 'consent' && hasScopes}
			<div class="mb-6 lg:break-before-column">
				<ScopeList
					scopes={interactionSession.scopes || []}
					scopeInfo={interactionSession.scopeInfo ?? []}
					bind:selectedScopes
				/>
			</div>
		{/if}

		<ActionPanel>
			<div class="flex flex-col items-center gap-3">
				{#if view === 'verify'}
					<Button class={authPrimaryClass} href="/settings/account">
						{m.verify_email_address()}
					</Button>
				{:else if view === 'sign-in'}
					<Button class={authPrimaryClass} {isLoading} onclick={handlePipeline}>
						{failure ? m.try_again() : m.auth_sign_in_with_passkey()}
					</Button>
					<Button variant="link" class={authQuietClass} href={alternativeSignInHref()}>
						{m.auth_other_ways_to_sign_in()}
					</Button>
				{:else if view === 'select_account'}
					<Button variant="link" class={authQuietClass} onclick={useDifferentAccount}>
						{m.use_a_different_account()}
					</Button>
				{:else}
					<!-- A metadata-document client goes by its host, which can be longer than the button -->
					<Button class={authPrimaryClass} {isLoading} onclick={handlePipeline}>
						<span class="truncate">
							{failure
								? m.try_again()
								: view === 'reauthenticate'
									? m.auth_continue_with_passkey()
									: m.auth_continue_to({ name: clientName })}
						</span>
					</Button>
				{/if}
				<!-- Cancel goes back through authorize, so the app hears access_denied instead of the person being left on Pocket ID -->
				<Button
					variant="link"
					class={authQuietClass}
					href={`/authorize?interaction=${encodeURIComponent(interactionSession.id)}&cancel=true`}
				>
					{m.cancel()}
				</Button>
			{/if}
			<Button variant="link" class={authQuietClass} href={cancelHref}>{m.cancel()}</Button>
		</div>

			{#if view === 'consent'}
				<p class="text-control text-fg-muted mt-2 text-center">{m.auth_remove_access_anytime()}</p>
			{/if}
		</ActionPanel>
	{/if}
</SignInWrapper>
