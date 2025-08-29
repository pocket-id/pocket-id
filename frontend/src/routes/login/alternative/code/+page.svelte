<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import Input from '$lib/components/ui/input/input.svelte';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import userStore from '$lib/stores/user-store.js';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import { onMount } from 'svelte';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';

	let { data } = $props();
	let code = $state(data.code ?? '');
	let isLoading = $state(false);
	let error: string | undefined = $state();

	const userService = new UserService();

	async function authenticate() {
		isLoading = true;
		try {
			const user = await userService.exchangeOneTimeAccessToken(code);
			await userStore.setUser(user);

			try {
				goto(data.redirect);
			} catch (e) {
				error = m.invalid_redirect_url();
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
	<title>{m.login_code()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">{m.login_code()}</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. {m.please_try_again()}
		</p>
	{:else}
		<p class="text-muted-foreground mt-2">{m.enter_the_code_you_received_to_sign_in()}</p>
	{/if}
	<form onsubmit={preventDefault(authenticate)} class="w-full max-w-[450px]">
		<Input id="Code" class="mt-7" placeholder={m.code()} bind:value={code} type="text" />
		<div class="mt-8 flex justify-between gap-2">
			<Button variant="secondary" class="flex-1" href={'/login/alternative' + page.url.search}
				>{m.go_back()}</Button
			>
			<Button class="flex-1" type="submit" {isLoading}>{m.submit()}</Button>
		</div>
	</form>
</SignInWrapper>
