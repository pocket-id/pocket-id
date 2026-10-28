<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass, emailedLoginCode } from '#lib/components/auth/auth-util.ts';
	import CodeLine from '#lib/components/auth/code-line.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { Label } from '#lib/components/ui/label/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { z } from 'zod/v4';

	const { data } = $props();

	const userService = new UserService();
	const { inputs, ...form } = createForm(
		z.object({ email: z.email(m.auth_enter_a_full_email_address()) }),
		{ email: emailedLoginCode.sentTo }
	);

	let isLoading = $state(false);
	let error: string | undefined = $state();

	async function requestEmail() {
		const values = form.validate();
		if (!values) return;
		isLoading = true;
		error = undefined;
		try {
			await userService.requestOneTimeAccessEmailAsUnauthenticatedUser(values.email, data.redirect);
			emailedLoginCode.sentTo = values.email;
			await goto('/login/alternative/code' + page.url.search);
		} catch (e) {
			error = getAxiosErrorMessage(e);
		}
		isLoading = false;
	}
</script>

<svelte:head>
	<title>{m.auth_get_a_login_code_by_email()}</title>
</svelte:head>

<SignInWrapper back={'/login/alternative' + page.url.search}>
	<AuthTile />
	<AuthHeading title={m.auth_get_a_login_code_by_email()} tone={error ? 'danger' : undefined}>
		{error ?? m.auth_email_login_description({ appName: $appConfigStore.appName })}
	</AuthHeading>
	<form onsubmit={preventDefault(requestEmail)} class="flex flex-col" novalidate>
		<Label for="email" class="mb-2">{m.email_address()}</Label>
		<Input
			id="email"
			type="email"
			autocomplete="email"
			class="h-(--h-xl)"
			bind:value={$inputs.email.value}
			oninput={() => ($inputs.email.error = null)}
			aria-invalid={!!$inputs.email.error}
			aria-describedby="email-line"
			autofocus
		/>
		<CodeLine id="email-line" tone={$inputs.email.error ? 'error' : undefined}>
			{$inputs.email.error ?? m.auth_email_login_helper()}
		</CodeLine>
		<Button type="submit" class={authPrimaryClass + ' mt-6'} {isLoading}>
			{error ? m.try_again() : m.auth_email_me_a_code()}
		</Button>
	</form>
</SignInWrapper>
