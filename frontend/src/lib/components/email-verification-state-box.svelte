<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import * as Alert from '@pocket-id/ui/alert';
	import { m } from '#lib/paraglide/messages.js';
	import userStore from '#lib/stores/user-store.ts';
	import CircleCheckIcon from '~icons/solar/check-circle-bold-duotone';
	import CircleXIcon from '~icons/solar/close-circle-bold-duotone';
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';

	let emailVerificationState = $state(page.url.searchParams.get('emailVerificationState'));

	async function onDismiss() {
		const url = new URL(page.url.href);
		url.searchParams.delete('emailVerificationState');
		await goto(url, { shallow: true, replace: true, state: page.state });
		emailVerificationState = null;
	}

	onMount(() => {
		const user = get(userStore);
		if (emailVerificationState === 'success' && user) {
			user.emailVerified = true;
			userStore.setUser(user);
		}
	});
</script>

{#if emailVerificationState === 'success'}
	<Alert.Root variant="success" {onDismiss}>
		<CircleCheckIcon />
		<Alert.Title class="wrap-anywhere"
			>{m.email_verified_title({ email: $userStore?.email ?? '' })}</Alert.Title
		>
	</Alert.Root>
{:else if emailVerificationState}
	<Alert.Root variant="destructive" {onDismiss}>
		<CircleXIcon />
		<Alert.Title>{m.email_verification_failed_title()}</Alert.Title>
		<Alert.Description>{emailVerificationState}</Alert.Description>
	</Alert.Root>
{/if}
