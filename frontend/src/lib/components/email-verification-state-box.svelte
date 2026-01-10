<script lang="ts">
	import { page } from '$app/state';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideAlertTriangle, LucideCheckCircle2, LucideCircleX } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { get } from 'svelte/store';

	const userService = new UserService();

	let emailVerificationState = $state(page.url.searchParams.get('emailVerificationState'));

	async function sendEmailVerification() {
		await userService
			.sendEmailVerification()
			.then(() => {
				toast.success(m.email_verification_sent());
			})
			.catch(axiosErrorToast);
	}

	function onDismiss() {
		const url = new URL(page.url);
		url.searchParams.delete('emailVerificationState');
		history.replaceState(null, '', url.toString());
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

{#if emailVerificationState}
	{#if emailVerificationState === 'success'}
		<Alert.Root variant="success" {onDismiss}>
			<LucideCheckCircle2 class="size-4" />
			<Alert.Title class="font-semibold">{m.email_verification_success_title()}</Alert.Title>
			<Alert.Description class="text-sm">
				{m.email_verification_success_description()}
			</Alert.Description>
		</Alert.Root>
	{:else}
		<Alert.Root variant="destructive" {onDismiss}>
			<LucideCircleX class="size-4" />
			<Alert.Title class="font-semibold">{m.email_verification_error_title()}</Alert.Title>
			<Alert.Description class="text-sm">
				{emailVerificationState}
			</Alert.Description>
		</Alert.Root>
	{/if}
{:else if $userStore && $appConfigStore.emailVerificationEnabled && !$userStore.emailVerified}
	<Alert.Root variant="warning" class="flex gap-3">
		<LucideAlertTriangle class="size-4" />
		<div class="md:flex md:w-full md:place-content-between">
			<div>
				<Alert.Title class="font-semibold">{m.email_verification_warning()}</Alert.Title>
				<Alert.Description class="text-sm">
					{m.email_verification_warning_description()}
				</Alert.Description>
			</div>
			<div>
				<Button class="mt-2 md:mt-0" usePromiseLoading onclick={sendEmailVerification}>
					{m.send_email()}
				</Button>
			</div>
		</div>
	</Alert.Root>
{/if}
