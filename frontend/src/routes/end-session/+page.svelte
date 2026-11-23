<script lang="ts">
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass, authQuietClass } from '#lib/components/auth/auth-util.ts';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const oidcService = new OidcService();

	let isLoading = $state(false);
	let failed = $state(false);
	let signedOut = $state(false);

	async function signOut() {
		isLoading = true;
		failed = false;
		try {
			const redirectUrl = await oidcService.confirmEndSession(data.params);
			if (redirectUrl) {
				window.location.href = redirectUrl;
				return;
			}
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
	<AuthTile error={!!data.error || failed} />
	{#if data.error}
		<AuthHeading title={m.auth_could_not_sign_you_out()}>{data.error}</AuthHeading>
		<Button class={authPrimaryClass} onclick={() => history.back()}>{m.go_back()}</Button>
	{:else if signedOut}
		<AuthHeading title={m.auth_you_are_signed_out()}>
			{m.auth_signed_out_description()}
		</AuthHeading>
		<Button class={authPrimaryClass} href="/login">{m.auth_sign_in_again()}</Button>
	{:else}
		<AuthHeading title={m.auth_app_sign_out_question({ appName: $appConfigStore.appName })}>
			{failed
				? m.auth_sign_out_failed()
				: m.auth_app_sign_out_requested({
						clientName: data.clientName!,
						appName: $appConfigStore.appName
					})}
		</AuthHeading>
		{#if $userStore}
			<div class="mb-6">
				<IdentityRow user={$userStore} />
			</div>
		{/if}
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
