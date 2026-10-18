<script lang="ts">
	import AuthHeading from '#lib/components/auth/auth-heading.svelte';
	import AuthTile from '#lib/components/auth/auth-tile.svelte';
	import SignInWrapper from '#lib/components/login-wrapper.svelte';
	import SignupForm from '#lib/components/signup/signup-form.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { UserSignUp } from '#lib/types/user.type.ts';
	import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
	import { tryCatch } from '#lib/utils/try-catch-util.ts';
	import { goto } from '$app/navigation';

	const userService = new UserService();

	let error: string | undefined = $state();

	async function handleSignup(userData: UserSignUp) {
		error = undefined;
		const result = await tryCatch(userService.signupInitialUser(userData));

		if (result.error) {
			error = getAxiosErrorMessage(result.error);
			return;
		}

		await userStore.setUser(result.data);
		await goto('/signup/welcome');
	}
</script>

<svelte:head>
	<title>{m.auth_set_up_pocket_id()}</title>
</svelte:head>

<SignInWrapper>
	<AuthTile />
	<AuthHeading title={m.auth_set_up_pocket_id()} tone={error ? 'danger' : undefined}>
		{error ?? m.auth_setup_description()}
	</AuthHeading>
	<SignupForm callback={handleSignup} submitLabel={m.auth_create_admin_account()} />
</SignInWrapper>
