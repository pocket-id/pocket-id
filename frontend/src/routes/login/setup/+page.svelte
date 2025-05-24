<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
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

<SignInWrapper animate={!$appConfigStore.disableAnimations}>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">
		{m.appname_setup({ appName: $appConfigStore.appName })}
	</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. {m.please_try_again()}
		</p>
	{:else}
		<p class="text-muted-foreground mt-2">
			{m.you_are_about_to_sign_in_to_the_initial_admin_account()}
		</p>
		<Button class="mt-5" {isLoading} onclick={authenticate}>{m.continue()}</Button>
	{/if}
</SignInWrapper>
