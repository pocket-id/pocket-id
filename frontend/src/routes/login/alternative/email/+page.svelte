<script lang="ts">
	import { page } from '$app/state';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import Input from '$lib/components/ui/input/input.svelte';
	import UserService from '$lib/services/user-service';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';
	import { m } from '$lib/paraglide/messages';

	const { data } = $props();

	const userService = new UserService();

	let email = $state('');
	let isLoading = $state(false);
	let error: string | undefined = $state(undefined);
	let success = $state(false);

	async function requestEmail() {
		isLoading = true;
		await userService
			.requestOneTimeAccessEmailAsUnauthenticatedUser(email, data.redirect)
			.then(() => (success = true))
			.catch((e) => (error = e.response?.data.error || m.an_unknown_error_occurred()));

		isLoading = false;
	}
</script>

<svelte:head>
	<title>{m.email_login()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator {success} error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-3xl font-bold sm:text-4xl">{m.email_login()}</h1>
	{#if error}
		<p class="text-muted-foreground mt-2" in:fade>
			{error}. {m.please_try_again()}
		</p>
		<div class="mt-10 flex w-full justify-stretch gap-2">
			<Button variant="secondary" class="w-full" href="/">{m.go_back()}</Button>
			<Button class="w-full" onclick={() => (error = undefined)}>{m.try_again()}</Button>
		</div>
	{:else if success}
		<p class="text-muted-foreground mt-2" in:fade>
			{m.an_email_has_been_sent_to_the_provided_email_if_it_exists_in_the_system()}
		</p>
		<div class="mt-8 flex w-full justify-stretch gap-2">
			<Button variant="secondary" class="w-full" href={'/login/alternative' + page.url.search}
				>{m.go_back()}</Button
			>
			<Button class="w-full" href={'/login/alternative/code' + page.url.search}>{m.enter_code()}</Button>
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
				{m.enter_your_email_address_to_receive_an_email_with_a_login_code()}
			</p>
			<Input id="Email" class="mt-7" placeholder={m.your_email()} bind:value={email} />
			<div class="mt-8 flex justify-stretch gap-2">
				<Button variant="secondary" class="w-full" href={'/login/alternative' + page.url.search}
					>{m.go_back()}</Button
				>
				<Button class="w-full" type="submit" {isLoading}>{m.submit()}</Button>
			</div>
		</form>
	{/if}
</SignInWrapper>
