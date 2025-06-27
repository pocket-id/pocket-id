<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import SignupForm from '$lib/components/signup/signup-form.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import type { UserSignUp } from '$lib/types/user.type';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { tryCatch } from '$lib/utils/try-catch-util';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from '../login/components/login-logo-error-success-indicator.svelte';

	let { data } = $props();
	const userService = new UserService();

	let isLoading = $state(false);
	let error: string | undefined = $state();

	async function handleSignup(userData: UserSignUp) {
		isLoading = true;

		const result = await tryCatch(userService.signup({ ...userData, token: data.token }));

		if (result.error) {
			error = getAxiosErrorMessage(result.error);
			isLoading = false;
			return false;
		}

		userStore.setUser(result.data);
		isLoading = false;

		goto('/signup/add-passkey');
		return true;
	}

	onMount(() => {
		if (!$appConfigStore.allowUserSignups || $appConfigStore.allowUserSignups === 'disabled') {
			error = m.user_signups_are_disabled();
			return;
		}

		// For token-based signups, check if we have a valid token
		if ($appConfigStore.allowUserSignups === 'withToken' && !data.token) {
			error = m.signup_requires_valid_token();
		}
	});
</script>

<svelte:head>
	<title>{m.signup()}</title>
</svelte:head>

<SignInWrapper animate={!$appConfigStore.disableAnimations}>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>

	<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">
		{m.signup_to_appname({ appName: $appConfigStore.appName })}
	</h1>

	{#if !error}
		<p class="text-muted-foreground mt-2" in:fade>
			{m.create_your_account_to_get_started()}
		</p>
	{:else}
		<p class="text-muted-foreground mt-2" in:fade>
			{error}.
		</p>
	{/if}
	{#if $appConfigStore.allowUserSignups === 'open' || data.token}
		<SignupForm callback={handleSignup} {isLoading} />
		<div class="mt-10 flex w-full items-center justify-between gap-2">
			<a class="text-muted-foreground mt-5 flex text-sm" href="/login"
				><LucideChevronLeft class="size-5" /> {m.back()}</a
			>
			<Button type="submit" form="sign-up-form" onclick={() => (error = undefined)}
				>{m.signup()}</Button
			>
		</div>
	{:else}
		<Button class="mt-10" href="/login">{m.go_to_login()}</Button>
	{/if}
</SignInWrapper>
