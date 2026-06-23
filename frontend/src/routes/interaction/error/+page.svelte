<script lang="ts">
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import userStore from '$lib/stores/user-store';
	import ClientProviderImages from '../../authorize/components/client-provider-images.svelte';
	import type { PageProps } from './$types';

	const webauthnService = new WebAuthnService();
	const oidcService = new OidcService();

	let { data }: PageProps = $props();
	let { error } = data;
</script>

<svelte:head>
	<title>{m.error()}</title>
</svelte:head>

<SignInWrapper>
	<ClientProviderImages error success={false} />
	<h1 class="font-gloock mt-5 text-3xl font-bold sm:text-4xl">
		{m.error()}
	</h1>
	<p class="text-muted-foreground mt-2 mb-10">
		{error}
	</p>

	<Button class="w-full sm:w-[50%]" variant="secondary" href={document.referrer || '/'}>
		{m.go_back()}
	</Button>
</SignInWrapper>
