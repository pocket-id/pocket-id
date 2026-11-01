<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import userStore from '#lib/stores/user-store.ts';
	import { isEmailVerificationPending } from '#lib/utils/email-verification-util.ts';
	import { LucideCircleCheck, LucideCircleX, LucideTriangleAlert } from '@lucide/svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';

	let emailVerificationState = $state(page.url.searchParams.get('emailVerificationState'));

	const emailVerificationRequired = $derived(
		isEmailVerificationPending($userStore, $appConfigStore)
	);

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
		<LucideCircleCheck />
		<Alert.Title class="wrap-anywhere"
			>{m.email_verified_title({ email: $userStore?.email ?? '' })}</Alert.Title
		>
	</Alert.Root>
{:else if emailVerificationState}
	<Alert.Root variant="destructive" {onDismiss}>
		<LucideCircleX />
		<Alert.Title>{m.email_verification_failed_title()}</Alert.Title>
		<Alert.Description>{emailVerificationState}</Alert.Description>
	</Alert.Root>
{:else if $userStore && $appConfigStore.emailVerification !== 'disabled' && !$userStore.emailVerified && (emailVerificationRequired || $userStore.email)}
	<Alert.Root variant={emailVerificationRequired ? 'destructive' : 'warning'}>
		<LucideTriangleAlert />
		{#if emailVerificationRequired}
			<Alert.Title>{m.email_verification_required_warning()}</Alert.Title>
			<Alert.Description>
				{m.email_verification_only_page()}
				{#if $userStore.emailVerificationDeadline}
					<strong class="font-semibold">
						{m.email_verification_deadline_description({
							time: formatDistanceToNow($userStore.emailVerificationDeadline, { addSuffix: true })
						})}
					</strong>
				{/if}
			</Alert.Description>
		{:else}
			<Alert.Title>{m.email_verification_warning()}</Alert.Title>
			<Alert.Description class="wrap-anywhere">
				{m.email_verification_send_hint({ email: $userStore.email ?? '' })}
			</Alert.Description>
		{/if}
	</Alert.Root>
{/if}
