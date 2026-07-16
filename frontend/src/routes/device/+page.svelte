<script lang="ts">
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import ScopeList from '$lib/components/scope-list.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as InputOTP from '$lib/components/ui/input-otp';
	import { Spinner } from '$lib/components/ui/spinner';
	import { m } from '$lib/paraglide/messages';
	import DeviceLoginService from '$lib/services/device-login-service';
	import OIDCService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import type { DeviceLoginVerificationInfo } from '$lib/types/device-login.type';
	import type { OidcDeviceCodeInfo } from '$lib/types/oidc.type';
	import { getWebauthnErrorMessage } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import ClientProviderImages from '../authorize/components/client-provider-images.svelte';
	import LoginLogoErrorSuccessIndicator from '../login/components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const deviceLoginService = new DeviceLoginService();
	const oidcService = new OIDCService();
	const webauthnService = new WebAuthnService();

	let userCode = $state(data.code || '');
	let isLoading = $state(false);
	let deviceInfo: OidcDeviceCodeInfo | undefined = $state();
	let deviceLoginInfo: DeviceLoginVerificationInfo | undefined = $state();
	let success = $state(false);
	let deviceLoginOutcome: 'approved' | 'denied' | undefined = $state();
	let deviceLoginDecision: 'approve' | 'deny' | undefined = $state();
	let errorMessage: string | null = $state(null);
	let authorizationRequired = $state(false);
	let reauthenticationRequired = $state(false);
	let reauthenticated = $state(false);
	let normalizedUserCode = $derived(userCode.trim().toUpperCase());
	let codeComplete = $derived(normalizedUserCode.length === 8);
	let completed = $derived(success || deviceLoginOutcome !== undefined);

	onMount(() => {
		if (data.code && $userStore) {
			authorize();
		}
	});

	async function authorize() {
		if (!data.code && !codeComplete) return;

		isLoading = true;
		errorMessage = null;
		try {
			await authenticateUserIfNeeded();

			let isDeviceLoginCode = normalizedUserCode.startsWith('P');
			if (isDeviceLoginCode) {
				deviceLoginInfo = await deviceLoginService.inspectRequest(normalizedUserCode);
				return;
			}

			const info = await oidcService.getDeviceCodeInfo(normalizedUserCode);
			deviceInfo = info;

			if (info.authorizationRequired && !authorizationRequired) {
				authorizationRequired = true;
				return;
			}

			if (info.reauthenticationRequired && !reauthenticationRequired && !authorizationRequired) {
				reauthenticationRequired = true;
				return;
			}

			if (info.reauthenticationRequired && !reauthenticated) {
				await reauthenticate();
				reauthenticated = true;
			}

			await oidcService.verifyDeviceCode(normalizedUserCode);
			success = true;
		} catch (error) {
			errorMessage = getWebauthnErrorMessage(error);
		} finally {
			isLoading = false;
		}
	}

	async function authenticateUserIfNeeded() {
		if ($userStore) return;

		const loginOptions = await webauthnService.getLoginOptions();
		const authResponse = await startAuthentication({ optionsJSON: loginOptions });
		const user = await webauthnService.finishLogin(authResponse);
		await userStore.setUser(user);
	}

	async function decideDeviceLogin(decision: 'approve' | 'deny') {
		isLoading = true;
		deviceLoginDecision = decision;
		errorMessage = null;
		try {
			if (decision === 'approve') {
				await reauthenticate();
			}
			await deviceLoginService.decideRequest(normalizedUserCode, decision);
			deviceLoginOutcome = decision === 'approve' ? 'approved' : 'denied';
		} catch (error) {
			errorMessage = getWebauthnErrorMessage(error);
		} finally {
			isLoading = false;
			deviceLoginDecision = undefined;
		}
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

	function retry() {
		errorMessage = null;
		if (!deviceLoginInfo) {
			deviceInfo = undefined;
			authorizationRequired = false;
			reauthenticationRequired = false;
		}
	}
</script>

<svelte:head>
	<title>{m.authorize_device()}</title>
</svelte:head>

<SignInWrapper showAlternativeSignInMethodButton={$userStore == null}>
	<div class="flex justify-center">
		{#if deviceInfo?.client}
			<ClientProviderImages client={deviceInfo.client} {success} error={!!errorMessage} />
		{:else}
			<LoginLogoErrorSuccessIndicator
				success={success || deviceLoginOutcome === 'approved'}
				error={!!errorMessage}
			/>
		{/if}
	</div>
	<h1 class="font-gloock mt-5 text-4xl font-bold">
		{m.authorize_device()}
	</h1>
	{#if errorMessage}
		<p class="text-muted-foreground mt-2">
			{errorMessage}. {m.please_try_again()}
		</p>
	{:else if deviceLoginOutcome === 'approved'}
		<p class="text-muted-foreground mt-2">{m.the_requesting_device_has_been_signed_in()}</p>
	{:else if deviceLoginOutcome === 'denied'}
		<p class="text-muted-foreground mt-2">{m.the_sign_in_request_was_denied()}</p>
	{:else if success}
		<p class="text-muted-foreground mt-2">{m.the_device_has_been_authorized()}</p>
	{:else if deviceLoginInfo}
		<p class="text-muted-foreground mt-2">{m.review_the_request_before_approving_it()}</p>
		<div class="w-full max-w-112.5" transition:slide={{ duration: 300 }}>
			<Card.Root class="mt-6 text-start">
				<Card.Content>
					<dl class="flex flex-col gap-4 text-sm">
						<div class="flex items-start justify-between gap-6">
							<dt class="text-muted-foreground">{m.code()}</dt>
							<dd class="font-medium">
								{deviceLoginInfo.userCode.substring(0, 4)} - {deviceLoginInfo.userCode.substring(
									4,
									8
								)}
							</dd>
						</div>
						<div class="flex items-start justify-between gap-6">
							<dt class="text-muted-foreground">{m.device()}</dt>
							<dd class="text-right font-medium">{deviceLoginInfo.device}</dd>
						</div>
						<div class="flex items-start justify-between gap-6">
							<dt class="text-muted-foreground">{m.ip_address()}</dt>
							<dd class="font-medium">{deviceLoginInfo.ipAddress || m.unknown()}</dd>
						</div>
					</dl>
				</Card.Content>
			</Card.Root>
		</div>
	{:else if reauthenticationRequired && deviceInfo?.client}
		<p class="text-muted-foreground mt-2">
			<FormattedMessage
				m={m.do_you_want_to_sign_in_to_client_with_your_app_name_account({
					client: deviceInfo.client.name,
					appName: $appConfigStore.appName
				})}
			/>
		</p>
	{:else if authorizationRequired}
		<div class="w-full max-w-112.5" transition:slide={{ duration: 300 }}>
			<Card.Root class="mt-6 gap-2">
				<Card.Header>
					<Card.Description class="text-start">
						<FormattedMessage
							m={m.client_wants_to_access_the_following_information({
								client: deviceInfo!.client.name
							})}
						/>
					</Card.Description>
				</Card.Header>
				<Card.Content data-testid="scopes">
					<ScopeList scopes={deviceInfo!.scope || []} scopeInfo={deviceInfo!.scopeInfo || []} />
				</Card.Content>
			</Card.Root>
		</div>
	{:else}
		<p class="text-muted-foreground mt-2">{m.enter_code_displayed_in_previous_step()}</p>
		<form
			id="device-code-form"
			onsubmit={preventDefault(authorize)}
			class="mt-7 flex w-full max-w-112.5 justify-center"
		>
			<InputOTP.Root
				maxlength={8}
				aria-label={m.code()}
				bind:value={userCode}
				onValueChange={(value) => (userCode = value.toUpperCase())}
				pasteTransformer={(value) => value.replace(/[^a-zA-Z0-9]/g, '').toUpperCase()}
			>
				{#snippet children({ cells })}
					<InputOTP.Group>
						{#each cells.slice(0, 4) as cell}
							<InputOTP.Slot {cell} />
						{/each}
					</InputOTP.Group>
					<InputOTP.Separator />
					<InputOTP.Group>
						{#each cells.slice(4) as cell}
							<InputOTP.Slot {cell} />
						{/each}
					</InputOTP.Group>
				{/snippet}
			</InputOTP.Root>
		</form>
	{/if}
	{#if !completed}
		<div class="mt-10 flex w-full max-w-112.5 gap-2">
			{#if errorMessage}
				<Button class="flex-1" variant="secondary" href="/">{m.cancel()}</Button>
				<Button class="flex-1" onclick={retry}>{m.try_again()}</Button>
			{:else if deviceLoginInfo}
				<Button
					class="flex-1"
					variant="secondary"
					disabled={isLoading}
					onclick={() => decideDeviceLogin('deny')}
				>
					{#if deviceLoginDecision === 'deny'}<Spinner data-icon="inline-start" />{/if}
					{m.deny()}
				</Button>
				<Button class="flex-1" {isLoading} onclick={() => decideDeviceLogin('approve')}>
					{m.approve()}
				</Button>
			{:else}
				<Button href="/" class="flex-1" variant="secondary">{m.cancel()}</Button>
				<Button
					form="device-code-form"
					class="flex-1"
					disabled={isLoading || !codeComplete}
					onclick={authorize}
				>
					{#if isLoading}<Spinner data-icon="inline-start" />{/if}
					{m.authorize()}
				</Button>
			{/if}
		</div>
	{/if}
</SignInWrapper>
