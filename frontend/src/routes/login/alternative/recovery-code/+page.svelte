<script lang="ts">
	import { goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import Input from '$lib/components/ui/input/input.svelte';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import userStore from '$lib/stores/user-store.js';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { preventDefault } from '$lib/utils/event-util';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';

	let { data } = $props();
	let code = $state('');
	let isLoading = $state(false);
	let error: string | undefined = $state();

	const userService = new UserService();

	async function authenticate() {
		if (!code.trim()) return;
		isLoading = true;
		error = undefined;
		try {
			const user = await userService.redeemRecoveryCode(code.trim());
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
</script>

<svelte:head>
	<title>{m.recovery_codes_title()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">{m.recovery_codes_title()}</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. {m.please_try_again()}
		</p>
	{:else}
		<p class="text-muted-foreground mt-2">{m.enter_a_recovery_code_to_sign_in()}</p>
	{/if}
	<form onsubmit={preventDefault(authenticate)} class="w-full max-w-[450px]">
		<Input
			id="recovery-code"
			class="mt-7 font-mono tracking-wider"
			placeholder="xxxx-xxxx-xxxx-xxxx"
			aria-label={m.recovery_code_input_label()}
			autocomplete="one-time-code"
			spellcheck={false}
			autocapitalize="none"
			bind:value={code}
			type="text"
		/>
		<p class="text-muted-foreground mt-2 text-xs">
			{m.recovery_code_input_help()}
		</p>
		<div class="mt-8 flex justify-between gap-2">
			<Button variant="secondary" class="flex-1" href="/login/alternative">{m.go_back()}</Button>
			<Button class="flex-1" type="submit" {isLoading}>{m.submit()}</Button>
		</div>
	</form>
</SignInWrapper>
