<script lang="ts">
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from '../login/components/login-logo-error-success-indicator.svelte';
	import { onMount } from 'svelte';
	import SignupForm from '$lib/components/signup-form.svelte';

	let { data } = $props();
	const userService = new UserService();

	let isLoading = $state(false);
	let error: string | undefined = $state();

	async function handleSignup(userData: any) {
		if (!data.token) return false;

		isLoading = true;
		try {
			const user = await userService.signupWithToken(data.token, userData);
			userStore.setUser(user);
			return true;
		} catch (e) {
			error = getAxiosErrorMessage(e);
			return false;
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		if (!$appConfigStore.allowUserSignups || $appConfigStore.allowUserSignups === 'disabled') {
			error = m.user_signups_are_disabled();
			return;
		}

		if (!data.token) {
			error = m.signup_requires_valid_token();
		}
	});
</script>

<svelte:head>
	<title>{m.sign_up()}</title>
</svelte:head>

<SignInWrapper animate={!$appConfigStore.disableAnimations}>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>

	<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">
		{m.sign_up_to_appname({ appName: $appConfigStore.appName })}
	</h1>

	{#if error}
		<p class="text-muted-foreground mt-2" in:fade>
			{error}. {m.please_try_again()}
		</p>
		<div class="mt-10 flex justify-center">
			<Button href="/login">{m.go_to_login()}</Button>
		</div>
	{:else if data.token}
		<p class="text-muted-foreground mt-2" in:fade>
			{m.create_your_account_to_get_started()}
		</p>

		<SignupForm callback={handleSignup} {isLoading} />
	{/if}
</SignInWrapper>
