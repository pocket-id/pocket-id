<script lang="ts">
	import { page } from '$app/state';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import Input from '$lib/components/ui/input/input.svelte';
	import UserService from '$lib/services/user-service';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';

	const { data } = $props();

	const userService = new UserService();

	let email = $state('');
	let isLoading = $state(false);
	let error: string | undefined = $state(undefined);
	let success = $state(false);

	async function requestEmail() {
		isLoading = true;
		await userService
			.requestOneTimeAccessEmail(email, data.redirect)
			.then(() => (success = true))
			.catch((e) => (error = e.response?.data.error || 'An unknown error occurred'));

		isLoading = false;
	}
</script>

<svelte:head>
	<title>Email Login</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator {success} error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">Email Login</h1>
	{#if error}
		<p class="text-muted-foreground mt-2" in:fade>
			{error}. Please try again.
		</p>
		<div class="mt-10 flex w-full justify-stretch gap-2">
			<Button variant="secondary" class="w-full" href="/">Go back</Button>
			<Button class="w-full" onclick={() => (error = undefined)}>Try again</Button>
		</div>
	{:else if success}
		<p class="text-muted-foreground mt-2" in:fade>
			An email has been sent to the provided email, if it exists in the system.
		</p>
		<div class="mt-8 flex w-full justify-stretch gap-2">
			<Button variant="secondary" class="w-full" href={'/login/alternative' + page.url.search}
				>Go back</Button
			>
			<Button class="w-full" href={'/login/alternative/code' + page.url.search}>Enter code</Button>
		</div>
	{:else}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				requestEmail();
			}}
			class="w-full max-w-[450px]"
		>
			<p class="text-muted-foreground mt-2" in:fade>
				Enter your email address to receive an email with a login code.
			</p>
			<Input id="Email" class="mt-7" placeholder="Your email" bind:value={email} />
			<div class="mt-8 flex justify-stretch gap-2">
				<Button variant="secondary" class="w-full" href={'/login/alternative' + page.url.search}
					>Go back</Button
				>
				<Button class="w-full" type="submit" {isLoading}>Submit</Button>
			</div>
		</form>
	{/if}
</SignInWrapper>
