<script lang="ts">
	import { goto } from '$app/navigation';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import {
		authPrimaryClass,
		authQuietClass,
		emailedLoginCode,
		successHold,
		takingYouMessage
	} from '#lib/components/auth/auth-util.ts';
	import CodeGroup from '#lib/components/auth/code-group.svelte';
	import CodeLine from '#lib/components/auth/code-line.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { Label } from '#lib/components/ui/label/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.js';
	import { getAxiosErrorCode, getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { onMount, tick, untrack } from 'svelte';

	let { data } = $props();

	const SHORT = 6;
	const LONG = 12;
	const emailCodes = $appConfigStore.emailOneTimeAccessAsUnauthenticatedEnabled;
	const userService = new UserService();

	const sentTo = data.code ? '' : emailedLoginCode.sentTo;
	const redirectQuery =
		data.redirect === '/settings' ? '' : `?redirect=${encodeURIComponent(data.redirect)}`;

	let code = $state(data.code ?? '');
	let long = $state(!sentTo && (code ? code.length > SHORT : !emailCodes));
	let resendIn = $state(30);
	let resending = $state(false);
	let checking = $state(false);
	let success = $state(false);
	let error: { message: string; invalid: boolean } | undefined = $state();

	const token = $derived(code.replace(/[\s-]/g, ''));
	const complete = $derived(token.length >= (long ? LONG : SHORT));

	// Typing again clears the last outcome
	$effect(() => {
		void code;
		untrack(() => (error = undefined));
	});

	async function signIn() {
		if (checking) return;
		checking = true;
		error = undefined;
		try {
			const user = await userService.exchangeOneTimeAccessToken(token);
			await userStore.setUser(user);
			success = true;
			await successHold();
			await goto(data.redirect).catch(() => {
				success = false;
				error = { message: m.invalid_redirect_url(), invalid: false };
			});
		} catch (e) {
			const invalid = getAxiosErrorCode(e) === 'token_invalid_or_expired';
			error = { message: invalid ? m.auth_code_did_not_work() : getAxiosErrorMessage(e), invalid };
		}
		checking = false;
	}

	// The last outcome belonged to the other format, and the new field takes over the typing
	async function switchFormat() {
		long = !long;
		error = undefined;
		if (token.length > (long ? LONG : SHORT)) code = '';
		await tick();
		document.getElementById('login-code')?.focus();
	}

	// Resend code becomes available once the countdown under the primary runs out
	$effect(() => {
		if (!sentTo) return;
		const timer = setInterval(() => resendIn > 0 && resendIn--, 1000);
		return () => clearInterval(timer);
	});

	async function resend() {
		resending = true;
		try {
			await userService.requestOneTimeAccessEmailAsUnauthenticatedUser(sentTo, data.redirect);
			resendIn = 30;
		} catch (e) {
			error = { message: getAxiosErrorMessage(e), invalid: false };
		}
		resending = false;
		await tick();
		document.getElementById('login-code')?.focus();
	}

	onMount(() => {
		if (code) signIn();
	});
</script>

<svelte:head>
	<title>{m.auth_enter_your_login_code()}</title>
</svelte:head>

<SignInWrapper back={(sentTo ? '/login/alternative/email' : '/login/alternative') + redirectQuery}>
	<AuthTile {success} />
	<AuthHeading
		title={success
			? m.auth_signed_in()
			: sentTo
				? m.auth_check_your_email()
				: m.auth_enter_your_login_code()}
	>
		{#if success}
			{takingYouMessage(data.redirect)}
		{:else if sentTo}
			<span class="[&_b]:text-fg [&_b]:font-medium">
				<FormattedMessage
					message={m.auth_email_code_sent}
					inputs={{ email: sentTo, appName: $appConfigStore.appName }}
				/>
			</span>
		{:else if long}
			{m.auth_login_code_long_description()}
		{:else}
			{m.auth_login_code_short_description()}
		{/if}
	</AuthHeading>
	<form
		onsubmit={preventDefault(signIn)}
		class="flex flex-col transition-opacity duration-(--dur-fast)"
		class:opacity-0={success}
		inert={success}
	>
		<Label for="login-code" class="mb-2">{m.auth_login_code()}</Label>
		{#if long}
			<Input
				id="login-code"
				bind:value={code}
				class="h-(--h-xl) font-mono tracking-[0.12em]"
				autocomplete="one-time-code"
				autocapitalize="off"
				spellcheck={false}
				readonly={checking}
				aria-invalid={error?.invalid || undefined}
				aria-describedby="login-code-line"
				autofocus
			/>
		{:else}
			<CodeGroup
				id="login-code"
				groups={[3, 3]}
				bind:value={code}
				invalid={error?.invalid}
				readonly={checking}
				onComplete={signIn}
				autocomplete="one-time-code"
				autofocus
				aria-describedby="login-code-line"
			/>
		{/if}
		<CodeLine id="login-code-line" tone={checking ? 'checking' : error ? 'error' : undefined}>
			{#if checking}
				{m.auth_checking_code()}
			{:else if error}
				{error.message}
			{:else if long}
				{m.auth_spaces_and_dashes_are_ignored()}
			{/if}
		</CodeLine>
		<div class="mt-6 flex flex-col items-center gap-3">
			<Button type="submit" class={authPrimaryClass} disabled={!complete}>
				{m.sign_in()}
			</Button>
			{#if sentTo}
				{#if resendIn > 0}
					<p class="text-control text-fg-muted flex h-(--h-xl) items-center tabular-nums">
						{m.auth_resend_code_in({
							time: `${Math.floor(resendIn / 60)}:${String(resendIn % 60).padStart(2, '0')}`
						})}
					</p>
				{:else}
					<Button variant="link" class={authQuietClass} isLoading={resending} onclick={resend}>
						{m.auth_resend_code()}
					</Button>
				{/if}
			{:else if emailCodes}
				<Button variant="link" class={authQuietClass} disabled={checking} onclick={switchFormat}>
					{long ? m.auth_i_have_a_6_character_code() : m.auth_i_have_a_12_character_code()}
				</Button>
			{/if}
		</div>
	</form>
</SignInWrapper>
