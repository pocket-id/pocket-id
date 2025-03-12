<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store.js';
	import userStore from '$lib/stores/user-store.js';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import LoginLogoErrorSuccessIndicator from '../components/login-logo-error-success-indicator.svelte';

	let isLoading = $state(false);
	let error: string | undefined = $state();

	const userService = new UserService();

	async function authenticate() {
		isLoading = true;
		try {
			const user = await userService.exchangeOneTimeAccessToken('setup');
			userStore.setUser(user);

			goto('/settings');
		} catch (e) {
			error = getAxiosErrorMessage(e);
		}

		isLoading = false;
	}
</script>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">
		{`${$appConfigStore.appName} Setup`}
	</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. Please try again.
		</p>
	{:else}
		<p class="text-muted-foreground mt-2">
			You're about to sign in to the initial admin account. Anyone with this link can access the
			account until a passkey is added. Please set up a passkey as soon as possible to prevent
			unauthorized access.
		</p>
		<Button class="mt-5" {isLoading} on:click={authenticate}>Continue</Button>
	{/if}
</SignInWrapper>
