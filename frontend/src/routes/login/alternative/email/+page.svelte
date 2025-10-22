<script lang="ts">
	import { page } from '$app/state';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import Input from '$lib/components/ui/input/input.svelte';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import { fade } from 'svelte/transition';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';
	import { preventDefault } from '$lib/utils/event-util';
	import { afterNavigate } from '$app/navigation';

	const { data } = $props();

	const userService = new UserService();

	let email = $state('');
	let isLoading = $state(false);
	let error: string | undefined = $state(undefined);
	let success = $state(false);
	let backHref = $state('/login/alternative');

	// If the previous page is a Pocket ID page, go back there instead of the generic alternative login page
	afterNavigate((e) => {
		if (e.from?.url.pathname) {
			backHref = e.from.url.pathname + e.from.url.search;
		}
	});

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
		<div class="mt-10 flex justify-between gap-2">
			<Button variant="secondary" class="flex-1" href="/">{m.go_back()}</Button>
			<Button class="flex-1" onclick={() => (error = undefined)}>{m.try_again()}</Button>
		</div>
	{:else if success}
		<p class="text-muted-foreground mt-2" in:fade>
			{m.an_email_has_been_sent_to_the_provided_email_if_it_exists_in_the_system()}
		</p>
		<div class="mt-8 flex justify-between gap-2">
			<Button variant="secondary" class="flex-1" href={'/login/alternative' + page.url.search}
				>{m.go_back()}</Button
			>
			<Button class="flex-1" href={'/login/alternative/code' + page.url.search}
				>{m.enter_code()}</Button
			>
		</div>
	{:else}
		<form onsubmit={preventDefault(requestEmail)} class="w-full max-w-[450px]">
			<p class="text-muted-foreground mt-2" in:fade>
				{m.enter_your_email_address_to_receive_an_email_with_a_login_code()}
			</p>
			<Input id="Email" class="mt-7" placeholder={m.your_email()} bind:value={email} type="email" />
			<div class="mt-8 flex justify-between gap-2">
				<Button variant="secondary" class="flex-1" href={backHref}>{m.go_back()}</Button>
				<Button class="flex-1" type="submit" {isLoading}>{m.submit()}</Button>
			</div>
		</form>
	{/if}
</SignInWrapper>
