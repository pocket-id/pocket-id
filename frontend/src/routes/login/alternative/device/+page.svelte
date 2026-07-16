<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import Qrcode from '$lib/components/qrcode/qrcode.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Separator } from '$lib/components/ui/separator';
	import { Spinner } from '$lib/components/ui/spinner';
	import { m } from '$lib/paraglide/messages';
	import DeviceLoginService from '$lib/services/device-login-service';
	import userStore from '$lib/stores/user-store';
	import type { DeviceLoginRequest } from '$lib/types/device-login.type';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { mode } from 'mode-watcher';
	import { onMount } from 'svelte';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const deviceLoginService = new DeviceLoginService();

	let request: DeviceLoginRequest | undefined = $state();
	let errorMessage: string | null = $state(null);
	let isStarting = $state(true);
	let pollTimer: ReturnType<typeof setTimeout> | undefined;

	onMount(() => {
		startRequest();

		return () => {
			clearTimers();
		};
	});

	async function startRequest() {
		clearTimers();
		request = undefined;
		errorMessage = null;
		isStarting = true;

		try {
			request = await deviceLoginService.createRequest();
			schedulePoll();
		} catch (error) {
			errorMessage = getAxiosErrorMessage(error);
		} finally {
			isStarting = false;
		}
	}

	function schedulePoll() {
		if (!request) return;
		pollTimer = setTimeout(exchangeRequest, request.interval * 1000);
	}

	async function exchangeRequest() {
		if (!request) return;

		try {
			const user = await deviceLoginService.exchangeRequest(request.id);
			if (!user) {
				schedulePoll();
				return;
			}

			clearTimers();
			await userStore.setUser(user);
			await goto(data.redirect);
		} catch (error) {
			clearTimers();
			errorMessage = getAxiosErrorMessage(error);
		}
	}

	function clearTimers() {
		if (pollTimer) clearTimeout(pollTimer);
		pollTimer = undefined;
	}
</script>

<svelte:head>
	<title>{m.sign_in_with_another_device()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator error={!!errorMessage} />
	</div>
	<h1 class="font-gloock mt-5 text-2xl font-bold sm:text-4xl">
		{m.sign_in_with_another_device()}
	</h1>
	<p class="text-muted-foreground mt-2">
		{errorMessage ? errorMessage : m.sign_in_with_another_device_description()}
	</p>

	{#if isStarting}
		<div class="mt-10 flex items-center gap-2 text-sm">
			<Spinner />
			{m.creating_device_login_request()}
		</div>
	{:else if request && !errorMessage}
		<Card.Root class="mt-8 w-full max-w-sm shrink-0">
			<Card.Content class="flex flex-col items-center gap-5">
				<Qrcode
					value={request.verificationUriComplete}
					color={mode.current === 'dark' ? '#FFFFFF' : '#000000'}
					aria-label={m.device_login_qr_code()}
					class="h-[12dvh]"
				/>
				<div class="flex w-full items-center gap-3">
					<Separator class="flex-1" />
					<span class="text-muted-foreground text-xs">{m.or()}</span>
					<Separator class="flex-1" />
				</div>
				<div>
					<p class="text-muted-foreground text-sm mb-2">
						{m.visit_and_enter({ url: request.verificationUri })}
					</p>
					<CopyToClipboard value={request.userCode}>
						<p class="text-xl sm:text-2xl font-bold tracking-wider" data-testid="device-login-code">
							{request.userCode.substring(0, 4)}
							<span class="text-muted-foreground font-normal">-</span>
							{request.userCode.substring(4, 8)}
						</p>
					</CopyToClipboard>
				</div>
			</Card.Content>
		</Card.Root>
	{/if}
	<div class="flex mt-7 md:mt-15 gap-3 w-full max-w-112.5">
		<Button class="flex-1" href={'/login/alternative' + page.url.search} variant="secondary"
			>{m.go_back()}</Button
		>
		<Button class="flex-1" isLoading={!errorMessage} onclick={startRequest}>
			{errorMessage ? m.try_again() : m.waiting_for_approval()}
		</Button>
	</div>
</SignInWrapper>
