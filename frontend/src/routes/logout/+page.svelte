<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import Logo from '$lib/components/logo.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import userStore from '$lib/stores/user-store.js';
	import { axiosErrorToast } from '$lib/utils/error-util.js';

	let isLoading = $state(false);

	const webauthnService = new WebAuthnService();

	async function signOut() {
		isLoading = true;
		await webauthnService
			.logout()
			.then(() => goto('/'))
			.catch(axiosErrorToast);
		isLoading = false;
	}
</script>

<svelte:head>
	<title>{m.logout()}</title>
</svelte:head>

<SignInWrapper animate>
	<div class="flex justify-center">
		<div class="bg-muted rounded-2xl p-3">
			<Logo class="h-10 w-10" />
		</div>
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">{m.sign_out()}</h1>

	<p class="text-muted-foreground mt-2">
		{@html m.do_you_want_to_sign_out_of_pocketid_with_the_account({
			username: $userStore?.username ?? ''
		})}
	</p>
	<div class="mt-10 flex w-full justify-stretch gap-2">
		<Button class="w-full" variant="secondary" onclick={() => history.back()}>{m.cancel()}</Button>
		<Button class="w-full" {isLoading} onclick={signOut}>{m.sign_out()}</Button>
	</div>
</SignInWrapper>
