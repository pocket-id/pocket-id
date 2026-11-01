<script lang="ts">
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import { authPrimaryClass } from '#lib/components/auth/auth-util.ts';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import SignupForm from '#lib/components/signup/signup-form.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { UserSignUp } from '#lib/types/user.type.ts';
	import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { tryCatch } from '#lib/utils/try-catch-util.ts';
	import { goto } from '$app/navigation';

	let { data } = $props();
	const userService = new UserService();

	const mode = $appConfigStore.allowUserSignups;
	const closed = !data.token && mode !== 'open';

	let error: string | undefined = $state();

	async function handleSignup(userData: UserSignUp) {
		error = undefined;
		const result = await tryCatch(userService.signup({ ...userData, token: data.token }));

		if (result.error) {
			error = getAxiosErrorMessage(result.error);
			return;
		}

		await userStore.setUser(result.data);
		await goto('/');
	}
</script>

<svelte:head>
	<title>{m.signup()}</title>
</svelte:head>

<SignInWrapper back={closed ? undefined : '/login'}>
	<AuthTile />
	{#if closed}
		<AuthHeading
			title={mode === 'withToken' ? m.auth_signups_invite_only() : m.auth_signups_closed()}
		>
			{mode === 'withToken'
				? m.auth_signups_invite_only_description()
				: m.auth_signups_closed_description({ appName: $appConfigStore.appName })}
		</AuthHeading>
		<Button class={authPrimaryClass} href="/login">{m.auth_go_to_sign_in()}</Button>
	{:else}
		<AuthHeading
			title={data.token ? invite : m.auth_create_your_account({ appName: $appConfigStore.appName })}
			tone={error ? 'danger' : undefined}
		>
			{error ?? m.auth_signup_description()}
		</AuthHeading>
		<SignupForm callback={handleSignup} submitLabel={m.auth_create_account()} />
	{/if}
</SignInWrapper>

{#snippet invite()}
	<span class="text-control text-fg-muted block font-sans font-normal tracking-normal">
		{m.auth_invited_to()}
	</span>
	<span class="block">{$appConfigStore.appName}</span>
{/snippet}
