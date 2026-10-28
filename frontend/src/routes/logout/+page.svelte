<script lang="ts">
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass, authQuietClass } from '#lib/components/auth/auth-util.ts';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.js';

	const webauthnService = new WebAuthnService();
	const user = $userStore!;
	const name = user.displayName || user.username;

	let isLoading = $state(false);
	let failed = $state(false);
	let signedOut = $state(false);

	async function signOut() {
		isLoading = true;
		failed = false;
		try {
			await webauthnService.logout();
			signedOut = true;
		} catch {
			failed = true;
		}
		isLoading = false;
	}
</script>

<svelte:head>
	<title>{m.sign_out()}</title>
</svelte:head>

<SignInWrapper>
	<AuthTile />
	{#if signedOut}
		<AuthHeading title={m.auth_you_are_signed_out()}>
			{m.auth_signed_out_description()}
		</AuthHeading>
		<Button class={authPrimaryClass} href="/login">{m.auth_sign_in_again()}</Button>
	{:else}
		<AuthHeading
			title={m.auth_sign_out_question({ appName: $appConfigStore.appName, name })}
			tone={failed ? 'danger' : undefined}
		>
			{failed
				? m.auth_sign_out_failed()
				: m.auth_sign_out_requested({ appName: $appConfigStore.appName })}
		</AuthHeading>
		<div class="mb-6">
			<IdentityRow {user} />
		</div>
		<div class="flex flex-col items-center gap-3">
			<Button class={authPrimaryClass} {isLoading} onclick={signOut}>
				{failed ? m.try_again() : m.sign_out()}
			</Button>
			<Button variant="link" class={authQuietClass} onclick={() => history.back()}>
				{m.auth_stay_signed_in()}
			</Button>
		</div>
	{/if}
</SignInWrapper>
