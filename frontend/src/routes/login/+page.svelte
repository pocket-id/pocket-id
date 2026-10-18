<script lang="ts">
	import { goto } from '$app/navigation';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		alternativeSignInHref,
		authPrimaryClass,
		authQuietClass,
		isPasskeyPromptClosed,
		successHold,
		takingYouMessage
	} from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';

	let { data } = $props();

	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let success = $state(false);
	let failure: { message: string; closed: boolean } | undefined = $state();
	let primary = $state<HTMLElement | null>(null);

	// Enter works right away, but the ring only shows once a key is pressed
	onMount(() => primary?.focus({ focusVisible: false }));

	// Chrome keeps a focus taken without a ring ringless through later key presses, so the first one focuses it again
	function showFocusRing() {
		if (!primary || document.activeElement !== primary || primary.matches(':focus-visible')) return;
		primary.blur();
		primary.focus();
	}

	async function authenticate() {
		failure = undefined;
		isLoading = true;
		try {
			const loginOptions = await webauthnService.getLoginOptions();
			const authResponse = await startAuthentication({ optionsJSON: loginOptions });
			const user = await webauthnService.finishLogin(authResponse);

			await userStore.setUser(user);
			success = true;
			await successHold();
			await goto(data.redirect || '/settings');
		} catch (e) {
			failure = { message: getWebauthnErrorMessage(e), closed: isPasskeyPromptClosed(e) };
		}
		isLoading = false;
	}
</script>

<svelte:head>
	<title>{m.sign_in()}</title>
</svelte:head>

<svelte:window onkeydown={showFocusRing} />

<SignInWrapper>
	<AuthTile {success} />
	<AuthHeading
		title={success
			? m.auth_signed_in()
			: m.sign_in_to_appname({ appName: $appConfigStore.appName })}
		tone={failure ? (failure.closed ? 'info' : 'danger') : undefined}
	>
		{#if success}
			{takingYouMessage(data.redirect)}
		{:else if failure}
			{failure.closed ? m.auth_passkey_prompt_closed() : failure.message}
		{:else}
			{m.auth_sign_in_subtitle()}
			{#if $appConfigStore.allowUserSignups === 'open'}
				{m.auth_new_here()}
				<a href="/signup" class="inline-link">{m.auth_create_an_account()}</a>
			{/if}
		{/if}
	</AuthHeading>
	<div
		class="flex flex-col items-center gap-3 transition-opacity duration-(--dur-fast)"
		class:opacity-0={success}
		inert={success}
	>
		<Button bind:ref={primary} class={authPrimaryClass} {isLoading} onclick={authenticate}>
			{failure ? m.try_again() : m.auth_sign_in_with_passkey()}
		</Button>
		<Button variant="link" class={authQuietClass} href={alternativeSignInHref()}>
			{m.auth_other_ways_to_sign_in()}
		</Button>
	</div>
</SignInWrapper>
