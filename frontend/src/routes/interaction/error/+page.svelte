<script lang="ts">
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import { authPrimaryClass, authQuietClass } from '#lib/components/auth/auth-util.ts';
	import AuthorizeTiles from '#lib/components/auth/authorize-tiles.svelte';
	import IdentityRow from '#lib/components/identity-row.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { isEmailVerificationPending } from '#lib/utils/email-verification-util.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const backHref = document.referrer || '/';
</script>

<svelte:head>
	<title>{m.auth_could_not_sign_you_in()}</title>
</svelte:head>

<SignInWrapper>
	<AuthorizeTiles error />
	<AuthHeading title={m.auth_could_not_sign_you_in()}>{data.error}</AuthHeading>
	{#if $userStore}
		<div class="mb-6">
			<IdentityRow user={$userStore} />
		</div>
	{/if}
	<div class="flex flex-col items-center gap-3">
		{#if isEmailVerificationPending($userStore, $appConfigStore)}
			<Button class={authPrimaryClass} href="/settings/account">
				{m.verify_email_address()}
			</Button>
			<Button variant="link" class={authQuietClass} href={backHref}>{m.go_back()}</Button>
		{:else}
			<Button class={authPrimaryClass} href={backHref}>{m.go_back()}</Button>
		{/if}
	</div>
</SignInWrapper>
