<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import Input from '$lib/components/ui/input/input.svelte';
	import UserService from '$lib/services/user-service';
	import userStore from '$lib/stores/user-store.js';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { onMount } from 'svelte';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';
	import { page } from '$app/state';

	let { data } = $props();
	let code = $state(data.code ?? '');
	let isLoading = $state(false);
	let error: string | undefined = $state();

	const userService = new UserService();

	async function authenticate() {
		isLoading = true;
		try {
			const user = await userService.exchangeOneTimeAccessToken(code);
			userStore.setUser(user);

			try {
				goto(data.redirect);
			} catch (e) {
				error = 'Invalid redirect URL';
			}
		} catch (e) {
			error = getAxiosErrorMessage(e);
		}

		isLoading = false;
	}

	onMount(() => {
		if (code) {
			authenticate();
		}
	});
</script>

<svelte:head>
	<title>Login Code</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">Login Code</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. Please try again.
		</p>
	{:else}
		<p class="text-muted-foreground mt-2">Enter the code you received to sign in.</p>
	{/if}
	<form
		onsubmit={(e) => {
			e.preventDefault();
			authenticate();
		}}
		class="w-full max-w-[450px]"
	>
		<Input id="Email" class="mt-7" placeholder="Code" bind:value={code} type="text" />
		<div class="mt-8 flex justify-stretch gap-2">
			<Button variant="secondary" class="w-full" href={"/login/alternative" + page.url.search}>Go back</Button>
			<Button class="w-full" type="submit" {isLoading}>Submit</Button>
		</div>
	</form>
</SignInWrapper>
