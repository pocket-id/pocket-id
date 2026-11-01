<script lang="ts">
	import ActionPanel from '#lib/components/auth/action-panel.svelte';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		alternativeSignInHref,
		authPrimaryClass,
		authQuietClass,
		isPasskeyPromptClosed
	} from '#lib/components/auth/auth-util.ts';
	import AuthorizeTiles from '#lib/components/auth/authorize-tiles.svelte';
	import CodeGroup from '#lib/components/auth/code-group.svelte';
	import CodeLine from '#lib/components/auth/code-line.svelte';
	import TrustLine from '#lib/components/auth/trust-line.svelte';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import ScopeList from '#lib/components/scope-list.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Label } from '#lib/components/ui/label/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import DeviceLoginService from '#lib/services/device-login-service.ts';
	import OIDCService from '#lib/services/oidc-service.ts';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { DeviceLoginVerificationInfo } from '#lib/types/device-login.type.ts';
	import type { OidcDeviceCodeInfo } from '#lib/types/oidc.type.ts';
	import { getClientIDHost } from '#lib/utils/client-id-util.ts';
	import { getAxiosErrorCode, getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount, untrack } from 'svelte';

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
	let failure: { message: string; code?: string; closed: boolean } | undefined = $state();
	let authorizationRequired = $state(false);
	let selectedScopes: string[] = $state([]);
	let reauthenticationRequired = $state(false);
	let reauthenticated = $state(false);
	let normalizedUserCode = $derived(
		userCode.trim().toUpperCase().replaceAll('I', '1').replaceAll('O', '0')
	);
	let codeComplete = $derived(normalizedUserCode.length === 8);
	let signedInName = $derived($userStore?.displayName || $userStore?.username || '');
	let deviceLoginLocation = $derived(
		deviceLoginInfo?.city && deviceLoginInfo.country
			? `${deviceLoginInfo.city}, ${deviceLoginInfo.country}`
			: deviceLoginInfo?.city || deviceLoginInfo?.country || m.unknown()
	);

	const client = $derived(deviceInfo?.client);
	const clientName = $derived(client ? (getClientIDHost(client) ?? client.name) : '');
	const view = $derived(
		deviceLoginOutcome
			? deviceLoginOutcome
			: success
				? 'authorized'
				: deviceLoginInfo
					? 'review'
					: client && (authorizationRequired || reauthenticationRequired)
						? 'consent'
						: 'enter'
	);
	// Failures of the code itself show under the cells, everything else in the status line
	const codeFailure = $derived(
		view === 'enter' &&
			(failure?.code === 'oidc_device_code_expired'
				? 'expired'
				: failure?.code === 'oidc_invalid_device_code' || failure?.code === 'device_login_expired'
					? 'invalid'
					: undefined)
	);
	const statusFailure = $derived(codeFailure ? undefined : failure);
	const statusTone = $derived(
		statusFailure ? (statusFailure.closed ? 'info' : 'danger') : undefined
	);

	// Typing again clears the last outcome
	$effect(() => {
		void userCode;
		untrack(() => (failure = undefined));
	});

	onMount(() => {
		// Device login codes always show a confirmation before approval
		if (data.code && normalizedUserCode.startsWith('P') && $userStore) {
			authorize();
		}
	});

	async function authorize() {
		if (!codeComplete) return;
		if (codeFailure === 'expired') {
			userCode = '';
			return;
		}

		isLoading = true;
		failure = undefined;
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
				// Every requested scope starts selected so the user only has to deselect the ones they do not want to grant
				selectedScopes = [...(info.scope ?? [])];
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

			await oidcService.verifyDeviceCode(
				normalizedUserCode,
				authorizationRequired ? selectedScopes : undefined
			);
			success = true;
		} catch (error) {
			setFailure(error);
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
		failure = undefined;
		try {
			if (decision === 'approve') {
				await reauthenticate();
			}
			await deviceLoginService.decideRequest(normalizedUserCode, decision);
			deviceLoginOutcome = decision === 'approve' ? 'approved' : 'denied';
		} catch (error) {
			setFailure(error);
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

	function setFailure(error: unknown) {
		failure = {
			message: getWebauthnErrorMessage(error),
			code: getAxiosErrorCode(error),
			closed: isPasskeyPromptClosed(error)
		};
	}
</script>

<svelte:head>
	<title>{m.auth_connect_a_device()}</title>
</svelte:head>

{#snippet status(fallback: string)}
	{#if statusFailure}
		{statusFailure.closed ? m.auth_passkey_prompt_closed() : statusFailure.message}
	{:else}
		{fallback}
	{/if}
{/snippet}

<SignInWrapper wide={view === 'consent' && authorizationRequired}>
	{#if client && (view === 'consent' || view === 'authorized')}
		<AuthorizeTiles
			{client}
			success={view === 'authorized'}
			error={!!statusFailure && !statusFailure.closed}
		/>
	{:else}
		<AuthTile success={view === 'approved'} />
	{/if}

	{#if view === 'authorized'}
		<AuthHeading title={m.auth_authorized()}>
			{m.auth_device_authorized_description({ name: clientName })}
		</AuthHeading>
	{:else if view === 'approved'}
		<AuthHeading title={m.auth_signed_in()}>
			{m.auth_other_device_signed_in({ device: deviceLoginInfo!.device, name: signedInName })}
		</AuthHeading>
	{:else if view === 'denied'}
		<AuthHeading title={m.auth_request_denied()}>
			{m.auth_other_device_denied({ device: deviceLoginInfo!.device })}
		</AuthHeading>
	{:else if view === 'review'}
		<AuthHeading
			title={m.auth_device_wants_to_sign_in({ device: deviceLoginInfo!.device })}
			tone={statusTone}
		>
			{@render status(m.auth_does_this_code_match())}
		</AuthHeading>
		<div
			class="mb-4 flex items-center font-mono text-xl font-medium"
			data-testid="device-login-code"
		>
			{#each [deviceLoginInfo!.userCode.slice(0, 4), deviceLoginInfo!.userCode.slice(4)] as group, i (i)}
				{#if i > 0}
					<span class="text-fg-muted w-7 shrink-0 text-center text-xs">–</span>
				{/if}
				<span class="bg-surface-sunken flex min-w-0 flex-1 rounded-sm">
					{#each group as char, j (j)}
						<span class="grid h-13 min-w-0 flex-1 place-items-center max-lg:h-11">{char}</span>
					{/each}
				</span>
			{/each}
		</div>
		<dl class="mb-6 flex flex-col gap-1.5">
			<div class="grid grid-cols-[100px_minmax(0,1fr)] items-baseline gap-2">
				<dt class="text-caption text-fg-muted">{m.auth_location()}</dt>
				<dd class="text-control">{deviceLoginLocation}</dd>
			</div>
			<div class="grid grid-cols-[100px_minmax(0,1fr)] items-baseline gap-2">
				<dt class="text-caption text-fg-muted">{m.auth_ip_address()}</dt>
				<dd class="text-mono-s font-mono">{deviceLoginInfo!.ipAddress || m.unknown()}</dd>
			</div>
		</dl>
		<div class="mb-6">
			<IdentityRow user={$userStore!} />
		</div>
		<!-- Refusing a security request is an answer, so Deny is a full-width button rather than a text action -->
		<ActionPanel>
			<div class="flex flex-col gap-3">
				<Button
					class={authPrimaryClass}
					disabled={isLoading}
					isLoading={deviceLoginDecision === 'approve'}
					onclick={() => decideDeviceLogin('approve')}
				>
					{statusFailure ? m.try_again() : m.approve()}
				</Button>
				<Button
					variant="secondary"
					class={authPrimaryClass}
					disabled={isLoading}
					isLoading={deviceLoginDecision === 'deny'}
					onclick={() => decideDeviceLogin('deny')}
				>
					{m.deny()}
				</Button>
			</div>
			<p class="text-control text-fg-muted mt-2 text-center">
				{m.auth_approving_asks_for_passkey()}
			</p>
		</ActionPanel>
	{:else if view === 'consent' && client}
		<AuthHeading title={client.name} tone={statusTone}>
			{@render status(m.auth_wants_to_use_account_on_device({ appName: $appConfigStore.appName }))}
		</AuthHeading>
		<TrustLine {client} />
		<div class="mb-6">
			<IdentityRow user={$userStore!} />
		</div>
		{#if authorizationRequired}
			<div class="mb-6 lg:break-before-column">
				<ScopeList
					scopes={deviceInfo!.scope || []}
					scopeInfo={deviceInfo!.scopeInfo || []}
					bind:selectedScopes
				/>
			</div>
		{/if}
		<ActionPanel>
			<div class="flex flex-col items-center gap-3">
				{#if failure?.code === 'email_verification_required'}
					<Button class={authPrimaryClass} href="/settings/account">
						{m.verify_email_address()}
					</Button>
				{:else}
					<Button class={authPrimaryClass} {isLoading} onclick={authorize}>
						{statusFailure ? m.try_again() : m.approve()}
					</Button>
				{/if}
				<Button variant="link" class={authQuietClass} href="/">{m.cancel()}</Button>
			</div>
		</ActionPanel>
	{:else}
		<AuthHeading title={m.auth_connect_a_device()} tone={statusTone}>
			{@render status(
				codeFailure === 'expired'
					? m.auth_device_code_expired_description()
					: m.auth_enter_device_code()
			)}
		</AuthHeading>
		<form onsubmit={preventDefault(authorize)} class="flex flex-col">
			<Label for="device-code" class="mb-2">{m.auth_device_code()}</Label>
			<CodeGroup
				id="device-code"
				groups={[4, 4]}
				uppercase
				bind:value={userCode}
				invalid={codeFailure === 'invalid'}
				expired={codeFailure === 'expired'}
				readonly={isLoading}
				aria-describedby="device-code-line"
				autofocus={!userCode}
			/>
			<CodeLine
				id="device-code-line"
				tone={codeFailure === 'invalid' ? 'error' : codeFailure === 'expired' ? 'info' : undefined}
			>
				{#if codeFailure === 'invalid'}
					{m.auth_code_did_not_work()}
				{:else if codeFailure === 'expired'}
					{m.auth_code_has_expired()}
				{/if}
			</CodeLine>
			<div class="mt-6 flex flex-col items-center gap-3">
				<Button type="submit" class={authPrimaryClass} disabled={!codeComplete} {isLoading}>
					{codeFailure === 'expired' ? m.auth_enter_a_new_code() : m.auth_continue()}
				</Button>
				{#if !$userStore}
					<Button variant="link" class={authQuietClass} href={alternativeSignInHref()}>
						{m.auth_other_ways_to_sign_in()}
					</Button>
				{/if}
				<Button variant="link" class={authQuietClass} href="/">{m.cancel()}</Button>
			</div>
		</form>
	{/if}
</SignInWrapper>
