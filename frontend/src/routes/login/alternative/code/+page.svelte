<script lang="ts">
	import { afterNavigate, goto } from '$app/navigation';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as InputOTP from '$lib/components/ui/input-otp/index.js';
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
	let backHref = $state('/login/alternative');

	let longCodeRequested = $state(code.length > 6);
	let showLongCodeOption = $state(true);
	let codeComplete = $derived(longCodeRequested ? code.length === 16 : code.length === 6);
	let interactionSession = data.interactionSession;

	const userService = new UserService();
	const clientID = interactionSession?.client?.id ?? '';

	// If the previous page is a Pocket ID page, go back there instead of the generic alternative login page
	afterNavigate((e) => {
		if (e.from?.url.pathname) {
			backHref = e.from.url.pathname + e.from.url.search;
		}
	});

	async function authenticate(mode: 'normal' | 'incognito') {
		if (!code?.trim()) return;
		if (!codeComplete) return;
		isLoading = true;
		try {
			const payload: { token: string; permittedClient?: string } = {
				token: code
			};

			if (mode === 'incognito') {
				payload.permittedClient = clientID;
			}

			const user = await userService.exchangeOneTimeAccessToken(payload);
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
			authenticate('normal');
		}

		if (data.redirect.startsWith('/interaction')) {
			showLongCodeOption = false;
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
	<h1 class="font-gloock mt-5 text-4xl font-bold">{m.login_code()}</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. {m.please_try_again()}
		</p>
	{:else}
		<p class="text-muted-foreground mt-2">{m.enter_the_code_you_received_to_sign_in()}</p>
	{/if}
	<form
		onsubmit={preventDefault(() => authenticate('normal'))}
		class="flex w-full flex-col items-center mt-8"
	>
		<div class="flex flex-col w-full justify-center items-center">
			{#if longCodeRequested}
				<Input
					id="Code"
					class="w-[80%]"
					placeholder={m.code()}
					aria-label={m.code()}
					bind:value={code}
					type="text"
				/>
			{:else}
				<InputOTP.Root maxlength={6} bind:value={code}>
					{#snippet children({ cells })}
						<InputOTP.Group>
							{#each cells as cell}
								<InputOTP.Slot {cell} />
							{/each}
						</InputOTP.Group>
					{/snippet}
				</InputOTP.Root>
			{/if}
			{#if !longCodeRequested && showLongCodeOption}
				<div class="flex justify-center">
					<Button
						class="mt-2 text-muted-foreground text-xs"
						size="sm"
						variant="ghost"
						type="button"
						onclick={() => (longCodeRequested = true)}
					>
						{m.i_have_a_longer_code()}
					</Button>
				</div>
			{/if}
		</div>
		<div class="w-full max-w-[450px] flex gap-2 pt-7">
			<Button class="flex-1" variant="secondary" href={backHref}>
				{m.go_back()}
			</Button>
			<Button
				class="flex-1"
				disabled={!codeComplete}
				{isLoading}
				onclick={() => authenticate('normal')}
			>
				{m.submit()}
			</Button>
			{#if clientID}
				<Button
					class="flex-1 bg-purple-600 hover:bg-purple-700 "
					disabled={!codeComplete}
					{isLoading}
					onclick={() => authenticate('incognito')}
				>
					{m.incognito()}
				</Button>
			{/if}
		</div>
	</form>
</SignInWrapper>
